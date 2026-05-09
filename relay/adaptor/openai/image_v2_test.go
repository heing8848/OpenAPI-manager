package openai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newImageHandlerContextV2() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	return c, recorder
}

func TestImageHandlerV2AcceptsValidOpenAIStylePayload(t *testing.T) {
	c, recorder := newImageHandlerContextV2()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloserV2(`{"created":171,"data":[{"url":"https://example.com/test.png"}]}`),
	}

	errWithStatus, _ := ImageHandlerV2(c, resp)
	if errWithStatus != nil {
		t.Fatalf("expected success, got %+v", errWithStatus)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "test.png") {
		t.Fatalf("expected response body to be copied through, got %s", recorder.Body.String())
	}
}

func TestImageHandlerV2RejectsUpstreamErrorStatus(t *testing.T) {
	c, _ := newImageHandlerContextV2()
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloserV2(`{"error":{"message":"bad image request","type":"invalid_request_error","code":"bad_request"}}`),
	}

	errWithStatus, _ := ImageHandlerV2(c, resp)
	if errWithStatus == nil {
		t.Fatalf("expected upstream error")
	}
	if errWithStatus.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status code %d", errWithStatus.StatusCode)
	}
	if errWithStatus.Message != "bad image request" {
		t.Fatalf("unexpected message %q", errWithStatus.Message)
	}
}

func TestImageHandlerV2RejectsEmbeddedErrorPayloadOnSuccessStatus(t *testing.T) {
	c, _ := newImageHandlerContextV2()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloserV2(`{"error":{"message":"provider failed","type":"invalid_request_error","code":"provider_error"}}`),
	}

	errWithStatus, _ := ImageHandlerV2(c, resp)
	if errWithStatus == nil {
		t.Fatalf("expected error for embedded error payload")
	}
	if errWithStatus.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected status code %d", errWithStatus.StatusCode)
	}
	if errWithStatus.Message != "provider failed" {
		t.Fatalf("unexpected message %q", errWithStatus.Message)
	}
}

func TestImageHandlerV2RejectsMalformedPayload(t *testing.T) {
	c, _ := newImageHandlerContextV2()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloserV2(`{"created":171,"data":[]}`),
	}

	errWithStatus, _ := ImageHandlerV2(c, resp)
	if errWithStatus == nil {
		t.Fatalf("expected malformed payload error")
	}
	if errWithStatus.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected status code %d", errWithStatus.StatusCode)
	}
	if errWithStatus.Code != "bad_image_response" {
		t.Fatalf("unexpected error code %v", errWithStatus.Code)
	}
}

func ioNopCloserV2(body string) *readCloserV2 {
	return &readCloserV2{Reader: strings.NewReader(body)}
}

type readCloserV2 struct {
	*strings.Reader
}

func (r *readCloserV2) Close() error {
	return nil
}
