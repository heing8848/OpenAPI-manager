package cohere

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/render"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/model"
)

func ConvertRequestV2(textRequest model.GeneralOpenAIRequest) *RequestV2 {
	request := &RequestV2{
		Model:            textRequest.Model,
		Stream:           textRequest.Stream,
		MaxTokens:        textRequest.MaxTokens,
		Temperature:      textRequest.Temperature,
		P:                textRequest.TopP,
		K:                textRequest.TopK,
		Seed:             int(textRequest.Seed),
		FrequencyPenalty: textRequest.FrequencyPenalty,
		PresencePenalty:  textRequest.PresencePenalty,
		StopSequences:    extractStopSequencesV2(textRequest.Stop),
	}
	if request.Model == "" {
		request.Model = "command-r"
	}
	if textRequest.ResponseFormat != nil {
		request.ResponseFormat = &ResponseFormatV2{
			Type:       textRequest.ResponseFormat.Type,
			JsonSchema: textRequest.ResponseFormat.JsonSchema,
		}
	}
	for _, message := range textRequest.Messages {
		content := strings.TrimSpace(message.StringContent())
		if content == "" {
			continue
		}
		request.Messages = append(request.Messages, MessageV2{
			Role:    normalizeCohereRoleV2(message.Role),
			Content: content,
		})
	}
	return request
}

func normalizeCohereRoleV2(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant", "system", "tool":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return "user"
	}
}

func extractStopSequencesV2(stop any) []string {
	switch value := stop.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return nil
		}
		return []string{value}
	case []any:
		results := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				results = append(results, text)
			}
		}
		if len(results) == 0 {
			return nil
		}
		return results
	default:
		return nil
	}
}

func stopReasonCohereV22OpenAI(reason *string) string {
	if reason == nil {
		return ""
	}
	switch strings.ToUpper(strings.TrimSpace(*reason)) {
	case "COMPLETE":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "TOOL_CALL":
		return "tool_calls"
	default:
		return strings.ToLower(strings.TrimSpace(*reason))
	}
}

func flattenCohereContentTextV2(content []ContentBlockV2) string {
	var builder strings.Builder
	for _, block := range content {
		if block.Type == "" || block.Type == "text" {
			builder.WriteString(block.Text)
		}
	}
	return builder.String()
}

func ResponseCohereV22OpenAI(cohereResponse *ResponseV2) *openai.TextResponse {
	choice := openai.TextResponseChoice{
		Index: 0,
		Message: model.Message{
			Role:    "assistant",
			Content: flattenCohereContentTextV2(cohereResponse.Message.Content),
			Name:    nil,
		},
		FinishReason: stopReasonCohereV22OpenAI(cohereResponse.FinishReason),
	}
	fullTextResponse := openai.TextResponse{
		Id:      fmt.Sprintf("chatcmpl-%s", cohereResponse.Id),
		Model:   "model",
		Object:  "chat.completion",
		Created: helper.GetTimestamp(),
		Choices: []openai.TextResponseChoice{choice},
	}
	return &fullTextResponse
}

func StreamHandlerV2(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	createdTime := helper.GetTimestamp()
	scanner := bufio.NewScanner(resp.Body)
	common.SetEventStreamHeaders(c)

	var usage model.Usage
	var currentEvent string

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event: "))
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var streamResponse StreamResponseV2
		if err := json.Unmarshal([]byte(payload), &streamResponse); err != nil {
			logger.SysError("error unmarshalling cohere v2 stream response: " + err.Error())
			continue
		}
		if streamResponse.Type == "" {
			streamResponse.Type = currentEvent
		}

		switch streamResponse.Type {
		case "content-delta":
			content := streamResponse.Delta.Message.Content.Text
			if content == "" {
				continue
			}
			response := openai.ChatCompletionsStreamResponse{
				Id:      fmt.Sprintf("chatcmpl-%d", createdTime),
				Model:   c.GetString("original_model"),
				Object:  "chat.completion.chunk",
				Created: createdTime,
				Choices: []openai.ChatCompletionsStreamResponseChoice{{
					Delta: model.Message{
						Role:    "assistant",
						Content: content,
					},
				}},
			}
			if err := render.ObjectData(c, response); err != nil {
				logger.SysError(err.Error())
			}
		case "message-end":
			if streamResponse.Usage != nil {
				usage.PromptTokens += streamResponse.Usage.Tokens.InputTokens
				usage.CompletionTokens += streamResponse.Usage.Tokens.OutputTokens
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logger.SysError("error reading cohere v2 stream: " + err.Error())
	}

	render.Done(c)
	if err := resp.Body.Close(); err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	if usage.TotalTokens == 0 && (usage.PromptTokens != 0 || usage.CompletionTokens != 0) {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return nil, &usage
}

func HandlerV2(c *gin.Context, resp *http.Response, promptTokens int, modelName string) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	if err = resp.Body.Close(); err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}

	var cohereResponse ResponseV2
	if err = json.Unmarshal(responseBody, &cohereResponse); err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}

	if strings.TrimSpace(cohereResponse.Id) == "" {
		return &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: strings.TrimSpace(cohereResponse.MessageText),
				Type:    "upstream_error",
				Param:   "",
				Code:    resp.StatusCode,
			},
			StatusCode: resp.StatusCode,
		}, nil
	}

	fullTextResponse := ResponseCohereV22OpenAI(&cohereResponse)
	fullTextResponse.Model = modelName
	usage := model.Usage{
		PromptTokens:     cohereResponse.Usage.Tokens.InputTokens,
		CompletionTokens: cohereResponse.Usage.Tokens.OutputTokens,
		TotalTokens:      cohereResponse.Usage.Tokens.InputTokens + cohereResponse.Usage.Tokens.OutputTokens,
	}
	fullTextResponse.Usage = usage

	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	return nil, &usage
}
