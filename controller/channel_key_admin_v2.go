package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

func EnableChannelKeyV2(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid channel id",
		})
		return
	}

	channelKeyId, err := strconv.Atoi(c.Param("keyId"))
	if err != nil || channelKeyId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid channel key id",
		})
		return
	}

	key, err := model.EnableChannelKeyV2(channelId, channelKeyId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	operatorId := c.GetInt(ctxkey.Id)
	if operatorId != 0 {
		model.RecordLogWithChannel(
			c.Request.Context(),
			operatorId,
			model.LogTypeManage,
			fmt.Sprintf("enable key #%d on channel #%d", channelKeyId, channelId),
			channelId,
			channelKeyId,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    key,
	})
}
