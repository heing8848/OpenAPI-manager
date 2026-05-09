package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

func TestResolveOpenAICompatibleModelListEndpointV2(t *testing.T) {
	testCases := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "sdk base url already includes v1",
			baseURL:  "https://example.com/v1",
			expected: "https://example.com/v1/models",
		},
		{
			name:     "plain host base url appends v1",
			baseURL:  "https://example.com",
			expected: "https://example.com/v1/models",
		},
		{
			name:     "trailing slash is normalized",
			baseURL:  "https://example.com/v1/",
			expected: "https://example.com/v1/models",
		},
		{
			name:     "openai compatibility suffix does not append extra v1",
			baseURL:  "https://example.com/v1beta/openai",
			expected: "https://example.com/v1beta/openai/models",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := resolveOpenAICompatibleModelListEndpointV2(testCase.baseURL)
			if actual != testCase.expected {
				t.Fatalf("expected %q, got %q", testCase.expected, actual)
			}
		})
	}
}

func TestDiscoverOpenAICompatibleModelsSupportsSDKBaseURLV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o-mini"},{"id":"gpt-4.1"}]}`))
	}))
	defer server.Close()

	previousClient := client.ImpatientHTTPClient
	client.ImpatientHTTPClient = &http.Client{}
	defer func() {
		client.ImpatientHTTPClient = previousClient
	}()

	request := &ChannelModelDiscoverRequest{
		Type:    channeltype.OpenAICompatible,
		BaseURL: server.URL + "/v1",
	}

	models, endpoint, err := discoverOpenAICompatibleModelsV2(request, []string{"test-key"})
	if err != nil {
		t.Fatalf("expected discovery to succeed, got error: %v", err)
	}
	if endpoint != server.URL+"/v1/models" {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "gpt-4o-mini" || models[1] != "gpt-4.1" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestDiscoverOpenAICompatibleModelsWithoutKeyV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if authorization := r.Header.Get("Authorization"); authorization != "" {
			http.Error(w, "authorization not expected", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"public-model"}]}`))
	}))
	defer server.Close()

	previousClient := client.ImpatientHTTPClient
	client.ImpatientHTTPClient = &http.Client{}
	defer func() {
		client.ImpatientHTTPClient = previousClient
	}()

	request := &ChannelModelDiscoverRequest{
		Type:    channeltype.OpenAICompatible,
		BaseURL: server.URL + "/v1",
	}

	models, endpoint, err := discoverOpenAICompatibleModelsV2(request, nil)
	if err != nil {
		t.Fatalf("expected discovery to succeed, got error: %v", err)
	}
	if endpoint != server.URL+"/v1/models" {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0] != "public-model" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestDiscoverGeminiModelsV2PrefersV1BetaAndHeaderAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("pageSize") != "1000" {
			http.Error(w, "pageSize missing", http.StatusBadRequest)
			return
		}
		if r.Header.Get("x-goog-api-key") != "test-key" {
			http.Error(w, "missing api key header", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-2.5-flash"},{"name":"models/gemini-2.5-pro"}]}`))
	}))
	defer server.Close()

	previousClient := client.ImpatientHTTPClient
	client.ImpatientHTTPClient = &http.Client{}
	defer func() {
		client.ImpatientHTTPClient = previousClient
	}()

	previousVersion := config.GeminiVersion
	config.GeminiVersion = "v1"
	defer func() {
		config.GeminiVersion = previousVersion
	}()

	request := &ChannelModelDiscoverRequest{
		Type:    channeltype.Gemini,
		BaseURL: server.URL,
	}

	models, endpoint, err := discoverGeminiModelsV2(request, []string{"test-key"})
	if err != nil {
		t.Fatalf("expected discovery to succeed, got error: %v", err)
	}
	if endpoint != server.URL+"/v1beta/models?pageSize=1000" {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "gemini-2.5-flash" || models[1] != "gemini-2.5-pro" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestDiscoverChannelModelsUsesOpenAICompatibleStrategyForGeminiOpenAICompatibleV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/openai/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gemini-2.5-flash-preview"}]}`))
	}))
	defer server.Close()

	previousClient := client.ImpatientHTTPClient
	client.ImpatientHTTPClient = &http.Client{}
	defer func() {
		client.ImpatientHTTPClient = previousClient
	}()

	result := discoverChannelModels(&ChannelModelDiscoverRequest{
		Type:    channeltype.GeminiOpenAICompatible,
		BaseURL: server.URL + "/v1beta/openai",
	}, []string{"test-key"})

	if result.Source != "dynamic" {
		t.Fatalf("expected dynamic source, got %#v", result)
	}
	if result.DebugEndpoint != server.URL+"/v1beta/openai/models" {
		t.Fatalf("unexpected endpoint: %s", result.DebugEndpoint)
	}
	if len(result.Models) != 1 || result.Models[0] != "gemini-2.5-flash-preview" {
		t.Fatalf("unexpected models: %#v", result.Models)
	}
}
