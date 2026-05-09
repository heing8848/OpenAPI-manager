package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func newVideoJSONContextV1(method string, target string, body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestParseVideoGenerationRequestV1TextOnly(t *testing.T) {
	c := newVideoJSONContextV1(http.MethodPost, "/v1/videos/generations", `{"model":"provider/video","content":[{"type":"text","text":"make a calm ocean video"}]}`)

	request, err := parseVideoGenerationRequestV1(c)
	if err != nil {
		t.Fatalf("expected request to be valid, got %v", err)
	}
	if request.Model != "provider/video" {
		t.Fatalf("unexpected model %q", request.Model)
	}
	if len(request.Content) != 1 || request.Content[0].Type != relaymodel.VideoContentTypeTextV1 {
		t.Fatalf("unexpected content parsed: %+v", request.Content)
	}
}

func TestParseVideoGenerationRequestV1TextAndImage(t *testing.T) {
	c := newVideoJSONContextV1(http.MethodPost, "/v1/videos/generations", `{"model":"provider/video","content":[{"type":"text","text":"animate this image"},{"type":"image_url","image_url":{"url":"https://example.com/input.png"}}]}`)

	request, err := parseVideoGenerationRequestV1(c)
	if err != nil {
		t.Fatalf("expected request to be valid, got %v", err)
	}
	if len(request.Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(request.Content))
	}
}

func TestParseVideoGenerationRequestV1RejectsMissingText(t *testing.T) {
	c := newVideoJSONContextV1(http.MethodPost, "/v1/videos/generations", `{"model":"provider/video","content":[{"type":"image_url","image_url":{"url":"https://example.com/input.png"}}]}`)

	_, err := parseVideoGenerationRequestV1(c)
	if err == nil || !strings.Contains(err.Error(), "at least one text content item is required") {
		t.Fatalf("expected missing text validation error, got %v", err)
	}
}

func TestParseVideoGenerationRequestV1RejectsUnsupportedContentType(t *testing.T) {
	c := newVideoJSONContextV1(http.MethodPost, "/v1/videos/generations", `{"model":"provider/video","content":[{"type":"text","text":"hello"},{"type":"video_url","video_url":{"url":"https://example.com/input.mp4"}}]}`)

	_, err := parseVideoGenerationRequestV1(c)
	if err == nil || !strings.Contains(err.Error(), "unsupported content type") {
		t.Fatalf("expected unsupported type validation error, got %v", err)
	}
}

func TestParseVideoGenerationRequestV1RejectsMultipleImages(t *testing.T) {
	c := newVideoJSONContextV1(http.MethodPost, "/v1/videos/generations", `{"model":"provider/video","content":[{"type":"text","text":"animate this"},{"type":"image_url","image_url":{"url":"https://example.com/1.png"}},{"type":"image_url","image_url":{"url":"https://example.com/2.png"}}]}`)

	_, err := parseVideoGenerationRequestV1(c)
	if err == nil || !strings.Contains(err.Error(), "only one image_url content item is supported") {
		t.Fatalf("expected multiple image validation error, got %v", err)
	}
}

func TestParseVideoTaskQueryIDsV1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/generations/tasks?id=task-1&id=task-2&ids=task-2,task-3", nil)

	ids, err := parseVideoTaskQueryIDsV1(c)
	if err != nil {
		t.Fatalf("expected ids to parse, got %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 unique ids, got %d (%v)", len(ids), ids)
	}
}

func TestParseVideoTaskQueryIDsV1RejectsMissingIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/generations/tasks", nil)

	_, err := parseVideoTaskQueryIDsV1(c)
	if err == nil || !strings.Contains(err.Error(), "at least one task id is required") {
		t.Fatalf("expected missing task id error, got %v", err)
	}
}
