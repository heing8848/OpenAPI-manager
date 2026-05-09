package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type StandaloneOutputCapabilityV2 struct {
	ImageGeneration bool
	AudioSpeech     bool
}

func GetStandaloneOutputCapabilityV2(channelType int) StandaloneOutputCapabilityV2 {
	switch channelType {
	case channeltype.OpenAI,
		channeltype.API2D,
		channeltype.Azure,
		channeltype.CloseAI,
		channeltype.OpenAISB,
		channeltype.OpenAIMax,
		channeltype.OhMyGPT,
		channeltype.Custom,
		channeltype.Ails,
		channeltype.AIProxy,
		channeltype.API2GPT,
		channeltype.AIGC2D,
		channeltype.OpenAICompatible:
		return StandaloneOutputCapabilityV2{
			ImageGeneration: true,
			AudioSpeech:     true,
		}
	case channeltype.Zhipu,
		channeltype.Ali,
		channeltype.Replicate:
		return StandaloneOutputCapabilityV2{
			ImageGeneration: true,
		}
	default:
		return StandaloneOutputCapabilityV2{}
	}
}

func ValidateStandaloneOutputCapabilityV2(metaInfo *meta.Meta, relayMode int) *relaymodel.ErrorWithStatusCode {
	if metaInfo == nil {
		return openai.ErrorWrapper(fmt.Errorf("relay meta is nil"), "invalid_relay_meta_v2", http.StatusInternalServerError)
	}

	modelName := getStandaloneOutputModelNameV2(metaInfo)
	capability := GetStandaloneOutputCapabilityV2(metaInfo.ChannelType)
	switch relayMode {
	case relaymode.ImagesGenerations:
		if capability.ImageGeneration {
			return nil
		}
		message := fmt.Sprintf(
			"standalone image generation is not supported for channel type %s",
			getStandaloneOutputChannelNameV2(metaInfo.ChannelType),
		)
		logger.Warnf(
			context.TODO(),
			"blocking unsupported image generation request: channel_type=%s channel_id=%d model=%s",
			getStandaloneOutputChannelNameV2(metaInfo.ChannelType),
			metaInfo.ChannelId,
			modelName,
		)
		return openai.ErrorWrapper(
			errors.New(message),
			"image_generation_unsupported_v2",
			http.StatusBadRequest,
		)
	case relaymode.AudioSpeech:
		if capability.AudioSpeech {
			return nil
		}
		message := fmt.Sprintf(
			"standalone audio speech output is not supported for channel type %s",
			getStandaloneOutputChannelNameV2(metaInfo.ChannelType),
		)
		logger.Warnf(
			context.TODO(),
			"blocking unsupported audio speech request: channel_type=%s channel_id=%d model=%s",
			getStandaloneOutputChannelNameV2(metaInfo.ChannelType),
			metaInfo.ChannelId,
			modelName,
		)
		return openai.ErrorWrapper(
			errors.New(message),
			"audio_speech_unsupported_v2",
			http.StatusBadRequest,
		)
	default:
		return nil
	}
}

func getStandaloneOutputModelNameV2(metaInfo *meta.Meta) string {
	if metaInfo == nil {
		return ""
	}
	if metaInfo.ActualModelName != "" {
		return metaInfo.ActualModelName
	}
	return metaInfo.OriginModelName
}

func getStandaloneOutputChannelNameV2(channelType int) string {
	switch channelType {
	case channeltype.OpenAI:
		return "openai"
	case channeltype.API2D:
		return "api2d"
	case channeltype.Azure:
		return "azure"
	case channeltype.CloseAI:
		return "closeai"
	case channeltype.OpenAISB:
		return "openai-sb"
	case channeltype.OpenAIMax:
		return "openai-max"
	case channeltype.OhMyGPT:
		return "ohmygpt"
	case channeltype.Custom:
		return "custom"
	case channeltype.Ails:
		return "ails"
	case channeltype.AIProxy:
		return "aiproxy"
	case channeltype.API2GPT:
		return "api2gpt"
	case channeltype.AIGC2D:
		return "aigc2d"
	case channeltype.OpenAICompatible:
		return "openai-compatible"
	case channeltype.Zhipu:
		return "zhipu"
	case channeltype.Ali:
		return "ali"
	case channeltype.Replicate:
		return "replicate"
	case channeltype.Gemini:
		return "gemini"
	case channeltype.GeminiOpenAICompatible:
		return "gemini-openai-compatible"
	case channeltype.Groq:
		return "groq"
	case channeltype.Anthropic:
		return "anthropic"
	case channeltype.Baidu:
		return "baidu"
	default:
		return fmt.Sprintf("channel-type-%d", channelType)
	}
}
