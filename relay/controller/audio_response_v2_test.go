package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
)

func newAudioHandlerContextV2() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
	return c, recorder
}

func TestRelayAudioSpeechResponseHandlerV2AcceptsBinaryAudio(t *testing.T) {
	c, recorder := newAudioHandlerContextV2()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"audio/mpeg"}},
		Body:       ioNopCloserAudioV2("FAKE-MP3-DATA"),
	}
	metaInfo := &meta.Meta{ChannelType: channeltype.OpenAICompatible, ChannelId: 12, ActualModelName: "tts-1"}

	errWithStatus := RelayAudioSpeechResponseHandlerV2(c, resp, metaInfo)
	if errWithStatus != nil {
		t.Fatalf("expected success, got %+v", errWithStatus)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code %d", recorder.Code)
	}
	if recorder.Body.String() != "FAKE-MP3-DATA" {
		t.Fatalf("unexpected body %q", recorder.Body.String())
	}
}

func TestRelayAudioSpeechResponseHandlerV2RejectsUpstreamErrorStatus(t *testing.T) {
	c, _ := newAudioHandlerContextV2()
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloserAudioV2(`{"error":{"message":"bad speech request","type":"invalid_request_error","code":"bad_request"}}`),
	}
	metaInfo := &meta.Meta{ChannelType: channeltype.OpenAICompatible, ChannelId: 12, ActualModelName: "tts-1"}

	errWithStatus := RelayAudioSpeechResponseHandlerV2(c, resp, metaInfo)
	if errWithStatus == nil {
		t.Fatalf("expected upstream error")
	}
	if errWithStatus.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status code %d", errWithStatus.StatusCode)
	}
	if errWithStatus.Message != "bad speech request" {
		t.Fatalf("unexpected message %q", errWithStatus.Message)
	}
}

func TestRelayAudioSpeechResponseHandlerV2RejectsSuccessStatusJsonError(t *testing.T) {
	c, _ := newAudioHandlerContextV2()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloserAudioV2(`{"error":{"message":"provider failed","type":"invalid_request_error","code":"provider_error"}}`),
	}
	metaInfo := &meta.Meta{ChannelType: channeltype.OpenAICompatible, ChannelId: 12, ActualModelName: "tts-1"}

	errWithStatus := RelayAudioSpeechResponseHandlerV2(c, resp, metaInfo)
	if errWithStatus == nil {
		t.Fatalf("expected embedded error payload to fail")
	}
	if errWithStatus.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected status code %d", errWithStatus.StatusCode)
	}
	if errWithStatus.Message != "provider failed" {
		t.Fatalf("unexpected message %q", errWithStatus.Message)
	}
}

func ioNopCloserAudioV2(body string) *readCloserAudioV2 {
	return &readCloserAudioV2{Reader: strings.NewReader(body)}
}

type readCloserAudioV2 struct {
	*strings.Reader
}

func (r *readCloserAudioV2) Close() error {
	return nil
}
