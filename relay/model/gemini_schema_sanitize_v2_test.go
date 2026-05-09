package model

import "testing"

func TestSanitizeGeminiCompatibleRequestV2RemovesUnsupportedSchemaKeys(t *testing.T) {
	request := &GeneralOpenAIRequest{
		Model: "google-gemini-2.0-flash",
		Tools: []Tool{
			{
				Type: "function",
				Function: Function{
					Name: "builtin_web_search",
					Parameters: map[string]any{
						"$schema":              "http://json-schema.org/draft-07/schema#",
						"type":                 "object",
						"additionalProperties": false,
						"properties": map[string]any{
							"nested": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
							},
						},
					},
				},
			},
		},
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JsonSchema: &JSONSchema{
				Name: "result",
				Schema: map[string]any{
					"$schema":              "http://json-schema.org/draft-07/schema#",
					"type":                 "object",
					"additionalProperties": false,
				},
			},
		},
	}

	SanitizeGeminiCompatibleRequestV2(request)

	parameters, ok := request.Tools[0].Function.Parameters.(map[string]any)
	if !ok {
		t.Fatalf("expected parameters to remain a map, got %#v", request.Tools[0].Function.Parameters)
	}
	if _, exists := parameters["$schema"]; exists {
		t.Fatalf("expected $schema to be removed, got %#v", parameters)
	}
	if _, exists := parameters["additionalProperties"]; exists {
		t.Fatalf("expected additionalProperties to be removed, got %#v", parameters)
	}
	properties := parameters["properties"].(map[string]any)
	nested := properties["nested"].(map[string]any)
	if _, exists := nested["additionalProperties"]; exists {
		t.Fatalf("expected nested additionalProperties to be removed, got %#v", nested)
	}
	if _, exists := request.ResponseFormat.JsonSchema.Schema["$schema"]; exists {
		t.Fatalf("expected response schema $schema to be removed, got %#v", request.ResponseFormat.JsonSchema.Schema)
	}
	if _, exists := request.ResponseFormat.JsonSchema.Schema["additionalProperties"]; exists {
		t.Fatalf("expected response schema additionalProperties to be removed, got %#v", request.ResponseFormat.JsonSchema.Schema)
	}
}

func TestIsGeminiModelLikeV2(t *testing.T) {
	testCases := []struct {
		modelName string
		expected  bool
	}{
		{modelName: "google-gemini-3.1-flash-lite-preview", expected: true},
		{modelName: "gemini-2.0-flash", expected: true},
		{modelName: "gpt-4o-mini", expected: false},
	}

	for _, testCase := range testCases {
		if actual := IsGeminiModelLikeV2(testCase.modelName); actual != testCase.expected {
			t.Fatalf("expected %v for %q, got %v", testCase.expected, testCase.modelName, actual)
		}
	}
}
