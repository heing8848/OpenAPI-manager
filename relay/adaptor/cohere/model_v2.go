package cohere

type RequestV2 struct {
	Model            string            `json:"model,omitempty"`
	Messages         []MessageV2       `json:"messages,omitempty"`
	Stream           bool              `json:"stream"`
	MaxTokens        int               `json:"max_tokens,omitempty"`
	Temperature      *float64          `json:"temperature,omitempty"`
	P                *float64          `json:"p,omitempty"`
	K                int               `json:"k,omitempty"`
	Seed             int               `json:"seed,omitempty"`
	FrequencyPenalty *float64          `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64          `json:"presence_penalty,omitempty"`
	StopSequences    []string          `json:"stop_sequences,omitempty"`
	ResponseFormat   *ResponseFormatV2 `json:"response_format,omitempty"`
}

type MessageV2 struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormatV2 struct {
	Type       string      `json:"type,omitempty"`
	JsonSchema interface{} `json:"json_schema,omitempty"`
}

type ResponseV2 struct {
	Id           string           `json:"id"`
	FinishReason *string          `json:"finish_reason"`
	Message      AssistantMessage `json:"message"`
	Usage        UsageV2          `json:"usage"`
	MessageText  string           `json:"text"`
}

type AssistantMessage struct {
	Role    string           `json:"role"`
	Content []ContentBlockV2 `json:"content"`
}

type ContentBlockV2 struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type UsageV2 struct {
	BilledUnits BilledUnitsV2 `json:"billed_units"`
	Tokens      TokenUsageV2  `json:"tokens"`
}

type BilledUnitsV2 struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type TokenUsageV2 struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type StreamResponseV2 struct {
	Type  string        `json:"type"`
	Delta StreamDeltaV2 `json:"delta"`
	Usage *UsageV2      `json:"usage,omitempty"`
}

type StreamDeltaV2 struct {
	Message StreamMessageDeltaV2 `json:"message"`
}

type StreamMessageDeltaV2 struct {
	Content StreamContentDeltaV2 `json:"content"`
}

type StreamContentDeltaV2 struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}
