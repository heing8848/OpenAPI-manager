package controller

import (
	"errors"
	"strings"

	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

func validateVideoChannelWriteRequestV1(request *model.ChannelWriteRequest) error {
	if request == nil || request.Type != channeltype.VideoTaskV1 {
		return nil
	}
	if request.BaseURL == nil || strings.TrimSpace(*request.BaseURL) == "" {
		return errors.New("base url is required for video task v1 channels")
	}
	if strings.TrimSpace(request.Models) == "" {
		return errors.New("at least one model is required for video task v1 channels")
	}
	return nil
}
