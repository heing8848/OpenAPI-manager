package controller

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	dbmodel "github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// retryRemainingKeysOnCurrentChannelV2 intentionally keeps the request pinned
// to the key chosen at distribution time. We still rotate keys between
// requests, but we no longer cascade through every key in a single failing
// request.
func retryRemainingKeysOnCurrentChannelV2(
	_ *gin.Context,
	_ int,
	_ int,
	_ []byte,
	_ string,
	currentErr *relaymodel.ErrorWithStatusCode,
) *relaymodel.ErrorWithStatusCode {
	return currentErr
}

func retryOtherChannelsV2(
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

		processFailedRelayAttemptV2(ctx, c, userId, *bizErr)
		if !shouldRetryStatusV2(bizErr.StatusCode) {
			return bizErr
		}
	}

	return bizErr
}

func processFailedRelayAttemptV2(ctx context.Context, c *gin.Context, userId int, err relaymodel.ErrorWithStatusCode) {
	channelId := c.GetInt(ctxkey.ChannelId)
	channelName := c.GetString(ctxkey.ChannelName)
	channelKeyId := c.GetInt(ctxkey.ChannelKeyId)
	if updateErr := dbmodel.MarkChannelKeyFailureV2(channelId, channelKeyId, err.StatusCode, err.Message); updateErr != nil {
		logger.Errorf(ctx, "failed to update channel key failure state: %s", updateErr.Error())
	}
	go processChannelRelayError(ctx, userId, channelId, channelName, err)
}

func shouldRetryStatusV2(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	return statusCode/100 == 5
}

func shouldRetryAcrossChannelsV2(c *gin.Context, statusCode int) bool {
	if _, ok := c.Get(ctxkey.SpecificChannelId); ok {
		return false
	}
	return shouldRetryStatusV2(statusCode)
}
