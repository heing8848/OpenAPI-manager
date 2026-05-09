package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type StandaloneOutputCapabilityV3 struct {
	ImageGeneration bool
	AudioSpeech     bool
	VideoGeneration bool
}

func GetStandaloneOutputCapabilityV3(channelType int) StandaloneOutputCapabilityV3 {
	capabilityV2 := GetStandaloneOutputCapabilityV2(channelType)
	capability := StandaloneOutputCapabilityV3{
		ImageGeneration: capabilityV2.ImageGeneration,
		AudioSpeech:     capabilityV2.AudioSpeech,
	}
	if channelType == channeltype.VideoTaskV1 {
		capability.VideoGeneration = true
	}
	return capability
}

func ValidateStandaloneOutputCapabilityV3(metaInfo *meta.Meta, relayMode int) *relaymodel.ErrorWithStatusCode {
	if metaInfo == nil {
		return openai.ErrorWrapper(errors.New("relay meta is nil"), "invalid_relay_meta_v3", http.StatusInternalServerError)
	}

	capability := GetStandaloneOutputCapabilityV3(metaInfo.ChannelType)
	switch relayMode {
	case relaymode.VideosGenerationsV1:
		if capability.VideoGeneration {
			return nil
		}
		message := fmt.Sprintf(
			"standalone video generation is not supported for channel type %s",
			getStandaloneOutputChannelNameV3(metaInfo.ChannelType),
		)
		return openai.ErrorWrapper(
			errors.New(message),
			"video_generation_unsupported_v1",
			http.StatusBadRequest,
		)
	default:
		return ValidateStandaloneOutputCapabilityV2(metaInfo, relayMode)
	}
}

func getStandaloneOutputChannelNameV3(channelType int) string {
	if channelType == channeltype.VideoTaskV1 {
		return "video-task-v1"
	}
	return getStandaloneOutputChannelNameV2(channelType)
}
