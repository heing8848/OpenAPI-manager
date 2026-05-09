package openai

import (
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/model"
)

func ResponseText2Usage(responseText string, modelName string, promptTokens int) *model.Usage {
	usage := &model.Usage{}
	usage.PromptTokens = promptTokens
	usage.CompletionTokens = CountTokenText(responseText, modelName)
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage
}

func GetFullRequestURL(baseURL string, requestURL string, channelType int) string {
	if channelType == channeltype.OpenAICompatible {
		return fmt.Sprintf("%s%s", strings.TrimSuffix(baseURL, "/"), strings.TrimPrefix(requestURL, "/v1"))
	}
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, requestURL)

	if strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") {
		switch channelType {
		case channeltype.OpenAI:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/v1"))
		case channeltype.Azure:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/openai/deployments"))
		}
	}
	return fullRequestURL
}

func GetFullRequestURLV2(baseURL string, requestURL string, channelType int) string {
	baseURL = strings.TrimSpace(baseURL)
	requestURL = strings.TrimSpace(requestURL)
	if channelType == channeltype.OpenAICompatible {
		return fmt.Sprintf("%s%s", strings.TrimSuffix(baseURL, "/"), strings.TrimPrefix(requestURL, "/v1"))
	}

	normalizedBaseURL, normalizedRequestURL := normalizeVersionedProviderURLV2(baseURL, requestURL)
	fullRequestURL := fmt.Sprintf("%s%s", normalizedBaseURL, normalizedRequestURL)

	if strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") {
		switch channelType {
		case channeltype.OpenAI:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/v1"))
		case channeltype.Azure:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/openai/deployments"))
		}
	}
	return fullRequestURL
}

func normalizeVersionedProviderURLV2(baseURL string, requestURL string) (string, string) {
	trimmedBaseURL := strings.TrimSuffix(baseURL, "/")
	normalizedRequestURL := requestURL
	lowerBaseURL := strings.ToLower(trimmedBaseURL)

	switch {
	case strings.HasSuffix(lowerBaseURL, "/v1") && strings.HasPrefix(normalizedRequestURL, "/v1/"):
		normalizedRequestURL = strings.TrimPrefix(normalizedRequestURL, "/v1")
	case strings.HasSuffix(lowerBaseURL, "/v2") && strings.HasPrefix(normalizedRequestURL, "/v2/"):
		normalizedRequestURL = strings.TrimPrefix(normalizedRequestURL, "/v2")
	}

	return trimmedBaseURL, normalizedRequestURL
}
