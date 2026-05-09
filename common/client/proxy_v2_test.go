package client

import (
	"net/http"
	"testing"
)

func TestGetRelayHTTPClientByProxyV2ReturnsGlobalClientForEmptyProxy(t *testing.T) {
	originalHTTPClient := HTTPClient
	HTTPClient = &http.Client{}
	defer func() {
		HTTPClient = originalHTTPClient
	}()

	actualClient, err := GetRelayHTTPClientByProxyV2("")
	if err != nil {
		t.Fatalf("expected no error for empty proxy, got %v", err)
	}
	if actualClient != HTTPClient {
		t.Fatalf("expected empty proxy to reuse global relay client")
	}
}

func TestGetRelayHTTPClientByProxyV2RejectsInvalidProxy(t *testing.T) {
	_, err := GetRelayHTTPClientByProxyV2("://bad-proxy")
	if err == nil {
		t.Fatalf("expected invalid upstream proxy to return an error")
	}
}
