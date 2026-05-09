package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

type ModelRequest struct {
	Model string `json:"model" form:"model"`
}

func Distribute() func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(userId)
		c.Set(ctxkey.Group, userGroup)

		var requestModel string
		var channel *model.Channel

		channelId, ok := c.Get(ctxkey.SpecificChannelId)
		if ok {
			id, err := strconv.Atoi(channelId.(string))
			if err != nil {
				abortWithMessage(c, http.StatusBadRequest, "invalid channel id")
				return
			}
			channel, err = model.GetChannelById(id, true)
			if err != nil {
				abortWithMessage(c, http.StatusBadRequest, "invalid channel id")
				return
			}
			if channel.Status != model.ChannelStatusEnabled {
				abortWithMessage(c, http.StatusForbidden, "channel is disabled")
				return
			}
			if err = SetupContextForSelectedChannel(c, channel, requestModel); err != nil {
				abortWithMessage(c, http.StatusServiceUnavailable, err.Error())
				return
			}
		} else {
			requestModel = c.GetString(ctxkey.RequestModel)
			maxAttempts := config.RetryTimes + 3
			if maxAttempts < 1 {
				maxAttempts = 1
			}

			attemptedChannels := make(map[int]bool)
			var lastErr error
			for attempt := 0; attempt < maxAttempts; attempt++ {
				var err error
				channel, err = model.CacheGetRandomSatisfiedChannel(userGroup, requestModel, attempt > 0)
				if err != nil {
					lastErr = err
					break
				}
				if attemptedChannels[channel.Id] {
					continue
				}
				attemptedChannels[channel.Id] = true
				if err = SetupContextForSelectedChannel(c, channel, requestModel); err != nil {
					lastErr = err
					logger.Warnf(ctx, "channel #%d skipped during distribution: %s", channel.Id, err.Error())
					channel = nil
					continue
				}
				lastErr = nil
				break
			}

			if lastErr != nil || channel == nil {
				message := fmt.Sprintf("no available channel for group %s and model %s", userGroup, requestModel)
				if lastErr != nil {
					message = lastErr.Error()
				}
				abortWithMessage(c, http.StatusServiceUnavailable, message)
				return
			}
		}

		logger.Debugf(ctx, "user id %d, user group: %s, request model: %s, using channel #%d", userId, userGroup, requestModel, channel.Id)
		c.Next()
	}
}

func SetupContextForSelectedChannel(c *gin.Context, channel *model.Channel, modelName string) error {
	channelKeys, err := model.PrepareChannelKeyCandidatesV2(channel)
	if err != nil {
		return err
	}
	c.Set(ctxkey.ChannelObject, channel)
	c.Set(ctxkey.ChannelKeyPool, channelKeys)
	SetupContextForSelectedChannelWithKey(c, channel, &channelKeys[0], modelName)
	return nil
}

func SetupContextForSelectedChannelWithKey(c *gin.Context, channel *model.Channel, channelKey *model.ChannelKey, modelName string) {
	c.Set(ctxkey.Channel, channel.Type)
	c.Set(ctxkey.ChannelId, channel.Id)
	c.Set(ctxkey.ChannelName, channel.Name)
	c.Set(ctxkey.ChannelKeyId, channelKey.Id)
	c.Set(ctxkey.ChannelKeyIndex, channelKey.Position+1)
	c.Set(ctxkey.ChannelKeyValue, channelKey.KeyValue)
	if channel.SystemPrompt != nil && *channel.SystemPrompt != "" {
		c.Set(ctxkey.SystemPrompt, *channel.SystemPrompt)
	}
	c.Set(ctxkey.ModelMapping, channel.GetModelMappingV2())
	c.Set(ctxkey.OriginalModel, modelName)
	c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channelKey.KeyValue))
	c.Set(ctxkey.BaseURL, channel.GetBaseURL())

	cfg, _ := channel.LoadConfig()
	if channel.Other != nil {
		switch channel.Type {
		case channeltype.Azure:
			if cfg.APIVersion == "" {
				cfg.APIVersion = *channel.Other
			}
		case channeltype.Xunfei:
			if cfg.APIVersion == "" {
				cfg.APIVersion = *channel.Other
			}
		case channeltype.Gemini:
			if cfg.APIVersion == "" {
				cfg.APIVersion = *channel.Other
			}
		case channeltype.AIProxyLibrary:
			if cfg.LibraryID == "" {
				cfg.LibraryID = *channel.Other
			}
		case channeltype.Ali:
			if cfg.Plugin == "" {
				cfg.Plugin = *channel.Other
			}
		}
	}
	c.Set(ctxkey.Config, cfg)
}
