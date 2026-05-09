package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/monitor"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func TestProcessChannelRelayErrorRecordsFailuresForGenericUpstreamErrors(t *testing.T) {
	monitor.ClearChannelFailures(42)
	processChannelRelayError(context.Background(), 7, 42, "groq", relaymodel.ErrorWithStatusCode{
		StatusCode: 422,
		Error: relaymodel.Error{
			Message: "bad response status code 422",
			Type:    "upstream_error",
			Code:    "bad_response_status_code",
		},
	})

	if got := monitor.GetChannelFailures(42); got != 1 {
		t.Fatalf("expected channel failures to increment for generic relay errors, got %d", got)
	}
}

func TestBuildFinalRelayFailureLogV2CapturesChannelAndModelDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(ctxkey.Id, 1)
	c.Set(ctxkey.TokenName, "demo-token")
	c.Set(ctxkey.ChannelId, 9)
	c.Set(ctxkey.ChannelKeyId, 12)
	c.Set(ctxkey.ChannelKeyIndex, 1)
	c.Set(ctxkey.RequestModel, "groq-openai/gpt-oss-120b")

	logRecord := buildFinalRelayFailureLogV2(c, &relaymodel.ErrorWithStatusCode{
		StatusCode: 403,
		Error: relaymodel.Error{
			Message: "permission denied",
			Type:    "upstream_error",
			Code:    "bad_response_status_code",
		},
	})

	if logRecord == nil {
		t.Fatalf("expected failure consume log to be built")
	}
	if logRecord.ModelName != "groq-openai/gpt-oss-120b" {
		t.Fatalf("unexpected model name %q", logRecord.ModelName)
	}
	if logRecord.ChannelId != 9 || logRecord.ChannelKeyId != 12 || logRecord.ChannelKeyIndex != 1 {
		t.Fatalf("unexpected channel metadata in log: %+v", logRecord)
	}
	if logRecord.Content == "" {
		t.Fatalf("expected failure log content")
	}
}
