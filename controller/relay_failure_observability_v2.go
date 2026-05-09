package controller

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	dbmodel "github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func recordFinalRelayFailureLogV2(c *gin.Context, bizErr *relaymodel.ErrorWithStatusCode) {
	logRecord := buildFinalRelayFailureLogV2(c, bizErr)
	if logRecord == nil {
		return
	}
	dbmodel.RecordConsumeLog(c.Request.Context(), logRecord)
}

func buildFinalRelayFailureLogV2(c *gin.Context, bizErr *relaymodel.ErrorWithStatusCode) *dbmodel.Log {
	if c == nil || bizErr == nil {
		return nil
	}

	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		return nil
	}

	message := strings.TrimSpace(bizErr.Message)
	if message == "" {
		message = fmt.Sprintf("bad response status code %d", bizErr.StatusCode)
	}

	modelName := strings.TrimSpace(c.GetString(ctxkey.RequestModel))
	if modelName == "" {
		modelName = strings.TrimSpace(c.GetString(ctxkey.OriginalModel))
	}

	return &dbmodel.Log{
		UserId:          userId,
		ChannelId:       c.GetInt(ctxkey.ChannelId),
		ChannelKeyId:    c.GetInt(ctxkey.ChannelKeyId),
		ChannelKeyIndex: c.GetInt(ctxkey.ChannelKeyIndex),
		ModelName:       modelName,
		TokenName:       c.GetString(ctxkey.TokenName),
		Content:         fmt.Sprintf("请求失败 (%d): %s", bizErr.StatusCode, message),
	}
}
