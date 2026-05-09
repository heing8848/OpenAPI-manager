package openai

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func TestConvertRequestSanitizesGeminiLikeOpenAICompatibleToolsV2(t *testing.T) {
	adaptor := &Adaptor{ChannelType: channeltype.OpenAICompatible}
	request := &relaymodel.GeneralOpenAIRequest{
		Model: "google-gemini-3.1-flash-lite-preview",
		Tools: []relaymodel.Tool{
			{
				Type: "function",
				Function: relaymodel.Function{
					Name: "builtin_web_search",
					Parameters: map[string]any{
						"type":                 "object",
						"additionalProperties": false,
					},
				},
			},
		},
	}

	converted, err := adaptor.ConvertRequest(nil, 0, request)
	if err != nil {
		t.Fatalf("expected convert request to succeed, got %v", err)
	}
	convertedRequest := converted.(*relaymodel.GeneralOpenAIRequest)
	parameters := convertedRequest.Tools[0].Function.Parameters.(map[string]any)
	if _, exists := parameters["additionalProperties"]; exists {
		t.Fatalf("expected additionalProperties to be removed for gemini-like openai-compatible request, got %#v", parameters)
	}
}

func TestConvertRequestKeepsNonGeminiOpenAICompatibleToolsV2(t *testing.T) {
	adaptor := &Adaptor{ChannelType: channeltype.OpenAICompatible}
	request := &relaymodel.GeneralOpenAIRequest{
		Model: "gpt-4o-mini",
		Tools: []relaymodel.Tool{
			{
				Type: "function",
				Function: relaymodel.Function{
					Name: "builtin_web_search",
					Parameters: map[string]any{
						"type":                 "object",
						"additionalProperties": false,
					},
				},
			},
		},
	}

	converted, err := adaptor.ConvertRequest(nil, 0, request)
	if err != nil {
		t.Fatalf("expected convert request to succeed, got %v", err)
	}
	convertedRequest := converted.(*relaymodel.GeneralOpenAIRequest)
	parameters := convertedRequest.Tools[0].Function.Parameters.(map[string]any)
	if _, exists := parameters["additionalProperties"]; !exists {
		t.Fatalf("expected additionalProperties to stay for non-gemini openai-compatible request, got %#v", parameters)
	}
}
