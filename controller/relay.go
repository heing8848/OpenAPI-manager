package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	dbmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	relaycontroller "github.com/songquanpeng/one-api/relay/controller"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func relayHelper(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	var err *relaymodel.ErrorWithStatusCode
	switch relayMode {
	case relaymode.ImagesGenerations:
		err = relaycontroller.RelayImageHelperV2(c, relayMode)
	case relaymode.VideosGenerationsV1:
		fallthrough
	case relaymode.VideoGenerationsTasksV1:
		err = relaycontroller.RelayVideoHelperV1(c, relayMode)
	case relaymode.AudioSpeech:
		fallthrough
	case relaymode.AudioTranslation:
		fallthrough
	case relaymode.AudioTranscription:
		err = relaycontroller.RelayAudioHelperV2(c, relayMode)
	case relaymode.Proxy:
		err = relaycontroller.RelayProxyHelper(c, relayMode)
	default:
		err = relaycontroller.RelayTextHelper(c)
	}
	return err
}

func Relay(c *gin.Context) {
	ctx := c.Request.Context()
	relayMode := relaymode.GetByPath(c.Request.URL.Path)
	if config.DebugEnabled {
		requestBody, _ := common.GetRequestBody(c)
		logger.Debugf(ctx, "request body: %s", string(requestBody))
	}

	userId := c.GetInt(ctxkey.Id)
	requestId := c.GetString(helper.RequestIdKey)
	originalModel := c.GetString(ctxkey.OriginalModel)
	group := c.GetString(ctxkey.Group)

	if relayMode == relaymode.VideosGenerationsV1 || relayMode == relaymode.VideoGenerationsTasksV1 {
		bizErr := relayHelper(c, relayMode)
		if bizErr == nil {
			if c.GetInt(ctxkey.ChannelId) != 0 {
				finishRelaySuccess(c)
			}
			return
		}
		if c.GetInt(ctxkey.ChannelId) != 0 {
			processFailedRelayAttemptV2(ctx, c, userId, *bizErr)
		}
		respondRelayError(c, bizErr, requestId)
		return
	}

	bizErr := relayHelper(c, relayMode)
	if bizErr == nil {
		finishRelaySuccess(c)
		return
	}

	processFailedRelayAttemptV2(ctx, c, userId, *bizErr)

	if !shouldRetryStatusV2(bizErr.StatusCode) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry in this case", bizErr.StatusCode)
		respondRelayError(c, bizErr, requestId)
		return
	}

	requestBody, err := common.GetRequestBody(c)
	if err != nil {
		logger.Errorf(ctx, "failed to read request body for retry: %s", err.Error())
		respondRelayError(c, bizErr, requestId)
		return
	}

	bizErr = retryRemainingKeysOnCurrentChannelV2(c, relayMode, userId, requestBody, originalModel, bizErr)
	if bizErr == nil {
		return
	}

	if !shouldRetryAcrossChannelsV2(c, bizErr.StatusCode) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry across channels in this case", bizErr.StatusCode)
		respondRelayError(c, bizErr, requestId)
		return
	}

	bizErr = retryOtherChannelsV2(c, relayMode, userId, requestBody, group, originalModel, bizErr)
	respondRelayError(c, bizErr, requestId)
}

func retryRemainingKeysOnCurrentChannel(
	c *gin.Context,
	relayMode int,
	userId int,
	requestBody []byte,
	originalModel string,
	currentErr *relaymodel.ErrorWithStatusCode,
) *relaymodel.ErrorWithStatusCode {
	channelValue, ok := c.Get(ctxkey.ChannelObject)
	if !ok {
		return currentErr
	}
	channel, ok := channelValue.(*dbmodel.Channel)
	if !ok {
		return currentErr
	}

	channelKeyPoolValue, ok := c.Get(ctxkey.ChannelKeyPool)
	if !ok {
		return currentErr
	}
	channelKeyPool, ok := channelKeyPoolValue.([]dbmodel.ChannelKey)
	if !ok || len(channelKeyPool) == 0 {
		return currentErr
	}

	currentKeyId := c.GetInt(ctxkey.ChannelKeyId)
	startIndex := len(channelKeyPool)
	for i := range channelKeyPool {
		if channelKeyPool[i].Id == currentKeyId {
			startIndex = i + 1
			break
		}
	}

	ctx := c.Request.Context()
	bizErr := currentErr
	for i := startIndex; i < len(channelKeyPool); i++ {
		channelKey := channelKeyPool[i]
		logger.Infof(ctx, "retrying channel #%d with key #%d", channel.Id, channelKey.Id)
		middleware.SetupContextForSelectedChannelWithKey(c, channel, &channelKey, originalModel)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		bizErr = relayHelper(c, relayMode)
		if bizErr == nil {
			finishRelaySuccess(c)
			return nil
		}
		processFailedRelayAttempt(ctx, c, userId, *bizErr)
		if !shouldRetryStatus(bizErr.StatusCode) {
			return bizErr
		}
	}

	return bizErr
}

func retryOtherChannels(
	c *gin.Context,
	relayMode int,
	userId int,
	requestBody []byte,
	group string,
	originalModel string,
	currentErr *relaymodel.ErrorWithStatusCode,
) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	bizErr := currentErr
	retryTimes := config.RetryTimes
	if retryTimes <= 0 {
		return bizErr
	}

	attemptedChannels := map[int]bool{
		c.GetInt(ctxkey.ChannelId): true,
	}

	for remaining := retryTimes; remaining > 0; remaining-- {
		channel, err := dbmodel.CacheGetRandomSatisfiedChannel(group, originalModel, remaining != retryTimes)
		if err != nil {
			logger.Errorf(ctx, "CacheGetRandomSatisfiedChannel failed: %+v", err)
			break
		}
		if attemptedChannels[channel.Id] {
			continue
		}
		attemptedChannels[channel.Id] = true

		logger.Infof(ctx, "using channel #%d to retry (remain times %d)", channel.Id, remaining)
		if err = middleware.SetupContextForSelectedChannel(c, channel, originalModel); err != nil {
			logger.Warnf(ctx, "skip retry channel #%d: %s", channel.Id, err.Error())
			continue
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		bizErr = relayHelper(c, relayMode)
		if bizErr == nil {
			finishRelaySuccess(c)
			return nil
		}

		processFailedRelayAttempt(ctx, c, userId, *bizErr)
		if !shouldRetryStatus(bizErr.StatusCode) {
			return bizErr
		}

		bizErr = retryRemainingKeysOnCurrentChannel(c, relayMode, userId, requestBody, originalModel, bizErr)
		if bizErr == nil {
			return nil
		}
		if !shouldRetryStatus(bizErr.StatusCode) {
			return bizErr
		}
	}

	return bizErr
}

func finishRelaySuccess(c *gin.Context) {
	channelId := c.GetInt(ctxkey.ChannelId)
	channelKeyId := c.GetInt(ctxkey.ChannelKeyId)
	monitor.Emit(channelId, true)
	monitor.ClearChannelFailures(channelId)
	if err := dbmodel.MarkChannelKeySuccessV2(channelId, channelKeyId); err != nil {
		logger.Errorf(c.Request.Context(), "failed to update channel key success state: %s", err.Error())
	}
}

func processFailedRelayAttempt(ctx context.Context, c *gin.Context, userId int, err relaymodel.ErrorWithStatusCode) {
	channelId := c.GetInt(ctxkey.ChannelId)
	channelName := c.GetString(ctxkey.ChannelName)
	channelKeyId := c.GetInt(ctxkey.ChannelKeyId)
	if updateErr := dbmodel.MarkChannelKeyFailure(channelId, channelKeyId, err.StatusCode, err.Message); updateErr != nil {
		logger.Errorf(ctx, "failed to update channel key failure state: %s", updateErr.Error())
	}
	go processChannelRelayError(ctx, userId, channelId, channelName, err)
}

func respondRelayError(c *gin.Context, bizErr *relaymodel.ErrorWithStatusCode, requestId string) {
	if bizErr == nil {
		bizErr = &relaymodel.ErrorWithStatusCode{
			Error: relaymodel.Error{
				Message: "relay failed without detailed error",
				Type:    "one_api_error",
				Code:    "relay_failed",
			},
			StatusCode: http.StatusBadGateway,
		}
	}
	if bizErr.StatusCode == http.StatusTooManyRequests {
		bizErr.Error.Message = "当前分组上游负载已饱和，请稍后再试"
	}
	bizErr.Error.Message = helper.MessageWithRequestId(bizErr.Error.Message, requestId)
	recordFinalRelayFailureLogV2(c, bizErr)
	c.JSON(bizErr.StatusCode, gin.H{
		"error": bizErr.Error,
	})
}

func shouldRetryStatus(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if statusCode/100 == 5 {
		return true
	}
	if statusCode == http.StatusBadRequest {
		return false
	}
	if statusCode/100 == 2 {
		return false
	}
	return true
}

func shouldRetryAcrossChannels(c *gin.Context, statusCode int) bool {
	if _, ok := c.Get(ctxkey.SpecificChannelId); ok {
		return false
	}
	return shouldRetryStatus(statusCode)
}

func processChannelRelayError(ctx context.Context, userId int, channelId int, channelName string, err relaymodel.ErrorWithStatusCode) {
	logger.Errorf(ctx, "relay error (channel id %d, user id: %d): %s", channelId, userId, err.Message)
	monitor.RecordAndGetChannelFailures(channelId)
	monitor.Emit(channelId, false)
}

func RelayNotImplemented(c *gin.Context) {
	err := relaymodel.Error{
		Message: "API not implemented",
		Type:    "one_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := relaymodel.Error{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}
