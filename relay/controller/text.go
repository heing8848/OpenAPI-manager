package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/billing"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

func RelayTextHelper(c *gin.Context) *model.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	// get & validate textRequest
	textRequest, err := getAndValidateTextRequest(c, meta.Mode)
	if err != nil {
		logger.Errorf(ctx, "getAndValidateTextRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
	}
	meta.IsStream = textRequest.Stream

	// map model name
	meta.OriginModelName = textRequest.Model
	mappedModelName, mappingApplied := getMappedModelName(textRequest.Model, meta.ModelMapping)
	textRequest.Model = openai.NormalizeProviderCompatibleModelNameV2(meta.ChannelType, meta.BaseURL, mappedModelName, mappingApplied)
	meta.ActualModelName = textRequest.Model
	// set system prompt if not empty
	systemPromptReset := setSystemPrompt(ctx, textRequest, meta.ForcedSystemPrompt)
	// get model ratio & group ratio
	modelRatio := billingratio.GetModelRatio(textRequest.Model, meta.ChannelType)
	groupRatio := billingratio.GetGroupRatio(meta.Group)
	ratio := modelRatio * groupRatio
	// pre-consume quota
	promptTokens := getPromptTokens(textRequest, meta.Mode)
	meta.PromptTokens = promptTokens
	preConsumedQuota, bizErr := preConsumeQuota(ctx, textRequest, promptTokens, ratio, meta)
	if bizErr != nil {
		logger.Warnf(ctx, "preConsumeQuota failed: %+v", *bizErr)
		return bizErr
	}

	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	// get request body
	requestBody, err := getRequestBody(c, meta, textRequest, adaptor)
	if err != nil {
		return openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
	}

	// do request
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(meta, resp) {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return RelayErrorHandlerV2(resp)
	}

	// do response
	usage, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return respErr
	}
	// post-consume quota
	go postConsumeQuota(ctx, usage, meta, textRequest, ratio, preConsumedQuota, modelRatio, groupRatio, systemPromptReset)
	return nil
}

func getRequestBody(c *gin.Context, meta *meta.Meta, textRequest *model.GeneralOpenAIRequest, adaptor adaptor.Adaptor) (io.Reader, error) {
	if shouldBypassOpenAIRequestConversionV2(meta, textRequest) {
		sanitizeMessagesForUpstream(textRequest)
		jsonData, err := json.Marshal(textRequest)
		if err != nil {
			logger.Debugf(c.Request.Context(), "converted request json_marshal_failed (bypass): %s\n", err.Error())
			return nil, err
		}
		return bytes.NewBuffer(jsonData), nil
	}

	// get request body
	var requestBody io.Reader
	convertedRequest, err := adaptor.ConvertRequest(c, meta.Mode, textRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request failed: %s\n", err.Error())
		return nil, err
	}
	jsonData, err := json.Marshal(convertedRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request json_marshal_failed: %s\n", err.Error())
		return nil, err
	}
	logger.Debugf(c.Request.Context(), "converted request: \n%s", string(jsonData))
	requestBody = bytes.NewBuffer(jsonData)
	return requestBody, nil
}

func shouldBypassOpenAIRequestConversionV2(meta *meta.Meta, textRequest *model.GeneralOpenAIRequest) bool {
	if meta == nil || textRequest == nil {
		return false
	}
	if config.EnforceIncludeUsage {
		return false
	}
	if meta.APIType != apitype.OpenAI {
		return false
	}
	if openai.RequiresProviderSpecificOpenAIRequestConversionV2(meta.ChannelType, meta.BaseURL) {
		return false
	}
	if meta.OriginModelName != meta.ActualModelName {
		return false
	}
	if meta.ChannelType == channeltype.Baichuan {
		return false
	}
	if meta.ForcedSystemPrompt != "" {
		return false
	}
	if shouldForceGeminiCompatibleOpenAIRequestConversionV2(meta.ChannelType, textRequest.Model) {
		return false
	}
	return true
}

func shouldForceGeminiCompatibleOpenAIRequestConversionV2(channelType int, modelName string) bool {
	if channelType == channeltype.GeminiOpenAICompatible {
		return true
	}
	return channelType == channeltype.OpenAICompatible && model.IsGeminiModelLikeV2(modelName)
}

func sanitizeMessagesForUpstream(req *model.GeneralOpenAIRequest) {
	if req == nil || req.Messages == nil {
		return
	}
	for i := range req.Messages {
		req.Messages[i].ReasoningContent = nil
	}
}
