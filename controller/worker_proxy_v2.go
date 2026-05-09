package controller

import (
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	relayopenai "github.com/songquanpeng/one-api/relay/adaptor/openai"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	relaycontroller "github.com/songquanpeng/one-api/relay/controller"
	relayvalidator "github.com/songquanpeng/one-api/relay/controller/validator"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

const workerRelayPathHeaderV2 = "X-Alfred-Relay-Path"

type WorkerPrepareResponseV2 struct {
	ChannelId       int               `json:"channel_id"`
	ChannelKeyId    int               `json:"channel_key_id"`
	ChannelKeyIndex int               `json:"channel_key_index"`
	TargetURL       string            `json:"target_url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	ActualModel     string            `json:"actual_model"`
	UserId          int               `json:"user_id"`
	TokenId         int               `json:"token_id"`
	TokenName       string            `json:"token_name"`
	IsStream        bool              `json:"is_stream"`
}

type WorkerCallbackRequestV2 struct {
	UserId           int    `json:"user_id"`
	TokenId          int    `json:"token_id"`
	TokenName        string `json:"token_name"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	ChannelId        int    `json:"channel_id"`
	ChannelKeyId     int    `json:"channel_key_id"`
	ChannelKeyIndex  int    `json:"channel_key_index"`
	ElapsedTime      int64  `json:"elapsed_time"`
	IsStream         bool   `json:"is_stream"`
	IsError          bool   `json:"is_error"`
	ErrorMsg         string `json:"error_msg"`
	ErrorType        string `json:"error_type"`
	ErrorCode        string `json:"error_code"`
	StatusCode       int    `json:"status_code"`
	RequestID        string `json:"request_id"`
}

func WorkerPrepareV2(c *gin.Context) {
	relayPath := c.GetHeader(workerRelayPathHeaderV2)
	relayMode, bizErr := parseWorkerRelayModeV2(relayPath)
	if bizErr != nil {
		renderWorkerErrorV2(c, bizErr)
		return
	}

	requestModel, bizErr := parseWorkerRequestModelV2(c, relayMode)
	if bizErr != nil {
		renderWorkerErrorV2(c, bizErr)
		return
	}
	c.Set(ctxkey.RequestModel, requestModel)

	if !isWorkerModelAllowedV2(c, requestModel) {
		renderWorkerErrorV2(c, &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusForbidden,
			Error: relaymodel.Error{
				Message: "该令牌无权使用模型：" + requestModel,
				Type:    "invalid_request_error",
				Code:    "model_not_allowed",
			},
		})
		return
	}

	channel, channelKey, bizErr := selectWorkerChannelWithKeyV2(c, requestModel)
	if bizErr != nil {
		renderWorkerErrorV2(c, bizErr)
		return
	}

	middleware.SetupContextForSelectedChannelWithKey(c, channel, channelKey, requestModel)
	prepared, bizErr := relaycontroller.BuildPreparedTextRequestV2(c, relayPath)
	if bizErr != nil {
		renderWorkerErrorV2(c, bizErr)
		return
	}

	c.JSON(http.StatusOK, WorkerPrepareResponseV2{
		ChannelId:       channel.Id,
		ChannelKeyId:    channelKey.Id,
		ChannelKeyIndex: channelKey.Position + 1,
		TargetURL:       prepared.TargetURL,
		Method:          http.MethodPost,
		Headers:         prepared.Headers,
		Body:            string(prepared.Body),
		ActualModel:     prepared.ActualModelName,
		UserId:          c.GetInt(ctxkey.Id),
		TokenId:         c.GetInt(ctxkey.TokenId),
		TokenName:       c.GetString(ctxkey.TokenName),
		IsStream:        prepared.IsStream,
	})
}

func WorkerCallbackV2(c *gin.Context) {
	var req WorkerCallbackRequestV2
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	channel, err := model.GetChannelById(req.ChannelId, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	userGroup, _ := model.CacheGetUserGroup(req.UserId)
	modelRatio := billingratio.GetModelRatio(req.Model, channel.Type)
	groupRatio := billingratio.GetGroupRatio(userGroup)
	completionRatio := billingratio.GetCompletionRatio(req.Model, channel.Type)
	ratioVal := modelRatio * groupRatio

	quota := int64((float64(req.PromptTokens) + float64(req.CompletionTokens)*completionRatio) * ratioVal)
	if ratioVal != 0 && quota <= 0 {
		quota = 1
	}

	if err = model.CacheDecreaseUserQuota(req.UserId, quota); err != nil {
		logger.SysError("failed to decrease user quota for edge proxy v2: " + err.Error())
	}
	if err = model.PostConsumeTokenQuota(req.TokenId, quota); err != nil {
		logger.SysError("failed to post consume token quota for edge proxy v2: " + err.Error())
	}
	if err = model.CacheUpdateUserQuota(c.Request.Context(), req.UserId); err != nil {
		logger.SysError("failed to update user quota cache for edge proxy v2: " + err.Error())
	}

	if req.IsError {
		if err = model.MarkChannelKeyFailureV2(req.ChannelId, req.ChannelKeyId, req.StatusCode, req.ErrorMsg); err != nil {
			logger.Errorf(c.Request.Context(), "failed to update channel key failure state: %s", err.Error())
		}
		relayErr := buildWorkerRelayErrorV2(req)
		go processChannelRelayError(c.Request.Context(), req.UserId, req.ChannelId, channel.Name, relayErr)
		logger.SysError("[" + "Edge Proxy V2" + "] 渠道 #" + strconv.Itoa(req.ChannelId) + " 运行错误: " + req.ErrorMsg)
		model.RecordLogWithChannel(
			c.Request.Context(),
			req.UserId,
			model.LogTypeSystem,
			"[Edge Proxy V2] 渠道运行错误: "+req.ErrorMsg,
			req.ChannelId,
			req.ChannelKeyId,
		)
	} else {
		monitor.Emit(req.ChannelId, true)
		monitor.ClearChannelFailures(req.ChannelId)
		if err = model.MarkChannelKeySuccessV2(req.ChannelId, req.ChannelKeyId); err != nil {
			logger.Errorf(c.Request.Context(), "failed to update channel key success state: %s", err.Error())
		}
	}

	logContent := "Edge Proxy V2 倍率：" + formatFloatV2(modelRatio) + " × " + formatFloatV2(groupRatio) + " × " + formatFloatV2(completionRatio)
	if req.IsError {
		logContent += " (包含运行错误：" + req.ErrorMsg + ")"
	}
	model.RecordConsumeLog(c.Request.Context(), &model.Log{
		UserId:           req.UserId,
		ChannelId:        req.ChannelId,
		ChannelKeyId:     req.ChannelKeyId,
		ChannelKeyIndex:  req.ChannelKeyIndex,
		PromptTokens:     req.PromptTokens,
		CompletionTokens: req.CompletionTokens,
		ModelName:        req.Model,
		TokenName:        req.TokenName,
		Quota:            int(quota),
		Content:          logContent,
		IsStream:         req.IsStream,
		ElapsedTime:      req.ElapsedTime,
	})

	model.UpdateUserUsedQuotaAndRequestCount(req.UserId, quota)
	model.UpdateChannelUsedQuota(req.ChannelId, quota)

	c.JSON(http.StatusOK, gin.H{"status": "success", "quota": quota})
}

func parseWorkerRelayModeV2(relayPath string) (int, *relaymodel.ErrorWithStatusCode) {
	if relayPath == "" {
		return relaymode.Unknown, &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusBadRequest,
			Error: relaymodel.Error{
				Message: "missing worker relay path",
				Type:    "invalid_request_error",
				Code:    "missing_worker_relay_path",
			},
		}
	}

	parsedRelayURL, err := neturl.ParseRequestURI(relayPath)
	if err != nil {
		return relaymode.Unknown, relayopenai.ErrorWrapper(err, "invalid_worker_relay_path", http.StatusBadRequest)
	}

	relayMode := relaymode.GetByPath(parsedRelayURL.Path)
	if !isWorkerSupportedRelayModeV2(relayMode) {
		return relaymode.Unknown, &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusForbidden,
			Error: relaymodel.Error{
				Message: "worker prepare v2 does not support this relay path",
				Type:    "invalid_request_error",
				Code:    "worker_prepare_mode_not_supported",
			},
		}
	}
	return relayMode, nil
}

func parseWorkerRequestModelV2(c *gin.Context, relayMode int) (string, *relaymodel.ErrorWithStatusCode) {
	textRequest := &relaymodel.GeneralOpenAIRequest{}
	if err := common.UnmarshalBodyReusable(c, textRequest); err != nil {
		return "", relayopenai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
	}
	if relayMode == relaymode.Moderations && textRequest.Model == "" {
		textRequest.Model = "text-moderation-latest"
	}
	if err := relayvalidator.ValidateTextRequest(textRequest, relayMode); err != nil {
		return "", relayopenai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
	}
	return textRequest.Model, nil
}

func isWorkerSupportedRelayModeV2(relayMode int) bool {
	switch relayMode {
	case relaymode.ChatCompletions, relaymode.Completions, relaymode.Embeddings, relaymode.Moderations:
		return true
	default:
		return false
	}
}

func isWorkerModelAllowedV2(c *gin.Context, requestModel string) bool {
	if requestModel == "" {
		return true
	}
	availableModels := c.GetString(ctxkey.AvailableModels)
	if availableModels == "" {
		return true
	}
	modelList := strings.Split(availableModels, ",")
	for _, modelName := range modelList {
		if requestModel == strings.TrimSpace(modelName) {
			return true
		}
	}
	return false
}

func selectWorkerChannelWithKeyV2(c *gin.Context, requestModel string) (*model.Channel, *model.ChannelKey, *relaymodel.ErrorWithStatusCode) {
	userId := c.GetInt(ctxkey.Id)
	userGroup, _ := model.CacheGetUserGroup(userId)
	c.Set(ctxkey.Group, userGroup)

	var channel *model.Channel
	if specificChannelId, ok := c.Get(ctxkey.SpecificChannelId); ok {
		channelID, err := strconv.Atoi(specificChannelId.(string))
		if err != nil {
			return nil, nil, relayopenai.ErrorWrapper(err, "invalid_channel_id", http.StatusBadRequest)
		}
		channel, err = model.GetChannelById(channelID, true)
		if err != nil {
			return nil, nil, relayopenai.ErrorWrapper(err, "invalid_channel_id", http.StatusBadRequest)
		}
		if channel.Status != model.ChannelStatusEnabled {
			return nil, nil, &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusForbidden,
				Error: relaymodel.Error{
					Message: "channel is disabled",
					Type:    "invalid_request_error",
					Code:    "channel_disabled",
				},
			}
		}
	} else {
		var err error
		channel, err = model.CacheGetRandomSatisfiedChannel(userGroup, requestModel, false)
		if err != nil {
			return nil, nil, &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusServiceUnavailable,
				Error: relaymodel.Error{
					Message: "no available channel",
					Type:    "server_error",
					Code:    "channel_not_found",
				},
			}
		}
	}

	cfg, _ := channel.LoadConfig()
	if !cfg.EdgeProxy {
		return nil, nil, &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusForbidden,
			Error: relaymodel.Error{
				Message: "channel does not support edge proxy",
				Type:    "invalid_request_error",
				Code:    "edge_proxy_disabled",
			},
		}
	}

	keys, err := model.PrepareChannelKeyCandidatesV2(channel)
	if err != nil || len(keys) == 0 {
		if err == nil {
			err = http.ErrNoCookie
		}
		return nil, nil, relayopenai.ErrorWrapper(err, "no_keys_available_for_channel", http.StatusServiceUnavailable)
	}

	return channel, &keys[0], nil
}

func renderWorkerErrorV2(c *gin.Context, bizErr *relaymodel.ErrorWithStatusCode) {
	if bizErr == nil {
		bizErr = &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusBadGateway,
			Error: relaymodel.Error{
				Message: "worker prepare failed without detailed error",
				Type:    "upstream_error",
				Code:    "worker_prepare_failed",
			},
		}
	}
	bizErr.Error.Message = helper.MessageWithRequestId(bizErr.Error.Message, c.GetString(helper.RequestIdKey))
	c.JSON(bizErr.StatusCode, gin.H{"error": bizErr.Error})
}

func buildWorkerRelayErrorV2(req WorkerCallbackRequestV2) relaymodel.ErrorWithStatusCode {
	message := appendWorkerRequestIDV2(strings.TrimSpace(req.ErrorMsg), req.RequestID)
	if message == "" {
		message = "bad response status code " + strconv.Itoa(req.StatusCode)
	}
	errType := strings.TrimSpace(req.ErrorType)
	if errType == "" {
		errType = "upstream_error"
	}
	errCode := strings.TrimSpace(req.ErrorCode)
	if errCode == "" {
		errCode = "bad_response_status_code"
	}

	return relaymodel.ErrorWithStatusCode{
		StatusCode: req.StatusCode,
		Error: relaymodel.Error{
			Message: message,
			Type:    errType,
			Code:    errCode,
			Param:   strconv.Itoa(req.StatusCode),
		},
	}
}

func appendWorkerRequestIDV2(message string, requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return message
	}
	if strings.Contains(message, requestID) {
		return message
	}
	if message == "" {
		return "(request id: " + requestID + ")"
	}
	return message + " (request id: " + requestID + ")"
}

func formatFloatV2(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}
