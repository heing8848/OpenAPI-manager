package controller

import (
	"net/http"
	"testing"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func TestBuildWorkerRelayErrorV2AppendsRequestIDForPlainTextErrors(t *testing.T) {
	errInfo := buildWorkerRelayErrorV2(WorkerCallbackRequestV2{
		StatusCode: 403,
		ErrorMsg:   "Access denied",
		RequestID:  "req-123",
	})

	if errInfo.Error.Type != "upstream_error" {
		t.Fatalf("expected default error type, got %q", errInfo.Error.Type)
	}
	if errInfo.Error.Code != "bad_response_status_code" {
		t.Fatalf("expected default error code, got %q", errInfo.Error.Code)
	}
	if errInfo.Error.Message != "Access denied (request id: req-123)" {
		t.Fatalf("unexpected error message %q", errInfo.Error.Message)
	}
}

func TestRenderWorkerErrorDefaults(t *testing.T) {
	if got := appendWorkerRequestIDV2("", "req-abc"); got != "(request id: req-abc)" {
		t.Fatalf("expected request id wrapper, got %q", got)
	}
}

func TestParseWorkerRelayModeV2RejectsUnsupportedMode(t *testing.T) {
	_, bizErr := parseWorkerRelayModeV2("/v1/images/generations")
	if bizErr == nil {
		t.Fatalf("expected unsupported mode error")
	}
	if bizErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", bizErr.StatusCode)
	}
}

func TestBuildWorkerRelayErrorV2KeepsProvidedTypeAndCode(t *testing.T) {
	errInfo := buildWorkerRelayErrorV2(WorkerCallbackRequestV2{
		StatusCode: 422,
		ErrorMsg:   "bad response status code 422",
		ErrorType:  "upstream_error",
		ErrorCode:  "bad_response_status_code",
	})

	want := relaymodel.ErrorWithStatusCode{
		StatusCode: 422,
		Error: relaymodel.Error{
			Message: "bad response status code 422",
			Type:    "upstream_error",
			Code:    "bad_response_status_code",
			Param:   "422",
		},
	}
	if errInfo != want {
		t.Fatalf("unexpected error info: %#v", errInfo)
	}
}
