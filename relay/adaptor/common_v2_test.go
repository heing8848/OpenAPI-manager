package adaptor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/meta"
)

func TestSetupCommonRequestHeaderV2SetsDefaultAcceptAndUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}

	SetupCommonRequestHeader(ctx, req, &meta.Meta{})

	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("expected default accept application/json, got %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != defaultUpstreamUserAgentV2 {
		t.Fatalf("expected default upstream user agent, got %q", got)
	}
}

func TestSetupCommonRequestHeaderV2PreservesStreamAccept(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}

	SetupCommonRequestHeader(ctx, req, &meta.Meta{IsStream: true})

	if got := req.Header.Get("Accept"); got != "text/event-stream" {
		t.Fatalf("expected stream accept text/event-stream, got %q", got)
	}
}
