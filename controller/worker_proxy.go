package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

type WorkerRouteRequest struct {
	Model string `json:"model"`
}

type WorkerRouteResponse struct {
	ChannelId    int                 `json:"channel_id"`
	BaseURL      string              `json:"base_url"`
	Key          string              `json:"key"`
	ChannelType  int                 `json:"channel_type"`
	Config       model.ChannelConfig `json:"config"`
	SystemPrompt string              `json:"system_prompt,omitempty"`
	ActualModel  string              `json:"actual_model"`
	UserId       int                 `json:"user_id"`
	TokenId      int                 `json:"token_id"`
	TokenName    string              `json:"token_name"`
}

func WorkerRoute(c *gin.Context) {
	var req WorkerRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userId := c.GetInt(ctxkey.Id)
	userGroup, _ := model.CacheGetUserGroup(userId)

	channel, err := model.CacheGetRandomSatisfiedChannel(userGroup, req.Model, false)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available channel"})
		return
	}

	cfg, _ := channel.LoadConfig()
	if !cfg.EdgeProxy {
		c.JSON(http.StatusForbidden, gin.H{"error": "channel does not support edge proxy"})
		return
	}

	keys, err := model.PrepareChannelKeyCandidatesV2(channel)
	if err != nil || len(keys) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no keys available for channel"})
		return
	}

	actualModel := req.Model
	if mapping := channel.GetModelMappingV2(); mapping != nil {
		if mappedModel, ok := mapping[req.Model]; ok && mappedModel != "" {
			actualModel = mappedModel
		}
	}

	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		baseURL = channeltype.ChannelBaseURLs[channel.Type] // Fallback to default
	}

	res := WorkerRouteResponse{
		ChannelId:   channel.Id,
		BaseURL:     baseURL,
		Key:         keys[0].KeyValue,
		ChannelType: channel.Type,
		Config:      cfg,
		ActualModel: actualModel,
		UserId:      userId,
		TokenId:     c.GetInt(ctxkey.TokenId),
		TokenName:   c.GetString(ctxkey.TokenName),
	}
	if channel.SystemPrompt != nil {
		res.SystemPrompt = *channel.SystemPrompt
	}

	c.JSON(http.StatusOK, res)
}

type WorkerCallbackRequest struct {
	UserId           int    `json:"user_id"`
	TokenId          int    `json:"token_id"`
	TokenName        string `json:"token_name"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	ChannelId        int    `json:"channel_id"`
	ElapsedTime      int64  `json:"elapsed_time"`
	IsStream         bool   `json:"is_stream"`
	IsError          bool   `json:"is_error"`
	ErrorMsg         string `json:"error_msg"`
}

func WorkerCallback(c *gin.Context) {
	var req WorkerCallbackRequest
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

	var quota int64
	quota = int64((float64(req.PromptTokens) + float64(req.CompletionTokens)*completionRatio) * ratioVal)

	if ratioVal != 0 && quota <= 0 {
		quota = 1
	}

	// NOTE: In Edge Proxy mode, we don't PreConsume quota to avoid latency. We directly PostConsume it here.
	err = model.CacheDecreaseUserQuota(req.UserId, quota)
	if err != nil {
		logger.SysError("failed to decrease user quota for edge proxy: " + err.Error())
	}

	err = model.PostConsumeTokenQuota(req.TokenId, quota)
	if err != nil {
		logger.SysError("failed to post consume token quota for edge proxy: " + err.Error())
	}
	err = model.CacheUpdateUserQuota(c.Request.Context(), req.UserId)
	if err != nil {
		logger.SysError("failed to update user quota cache for edge proxy: " + err.Error())
	}

	if req.IsError {
		logger.SysError(fmt.Sprintf("[Edge Proxy] 渠道 #%d 运行错误: %s", req.ChannelId, req.ErrorMsg))
		model.RecordLogWithChannel(c.Request.Context(), req.UserId, model.LogTypeSystem, fmt.Sprintf("[Edge Proxy] 渠道运行错误: %s", req.ErrorMsg), req.ChannelId, 0)
	}

	logContent := fmt.Sprintf("[Edge Proxy] 倍率：%.2f × %.2f × %.2f", modelRatio, groupRatio, completionRatio)
	if req.IsError {
		logContent += fmt.Sprintf(" (包含运行错误：%s)", req.ErrorMsg)
	}
	model.RecordConsumeLog(c.Request.Context(), &model.Log{
		UserId:           req.UserId,
		ChannelId:        req.ChannelId,
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
