package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
)

type ChannelReorderRequestV2 struct {
	Id        int    `json:"id"`
	Direction string `json:"direction"`
}

func GetAllChannels(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	scope := c.DefaultQuery("scope", "limited")
	channels, err := model.GetAllChannelsV2(p*config.ItemsPerPage, config.ItemsPerPage, scope)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	for _, channel := range channels {
		channel.Failures = monitor.GetChannelFailures(channel.Id)
	}
	model.FillChannelHasDisabledKeys(channels)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
}

func SearchChannels(c *gin.Context) {
	keyword := c.Query("keyword")
	channels, err := model.SearchChannelsV2(keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	for _, channel := range channels {
		channel.Failures = monitor.GetChannelFailures(channel.Id)
	}
	model.FillChannelHasDisabledKeys(channels)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
}

func GetChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channel,
	})
}

func AddChannel(c *gin.Context) {
	request := model.ChannelWriteRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := validateVideoChannelWriteRequestV1(&request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	channel := request.ToChannel()
	channel.CreatedTime = helper.GetTimestamp()
	model.PrepareChannelModelIDPrefixForWriteV2(channel, nil)

	requestKeys := model.NormalizeChannelKeyValues(request.Keys)
	legacyKeys := model.SplitLegacyChannelKeys(request.Key)

	var err error
	createdChannels := make([]model.Channel, 0)
	switch {
	case len(requestKeys) > 0:
		channel.Key = strings.Join(requestKeys, "\n")
		err = channel.Insert()
		createdChannels = append(createdChannels, *channel)
	case len(legacyKeys) > 1:
		channels := make([]model.Channel, 0, len(legacyKeys))
		for _, key := range legacyKeys {
			localChannel := *channel
			localChannel.Key = key
			channels = append(channels, localChannel)
		}
		err = model.BatchInsertChannels(channels)
		createdChannels = append(createdChannels, channels...)
	default:
		channel.Key = strings.Join(legacyKeys, "\n")
		err = channel.Insert()
		createdChannels = append(createdChannels, *channel)
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	for _, createdChannel := range createdChannels {
		logChannelManageAction(c, createdChannel.Id, "create", createdChannel.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	channelName := ""
	if channel, err := model.GetChannelById(id, false); err == nil {
		channelName = channel.Name
	}
	channel := model.Channel{Id: id}
	err := channel.Delete()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	logChannelManageAction(c, id, "delete", channelName)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteDisabledChannel(c *gin.Context) {
	rows, err := model.DeleteDisabledChannel()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	model.RecordLog(c.Request.Context(), c.GetInt(ctxkey.Id), model.LogTypeManage, fmt.Sprintf("bulk deleted disabled channels, count: %d", rows))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
}

func UpdateChannel(c *gin.Context) {
	request := model.ChannelWriteRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := validateVideoChannelWriteRequestV1(&request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	channel := request.ToChannel()
	existingChannel, err := model.GetChannelById(channel.Id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	model.PrepareChannelModelIDPrefixForWriteV2(channel, existingChannel)
	keyValues := model.NormalizeChannelKeyValues(request.Keys)
	if len(keyValues) == 0 && strings.TrimSpace(request.Key) == "" {
		existingChannel, err := model.GetChannelById(channel.Id, true)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		for _, key := range existingChannel.Keys {
			keyValues = append(keyValues, key.KeyValue)
		}
	}
	if len(keyValues) == 0 {
		keyValues = model.SplitLegacyChannelKeys(request.Key)
	}
	channel.Key = strings.Join(keyValues, "\n")

	if err := channel.Update(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	logChannelManageAction(c, channel.Id, "update", channel.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channel,
	})
}

func MoveChannelOrderV2(c *gin.Context) {
	request := ChannelReorderRequestV2{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if request.Id == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "channel id is required",
		})
		return
	}

	direction := 0
	switch strings.ToLower(strings.TrimSpace(request.Direction)) {
	case "up":
		direction = -1
	case "down":
		direction = 1
	default:
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "direction must be up or down",
		})
		return
	}

	if err := model.MoveChannelDisplayOrderV2(request.Id, direction); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	action := "move_down"
	if direction < 0 {
		action = "move_up"
	}
	logChannelManageAction(c, request.Id, action, "")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func logChannelManageAction(c *gin.Context, channelId int, action string, channelName string) {
	if channelId == 0 {
		return
	}
	operatorId := c.GetInt(ctxkey.Id)
	if operatorId == 0 {
		return
	}
	name := strings.TrimSpace(channelName)
	if name == "" {
		model.RecordLogWithChannel(c.Request.Context(), operatorId, model.LogTypeManage, fmt.Sprintf("%s channel #%d", action, channelId), channelId, 0)
		return
	}
	model.RecordLogWithChannel(c.Request.Context(), operatorId, model.LogTypeManage, fmt.Sprintf("%s channel #%d (%s)", action, channelId, name), channelId, 0)
}
