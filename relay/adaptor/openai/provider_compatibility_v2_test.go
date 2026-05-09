package openai

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func TestRequiresProviderSpecificOpenAIRequestConversionV2DetectsCompatibleTargets(t *testing.T) {
	if !RequiresProviderSpecificOpenAIRequestConversionV2(channeltype.OpenAICompatible, "https://api.groq.com/openai/v1") {
		t.Fatalf("expected Groq-compatible OpenAI endpoint to force conversion")
	}
	if !RequiresProviderSpecificOpenAIRequestConversionV2(channeltype.OpenAICompatible, "https://api.mistral.ai") {
		t.Fatalf("expected Mistral-compatible OpenAI endpoint to force conversion")
	}
	if RequiresProviderSpecificOpenAIRequestConversionV2(channeltype.OpenAI, "https://api.openai.com") {
		t.Fatalf("did not expect plain OpenAI endpoint to force provider-specific conversion")
	}
}

func TestNormalizeProviderCompatibleModelNameV2StripsGroqPrefixWhenUnmapped(t *testing.T) {
	got := NormalizeProviderCompatibleModelNameV2(
		channeltype.OpenAICompatible,
		"https://api.groq.com/openai/v1",
		"groq-openai/gpt-oss-120b",
		false,
	)
	if got != "openai/gpt-oss-120b" {
		t.Fatalf("expected Groq display prefix to be removed, got %q", got)
	}
}

func TestSanitizeProviderCompatibleRequestV2ForGroq(t *testing.T) {
	messageName := "assistant-name"
	logprobs := true
	topLogprobs := 5
	request := &relaymodel.GeneralOpenAIRequest{
		Messages: []relaymodel.Message{{
			Role:             "assistant",
			Content:          "hello",
			ReasoningContent: "hidden",
			Name:             &messageName,
		}},
		Logprobs:    &logprobs,
		LogitBias:   map[string]float64{"42": 1},
		TopLogprobs: &topLogprobs,
		N:           3,
	}

	SanitizeProviderCompatibleRequestV2(channeltype.Groq, "https://api.groq.com/openai/v1", request)

	if request.Messages[0].ReasoningContent != nil {
		t.Fatalf("expected Groq sanitizer to remove reasoning content")
	}
	if request.Messages[0].Name != nil {
		t.Fatalf("expected Groq sanitizer to remove messages[].name")
	}
	if request.Logprobs != nil || request.LogitBias != nil || request.TopLogprobs != nil {
		t.Fatalf("expected Groq sanitizer to drop unsupported logprob fields")
	}
	if request.N != 1 {
		t.Fatalf("expected Groq sanitizer to clamp n to 1, got %d", request.N)
	}
}

func TestSanitizeProviderCompatibleRequestV2ForMistral(t *testing.T) {
	request := &relaymodel.GeneralOpenAIRequest{
		Messages: []relaymodel.Message{{
			Role:             "assistant",
			Content:          "hello",
			ReasoningContent: "hidden",
		}},
		StreamOptions: &relaymodel.StreamOptions{
			IncludeUsage: true,
		},
	}

	SanitizeProviderCompatibleRequestV2(channeltype.Mistral, "https://api.mistral.ai", request)

	if request.Messages[0].ReasoningContent != nil {
		t.Fatalf("expected Mistral sanitizer to remove reasoning content")
	}
	if request.StreamOptions != nil {
		t.Fatalf("expected Mistral sanitizer to drop stream_options")
	}
}
