package openai

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
)

func TestGetFullRequestURLV2AvoidsDuplicateV1Segments(t *testing.T) {
	got := GetFullRequestURLV2("https://api.groq.com/openai/v1", "/v1/chat/completions", channeltype.Groq)
	want := "https://api.groq.com/openai/v1/chat/completions"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestGetFullRequestURLV2AvoidsDuplicateV2Segments(t *testing.T) {
	got := GetFullRequestURLV2("https://api.cohere.com/v2", "/v2/chat", channeltype.Cohere)
	want := "https://api.cohere.com/v2/chat"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
