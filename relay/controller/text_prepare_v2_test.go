package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func TestBuildPreparedTextRequestV2SanitizesGeminiLikeOpenAICompatiblePayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{
		"model":"google-gemini-3.1-flash-lite-preview",
		"stream":true,
		"messages":[{"role":"user","content":"hello"}],
		"tools":[
			{
				"type":"function",
				"function":{
					"name":"builtin_web_search",
					"parameters":{
						"type":"object",
						"additionalProperties":false
					}
				}
			}
		]
	}`

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/worker/prepare_v2", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer upstream")
	c.Set(ctxkey.Channel, channeltype.OpenAICompatible)
	c.Set(ctxkey.ChannelId, 20)
	c.Set(ctxkey.ChannelKeyId, 3)
	c.Set(ctxkey.ChannelKeyIndex, 1)
	c.Set(ctxkey.BaseURL, "https://generativelanguage.googleapis.com/v1beta/openai/")
	c.Set(ctxkey.Group, "default")
	c.Set(ctxkey.Id, 1)
	c.Set(ctxkey.TokenId, 2)
	c.Set(ctxkey.TokenName, "edge token")
	c.Set(ctxkey.Config, model.ChannelConfig{})

	prepared, bizErr := BuildPreparedTextRequestV2(c, "/v1/chat/completions")
	if bizErr != nil {
		t.Fatalf("expected prepare to succeed, got %+v", bizErr)
	}
	if prepared.TargetURL != "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions" {
		t.Fatalf("unexpected target url %q", prepared.TargetURL)
	}
	if !prepared.IsStream {
		t.Fatalf("expected stream mode to stay enabled")
	}

	var preparedRequest relaymodel.GeneralOpenAIRequest
	if err := json.Unmarshal(prepared.Body, &preparedRequest); err != nil {
		t.Fatalf("failed to decode prepared request: %v", err)
	}
	if preparedRequest.StreamOptions == nil || !preparedRequest.StreamOptions.IncludeUsage {
		t.Fatalf("expected prepared request to include stream usage")
	}

	parameters, ok := preparedRequest.Tools[0].Function.Parameters.(map[string]any)
	if !ok {
		t.Fatalf("expected tool parameters map, got %#v", preparedRequest.Tools[0].Function.Parameters)
	}
	if _, exists := parameters["additionalProperties"]; exists {
		t.Fatalf("expected Gemini-like payload to drop additionalProperties, got %#v", parameters)
	}
}
