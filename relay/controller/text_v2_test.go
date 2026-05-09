package controller

import (
	"testing"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

func TestShouldBypassOpenAIRequestConversionV2ReturnsFalseForGeminiOpenAICompatible(t *testing.T) {
	previousEnforceIncludeUsage := config.EnforceIncludeUsage
	config.EnforceIncludeUsage = false
	defer func() {
		config.EnforceIncludeUsage = previousEnforceIncludeUsage
	}()

	shouldBypass := shouldBypassOpenAIRequestConversionV2(&meta.Meta{
		APIType:         apitype.OpenAI,
		ChannelType:     channeltype.GeminiOpenAICompatible,
		OriginModelName: "google-gemini-3.1-flash-lite-preview",
		ActualModelName: "google-gemini-3.1-flash-lite-preview",
	}, &model.GeneralOpenAIRequest{
		Model: "google-gemini-3.1-flash-lite-preview",
	})
	if shouldBypass {
		t.Fatalf("expected Gemini OpenAI-compatible requests to force conversion")
	}
}

func TestShouldBypassOpenAIRequestConversionV2ReturnsFalseForGeminiLikeOpenAICompatible(t *testing.T) {
	previousEnforceIncludeUsage := config.EnforceIncludeUsage
	config.EnforceIncludeUsage = false
	defer func() {
		config.EnforceIncludeUsage = previousEnforceIncludeUsage
	}()

	shouldBypass := shouldBypassOpenAIRequestConversionV2(&meta.Meta{
		APIType:         apitype.OpenAI,
		ChannelType:     channeltype.OpenAICompatible,
		OriginModelName: "google-gemini-3.1-flash-lite-preview",
		ActualModelName: "google-gemini-3.1-flash-lite-preview",
	}, &model.GeneralOpenAIRequest{
		Model: "google-gemini-3.1-flash-lite-preview",
	})
	if shouldBypass {
		t.Fatalf("expected Gemini-like OpenAI-compatible requests to force conversion")
	}
}

func TestShouldBypassOpenAIRequestConversionV2ReturnsTrueForPlainOpenAI(t *testing.T) {
	previousEnforceIncludeUsage := config.EnforceIncludeUsage
	config.EnforceIncludeUsage = false
	defer func() {
		config.EnforceIncludeUsage = previousEnforceIncludeUsage
	}()

	shouldBypass := shouldBypassOpenAIRequestConversionV2(&meta.Meta{
		APIType:         apitype.OpenAI,
		ChannelType:     channeltype.OpenAI,
		OriginModelName: "gpt-4o-mini",
		ActualModelName: "gpt-4o-mini",
	}, &model.GeneralOpenAIRequest{
		Model: "gpt-4o-mini",
	})
	if !shouldBypass {
		t.Fatalf("expected plain OpenAI requests to keep bypassing conversion")
	}
}

func TestShouldBypassOpenAIRequestConversionV2ReturnsFalseForGroqCompatibleBaseURL(t *testing.T) {
	previousEnforceIncludeUsage := config.EnforceIncludeUsage
	config.EnforceIncludeUsage = false
	defer func() {
		config.EnforceIncludeUsage = previousEnforceIncludeUsage
	}()

	shouldBypass := shouldBypassOpenAIRequestConversionV2(&meta.Meta{
		APIType:         apitype.OpenAI,
		ChannelType:     channeltype.OpenAICompatible,
		BaseURL:         "https://api.groq.com/openai/v1",
		OriginModelName: "groq-openai/gpt-oss-120b",
		ActualModelName: "groq-openai/gpt-oss-120b",
	}, &model.GeneralOpenAIRequest{
		Model: "groq-openai/gpt-oss-120b",
	})
	if shouldBypass {
		t.Fatalf("expected Groq-compatible OpenAI requests to force conversion")
	}
}

func TestShouldBypassOpenAIRequestConversionV2ReturnsFalseForMistralCompatibleBaseURL(t *testing.T) {
	previousEnforceIncludeUsage := config.EnforceIncludeUsage
	config.EnforceIncludeUsage = false
	defer func() {
		config.EnforceIncludeUsage = previousEnforceIncludeUsage
	}()

	shouldBypass := shouldBypassOpenAIRequestConversionV2(&meta.Meta{
		APIType:         apitype.OpenAI,
		ChannelType:     channeltype.OpenAICompatible,
		BaseURL:         "https://api.mistral.ai",
		OriginModelName: "mistral-large-latest",
		ActualModelName: "mistral-large-latest",
	}, &model.GeneralOpenAIRequest{
		Model: "mistral-large-latest",
	})
	if shouldBypass {
		t.Fatalf("expected Mistral-compatible OpenAI requests to force conversion")
	}
}
