package controller

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRelayErrorHandlerV2PreservesRawTextErrorBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader("geo access denied by upstream")),
		Header:     make(http.Header),
	}

	errInfo := RelayErrorHandlerV2(resp)
	if errInfo.Error.Message != "geo access denied by upstream" {
		t.Fatalf("expected raw error text to be preserved, got %q", errInfo.Error.Message)
	}
}

func TestRelayErrorHandlerV2StripsHtmlNoise(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader("<html><body><h1>Forbidden</h1><p>Region blocked</p></body></html>")),
		Header:     make(http.Header),
	}

	errInfo := RelayErrorHandlerV2(resp)
	if !strings.Contains(errInfo.Error.Message, "Forbidden") || !strings.Contains(errInfo.Error.Message, "Region blocked") {
		t.Fatalf("expected html body text to be extracted, got %q", errInfo.Error.Message)
	}
}

func TestRelayErrorHandlerV2NormalizesCloudflareChallenge(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body: io.NopCloser(strings.NewReader(`
			<html>
				<head>
					<style>*{box-sizing:border-box}</style>
				</head>
				<body>
					<h1>Just a moment...</h1>
					<p>Enable JavaScript and cookies to continue</p>
					<script>window._cf_chl_opt = { cZone: 'api.groq.com' };</script>
				</body>
			</html>
		`)),
		Header: make(http.Header),
	}

	errInfo := RelayErrorHandlerV2(resp)
	if errInfo.Error.Code != "upstream_cloudflare_challenge" {
		t.Fatalf("expected cloudflare challenge code, got %q", errInfo.Error.Code)
	}
	if !strings.Contains(errInfo.Error.Message, "api.groq.com") {
		t.Fatalf("expected challenge message to mention upstream host, got %q", errInfo.Error.Message)
	}
	if strings.Contains(errInfo.Error.Message, "box-sizing") || strings.Contains(errInfo.Error.Message, "_cf_chl_opt") {
		t.Fatalf("expected challenge message to be normalized, got %q", errInfo.Error.Message)
	}
}
