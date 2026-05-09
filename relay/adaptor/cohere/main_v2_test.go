package cohere

import (
	"testing"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func TestConvertRequestV2UsesMessagesFormat(t *testing.T) {
	temp := 0.7
	request := ConvertRequestV2(relaymodel.GeneralOpenAIRequest{
		Model:       "command-r",
		Temperature: &temp,
		Messages: []relaymodel.Message{
			{Role: "system", Content: "be concise"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	})

	if len(request.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(request.Messages))
	}
	if request.Messages[0].Role != "system" || request.Messages[0].Content != "be concise" {
		t.Fatalf("unexpected first message: %+v", request.Messages[0])
	}
	if request.Messages[2].Role != "assistant" {
		t.Fatalf("expected assistant role to be preserved, got %+v", request.Messages[2])
	}
}
