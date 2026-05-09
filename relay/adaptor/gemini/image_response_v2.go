package gemini

import (
	"fmt"
	"strings"
)

func geminiResponseTextOnlyV2(response *ChatResponse) string {
	if response == nil || len(response.Candidates) == 0 {
		return ""
	}
	return geminiCandidateTextOnlyV2(&response.Candidates[0])
}

func geminiResponseContentV2(response *ChatResponse) string {
	if response == nil || len(response.Candidates) == 0 {
		return ""
	}
	return geminiCandidateContentV2(&response.Candidates[0])
}

func geminiCandidateTextOnlyV2(candidate *ChatCandidate) string {
	if candidate == nil {
		return ""
	}
	var builder strings.Builder
	hasContent := false
	for _, part := range candidate.Content.Parts {
		appendGeminiResponseSegmentV2(&builder, &hasContent, part.Text)
	}
	return builder.String()
}

func geminiCandidateContentV2(candidate *ChatCandidate) string {
	if candidate == nil {
		return ""
	}
	var builder strings.Builder
	hasContent := false
	for _, part := range candidate.Content.Parts {
		appendGeminiResponseSegmentV2(&builder, &hasContent, part.Text)
		appendGeminiResponseSegmentV2(&builder, &hasContent, geminiInlineDataMarkdownV2(part.InlineData))
	}
	return builder.String()
}

func appendGeminiResponseSegmentV2(builder *strings.Builder, hasContent *bool, segment string) {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return
	}
	if *hasContent {
		builder.WriteString("\n\n")
	}
	builder.WriteString(segment)
	*hasContent = true
}

func geminiInlineDataMarkdownV2(inlineData *InlineData) string {
	if inlineData == nil {
		return ""
	}
	data := strings.TrimSpace(inlineData.Data)
	if data == "" {
		return ""
	}
	mimeType := strings.TrimSpace(inlineData.MimeType)
	if mimeType == "" {
		mimeType = "image/png"
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, data)
	if strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return fmt.Sprintf("![generated image](%s)", dataURL)
	}
	return fmt.Sprintf("[generated file](%s)", dataURL)
}
