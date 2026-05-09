package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

type ChannelModelDiscoverRequest struct {
	Type    int            `json:"type"`
	BaseURL string         `json:"base_url"`
	Key     string         `json:"key"`
	Keys    []string       `json:"keys"`
	Config  map[string]any `json:"config"`
}

type ChannelModelDiscoverResult struct {
	Models        []string `json:"models"`
	Source        string   `json:"source"`
	Message       string   `json:"message"`
	DebugEndpoint string   `json:"debug_endpoint,omitempty"`
	DebugError    string   `json:"debug_error,omitempty"`
}

type openAIModelListResponse struct {
	Data []struct {
		Id string `json:"id"`
	} `json:"data"`
}

type geminiModelListResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type geminiDiscoveryAttemptV2 struct {
	Endpoint      string
	DebugEndpoint string
	Headers       http.Header
}

type channelModelDiscoverStrategy func(*ChannelModelDiscoverRequest, []string) ([]string, string, error)

var channelModelDiscoverStrategies = map[int]channelModelDiscoverStrategy{}

func init() {
	for _, channelType := range []int{
		channeltype.OpenAI,
		channeltype.API2D,
		channeltype.CloseAI,
		channeltype.OpenAISB,
		channeltype.OpenAIMax,
		channeltype.OhMyGPT,
		channeltype.Custom,
		channeltype.Ails,
		channeltype.AIProxy,
		channeltype.API2GPT,
		channeltype.AIGC2D,
		channeltype.OpenRouter,
		channeltype.FastGPT,
		channeltype.Moonshot,
		channeltype.Baichuan,
		channeltype.Minimax,
		channeltype.Mistral,
		channeltype.Groq,
		channeltype.LingYiWanWu,
		channeltype.StepFun,
		channeltype.DeepSeek,
		channeltype.TogetherAI,
		channeltype.Doubao,
		channeltype.Novita,
		channeltype.SiliconFlow,
		channeltype.XAI,
		channeltype.BaiduV2,
		channeltype.XunfeiV2,
		channeltype.AliBailian,
		channeltype.OpenAICompatible,
		channeltype.GeminiOpenAICompatible,
		channeltype.AI360,
	} {
		channelModelDiscoverStrategies[channelType] = discoverOpenAICompatibleModelsV2
	}
	for _, channelType := range []int{
		channeltype.Gemini,
		channeltype.PaLM,
	} {
		channelModelDiscoverStrategies[channelType] = discoverGeminiModelsV2
	}
	channelModelDiscoverStrategies[channeltype.Ollama] = discoverOllamaModels
}

func DiscoverChannelModels(c *gin.Context) {
	request := ChannelModelDiscoverRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	keys := model.NormalizeChannelKeyValues(request.Keys)
	if len(keys) == 0 {
		keys = model.SplitLegacyChannelKeys(request.Key)
	}

	result := discoverChannelModels(&request, keys)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

func discoverChannelModels(request *ChannelModelDiscoverRequest, keys []string) ChannelModelDiscoverResult {
	if request.Type == channeltype.VideoTaskV1 {
		return ChannelModelDiscoverResult{
			Models:  []string{},
			Source:  "manual_only",
			Message: "video task channels require manual model entry in V1",
		}
	}

	strategy, ok := channelModelDiscoverStrategies[request.Type]
	if !ok {
		return fallbackDiscoveredModelsV2(request.Type, "provider has no dynamic discovery strategy yet", "", "")
	}

	models, endpoint, err := strategy(request, keys)
	if err != nil {
		return fallbackDiscoveredModelsV2(request.Type, err.Error(), endpoint, err.Error())
	}
	if len(models) == 0 {
		return fallbackDiscoveredModelsV2(request.Type, "provider returned an empty model list", endpoint, "")
	}
	return ChannelModelDiscoverResult{
		Models:        normalizeDiscoveredModels(models),
		Source:        "dynamic",
		Message:       "models loaded from upstream model list",
		DebugEndpoint: endpoint,
	}
}

func discoverOpenAICompatibleModels(request *ChannelModelDiscoverRequest, keys []string) ([]string, string, error) {
	baseURL := strings.TrimRight(resolveDiscoveryBaseURL(request), "/")
	if baseURL == "" {
		return nil, "", fmt.Errorf("base_url is required for this provider")
	}
	endpoint := resolveOpenAICompatibleModelListEndpointV2(baseURL)
	var lastErr error
	for _, key := range keys {
		body, err := doModelDiscoveryRequest(http.MethodGet, endpoint, http.Header{
			"Authorization": []string{"Bearer " + key},
		})
		if err != nil {
			lastErr = err
			continue
		}
		response := openAIModelListResponse{}
		if err = json.Unmarshal(body, &response); err != nil {
			lastErr = err
			continue
		}
		models := make([]string, 0, len(response.Data))
		for _, modelData := range response.Data {
			if modelData.Id != "" {
				models = append(models, modelData.Id)
			}
		}
		if len(models) > 0 {
			return models, endpoint, nil
		}
	}
	if lastErr != nil {
		return nil, endpoint, fmt.Errorf("failed to fetch models from %s: %w", endpoint, lastErr)
	}
	return nil, endpoint, fmt.Errorf("failed to fetch models from %s", endpoint)
}

func discoverOpenAICompatibleModelsV2(request *ChannelModelDiscoverRequest, keys []string) ([]string, string, error) {
	baseURL := strings.TrimRight(resolveDiscoveryBaseURL(request), "/")
	if baseURL == "" {
		return nil, "", fmt.Errorf("base_url is required for this provider")
	}
	endpoint := resolveOpenAICompatibleModelListEndpointV2(baseURL)
	var headersList []http.Header
	for _, key := range keys {
		headersList = append(headersList, http.Header{
			"Authorization": []string{"Bearer " + key},
		})
	}
	if len(headersList) == 0 {
		headersList = append(headersList, nil)
	}
	var lastErr error
	for _, headers := range headersList {
		models, err := fetchOpenAICompatibleModelsV2(endpoint, headers)
		if err != nil {
			lastErr = err
			continue
		}
		if len(models) > 0 {
			return models, endpoint, nil
		}
	}
	if lastErr != nil {
		return nil, endpoint, fmt.Errorf("failed to fetch models from %s: %w", endpoint, lastErr)
	}
	return nil, endpoint, fmt.Errorf("failed to fetch models from %s", endpoint)
}

func fetchOpenAICompatibleModelsV2(endpoint string, headers http.Header) ([]string, error) {
	body, err := doModelDiscoveryRequest(http.MethodGet, endpoint, headers)
	if err != nil {
		return nil, err
	}
	response := openAIModelListResponse{}
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(response.Data))
	for _, modelData := range response.Data {
		if modelData.Id != "" {
			models = append(models, modelData.Id)
		}
	}
	return models, nil
}

func discoverGeminiModels(request *ChannelModelDiscoverRequest, keys []string) ([]string, string, error) {
	baseURL := strings.TrimRight(resolveDiscoveryBaseURL(request), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	version := config.GeminiVersion
	if version == "" {
		version = "v1"
	}
	var lastErr error
	lastEndpoint := baseURL
	for _, key := range keys {
		endpoint := fmt.Sprintf("%s/%s/models?key=%s", baseURL, version, url.QueryEscape(key))
		lastEndpoint = endpoint
		body, err := doModelDiscoveryRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		response := geminiModelListResponse{}
		if err = json.Unmarshal(body, &response); err != nil {
			lastErr = err
			continue
		}
		models := make([]string, 0, len(response.Models))
		for _, modelData := range response.Models {
			modelName := strings.TrimPrefix(modelData.Name, "models/")
			if modelName != "" {
				models = append(models, modelName)
			}
		}
		if len(models) > 0 {
			return models, endpoint, nil
		}
	}
	if lastErr != nil {
		return nil, lastEndpoint, fmt.Errorf("failed to fetch gemini models from %s: %w", lastEndpoint, lastErr)
	}
	return nil, lastEndpoint, fmt.Errorf("failed to fetch gemini models from %s", lastEndpoint)
}

func discoverGeminiModelsV2(request *ChannelModelDiscoverRequest, keys []string) ([]string, string, error) {
	baseURL := strings.TrimRight(resolveDiscoveryBaseURL(request), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	attempts := buildGeminiModelDiscoveryAttemptsV2(baseURL, keys)
	var lastErr error
	lastEndpoint := baseURL
	for _, attempt := range attempts {
		lastEndpoint = attempt.DebugEndpoint
		models, err := fetchGeminiModelsV2(attempt.Endpoint, attempt.Headers)
		if err != nil {
			lastErr = err
			continue
		}
		if len(models) > 0 {
			return models, attempt.DebugEndpoint, nil
		}
	}
	if lastErr != nil {
		return nil, lastEndpoint, fmt.Errorf("failed to fetch gemini models from %s: %w", lastEndpoint, lastErr)
	}
	return nil, lastEndpoint, fmt.Errorf("failed to fetch gemini models from %s", lastEndpoint)
}

func buildGeminiModelDiscoveryAttemptsV2(baseURL string, keys []string) []geminiDiscoveryAttemptV2 {
	endpoints := resolveGeminiModelListEndpointsV2(baseURL)
	attempts := make([]geminiDiscoveryAttemptV2, 0, len(endpoints)*maxIntV2(1, len(keys))*2)
	if len(keys) == 0 {
		for _, endpoint := range endpoints {
			attempts = append(attempts, geminiDiscoveryAttemptV2{
				Endpoint:      endpoint,
				DebugEndpoint: endpoint,
			})
		}
		return attempts
	}
	for _, key := range keys {
		for _, endpoint := range endpoints {
			attempts = append(attempts, geminiDiscoveryAttemptV2{
				Endpoint:      endpoint,
				DebugEndpoint: endpoint,
				Headers: http.Header{
					"x-goog-api-key": []string{key},
				},
			})
			attempts = append(attempts, geminiDiscoveryAttemptV2{
				Endpoint:      endpoint + "&key=" + url.QueryEscape(key),
				DebugEndpoint: endpoint,
			})
		}
	}
	return attempts
}

func maxIntV2(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func resolveGeminiModelListEndpointsV2(baseURL string) []string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	if strings.HasSuffix(baseURL, "/models") {
		return []string{appendGeminiModelListQueryV2(baseURL)}
	}
	if strings.HasSuffix(baseURL, "/v1") || strings.HasSuffix(baseURL, "/v1beta") {
		return []string{appendGeminiModelListQueryV2(baseURL + "/models")}
	}
	versions := uniqueGeminiDiscoveryVersionsV2(config.GeminiVersion, "v1beta", "v1")
	endpoints := make([]string, 0, len(versions))
	for _, version := range versions {
		endpoints = append(endpoints, appendGeminiModelListQueryV2(baseURL+"/"+version+"/models"))
	}
	return endpoints
}

func appendGeminiModelListQueryV2(endpoint string) string {
	if strings.Contains(endpoint, "?") {
		return endpoint
	}
	return endpoint + "?pageSize=1000"
}

func uniqueGeminiDiscoveryVersionsV2(versions ...string) []string {
	seen := make(map[string]bool)
	unique := make([]string, 0, len(versions))
	for _, version := range versions {
		version = strings.Trim(strings.TrimSpace(version), "/")
		if version == "" || seen[version] {
			continue
		}
		seen[version] = true
		unique = append(unique, version)
	}
	return unique
}

func fetchGeminiModelsV2(endpoint string, headers http.Header) ([]string, error) {
	body, err := doModelDiscoveryRequest(http.MethodGet, endpoint, headers)
	if err != nil {
		return nil, err
	}
	response := geminiModelListResponse{}
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(response.Models))
	for _, modelData := range response.Models {
		modelName := strings.TrimPrefix(modelData.Name, "models/")
		if modelName != "" {
			models = append(models, modelName)
		}
	}
	return models, nil
}

func discoverOllamaModels(request *ChannelModelDiscoverRequest, keys []string) ([]string, string, error) {
	baseURL := strings.TrimRight(resolveDiscoveryBaseURL(request), "/")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	endpoint := baseURL + "/api/tags"
	body, err := doModelDiscoveryRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, endpoint, err
	}
	response := ollamaTagsResponse{}
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, endpoint, err
	}
	models := make([]string, 0, len(response.Models))
	for _, modelData := range response.Models {
		if modelData.Name != "" {
			models = append(models, modelData.Name)
		}
	}
	return models, endpoint, nil
}

func fallbackDiscoveredModelsV2(channelType int, reason string, endpoint string, debugError string) ChannelModelDiscoverResult {
	models := channelId2Models[channelType]
	if len(models) > 0 {
		return ChannelModelDiscoverResult{
			Models:        normalizeDiscoveredModels(models),
			Source:        "fallback_static",
			Message:       fmt.Sprintf("dynamic discovery failed, using static fallback: %s", reason),
			DebugEndpoint: endpoint,
			DebugError:    debugError,
		}
	}
	return ChannelModelDiscoverResult{
		Models:        []string{},
		Source:        "manual_only",
		Message:       fmt.Sprintf("dynamic discovery unavailable, please input models manually: %s", reason),
		DebugEndpoint: endpoint,
		DebugError:    debugError,
	}
}

func resolveOpenAICompatibleModelListEndpointV2(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return ""
	}
	if strings.HasSuffix(baseURL, "/v1") || strings.HasSuffix(baseURL, "/openai") {
		return baseURL + "/models"
	}
	return baseURL + "/v1/models"
}

func resolveDiscoveryBaseURL(request *ChannelModelDiscoverRequest) string {
	if strings.TrimSpace(request.BaseURL) != "" {
		return strings.TrimSpace(request.BaseURL)
	}
	if request.Type >= 0 && request.Type < len(channeltype.ChannelBaseURLs) {
		return channeltype.ChannelBaseURLs[request.Type]
	}
	return ""
}

func doModelDiscoveryRequest(method string, endpoint string, headers http.Header) ([]byte, error) {
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	resp, err := client.ImpatientHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func normalizeDiscoveredModels(models []string) []string {
	seen := make(map[string]bool)
	normalized := make([]string, 0, len(models))
	for _, modelName := range models {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" || seen[modelName] {
			continue
		}
		seen[modelName] = true
		normalized = append(normalized, modelName)
	}
	sort.Strings(normalized)
	return normalized
}
