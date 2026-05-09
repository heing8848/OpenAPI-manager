package openai

import (
	"strings"

	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func RequiresProviderSpecificOpenAIRequestConversionV2(channelType int, baseURL string) bool {
	return isGroqCompatibleTargetV2(channelType, baseURL) || isMistralCompatibleTargetV2(channelType, baseURL)
}

func NormalizeProviderCompatibleModelNameV2(channelType int, baseURL string, modelName string, mappingApplied bool) string {
	if mappingApplied {
		return modelName
	}
	if isGroqCompatibleTargetV2(channelType, baseURL) && hasLowercasePrefixV2(modelName, "groq-") {
		return modelName[len("groq-"):]
	}
	return modelName
}

func ShouldSkipStreamOptionsForProviderV2(channelType int, baseURL string) bool {
	return isMistralCompatibleTargetV2(channelType, baseURL)
}

func SanitizeProviderCompatibleRequestV2(channelType int, baseURL string, request *relaymodel.GeneralOpenAIRequest) {
	if request == nil {
		return
	}
	if isGroqCompatibleTargetV2(channelType, baseURL) {
		sanitizeGroqCompatibleRequestV2(request)
	}
	if isMistralCompatibleTargetV2(channelType, baseURL) {
		sanitizeMistralCompatibleRequestV2(request)
	}
}

func sanitizeGroqCompatibleRequestV2(request *relaymodel.GeneralOpenAIRequest) {
	sanitizeReasoningContentForProviderV2(request)
	request.LogitBias = nil
	request.Logprobs = nil
	request.TopLogprobs = nil
	if request.N > 1 {
		request.N = 1
	}
	for i := range request.Messages {
		request.Messages[i].Name = nil
	}
}

func sanitizeMistralCompatibleRequestV2(request *relaymodel.GeneralOpenAIRequest) {
	sanitizeReasoningContentForProviderV2(request)
	request.StreamOptions = nil
}

func sanitizeReasoningContentForProviderV2(request *relaymodel.GeneralOpenAIRequest) {
	for i := range request.Messages {
		request.Messages[i].ReasoningContent = nil
	}
}

func isGroqCompatibleTargetV2(channelType int, baseURL string) bool {
	if channelType == channeltype.Groq {
		return true
	}
	return strings.Contains(normalizeProviderBaseURLV2(baseURL), "api.groq.com")
}

func isMistralCompatibleTargetV2(channelType int, baseURL string) bool {
	if channelType == channeltype.Mistral {
		return true
	}
	return strings.Contains(normalizeProviderBaseURLV2(baseURL), "api.mistral.ai")
}

func normalizeProviderBaseURLV2(baseURL string) string {
	return strings.ToLower(strings.TrimSpace(baseURL))
}

func hasLowercasePrefixV2(value string, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), prefix)
}
