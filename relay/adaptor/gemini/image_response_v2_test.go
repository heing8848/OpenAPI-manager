package gemini

import (
	"strings"
	"testing"
)

func TestGeminiCandidateContentV2IncludesInlineImage(t *testing.T) {
	response := &ChatResponse{
		Candidates: []ChatCandidate{
			{
				Content: ChatContent{
					Parts: []Part{
						{InlineData: &InlineData{MimeType: "image/png", Data: "abc123"}},
					},
				},
			},
		},
	}

	openaiResponse := responseGeminiChat2OpenAI(response)
	content, ok := openaiResponse.Choices[0].Message.Content.(string)
	if !ok {
		t.Fatalf("expected string content, got %T", openaiResponse.Choices[0].Message.Content)
	}
	if !strings.Contains(content, "![generated image](data:image/png;base64,abc123)") {
		t.Fatalf("expected inline image markdown, got %q", content)
	}
}

func TestGeminiResponseTextOnlyV2DoesNotCountInlineImageData(t *testing.T) {
	response := &ChatResponse{
		Candidates: []ChatCandidate{
			{
				Content: ChatContent{
					Parts: []Part{
						{Text: "Here is the image."},
						{InlineData: &InlineData{MimeType: "image/png", Data: "abc123"}},
					},
				},
			},
		},
	}

	if got := geminiResponseTextOnlyV2(response); got != "Here is the image." {
		t.Fatalf("expected text-only response, got %q", got)
	}
	if got := geminiResponseContentV2(response); strings.Contains(got, "Here is the image.\n\n![generated image]") == false {
		t.Fatalf("expected text and image response content, got %q", got)
	}
}

func TestGeminiStreamResponseV2IncludesInlineImage(t *testing.T) {
	response := &ChatResponse{
		Candidates: []ChatCandidate{
			{
				Content: ChatContent{
					Parts: []Part{
						{InlineData: &InlineData{MimeType: "image/jpeg", Data: "jpeg-data"}},
					},
				},
			},
		},
	}

	streamResponse := streamResponseGeminiChat2OpenAI(response)
	content, ok := streamResponse.Choices[0].Delta.Content.(string)
	if !ok {
		t.Fatalf("expected string delta content, got %T", streamResponse.Choices[0].Delta.Content)
	}
	if !strings.Contains(content, "data:image/jpeg;base64,jpeg-data") {
		t.Fatalf("expected inline image in stream delta, got %q", content)
	}
}
