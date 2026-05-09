package model

import "strings"

var geminiUnsupportedSchemaKeysV2 = map[string]struct{}{
	"$schema":               {},
	"additionalProperties":  {},
	"unevaluatedProperties": {},
	"patternProperties":     {},
	"$defs":                 {},
	"definitions":           {},
}

func IsGeminiModelLikeV2(modelName string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(modelName)), "gemini")
}

func SanitizeGeminiCompatibleRequestV2(request *GeneralOpenAIRequest) {
	if request == nil {
		return
	}
	for i := range request.Tools {
		request.Tools[i].Function.Parameters = sanitizeGeminiSchemaValueV2(request.Tools[i].Function.Parameters)
	}
	request.Functions = sanitizeGeminiFunctionsV2(request.Functions)
	if request.ResponseFormat != nil && request.ResponseFormat.JsonSchema != nil {
		request.ResponseFormat.JsonSchema.Schema = sanitizeGeminiSchemaMapV2(request.ResponseFormat.JsonSchema.Schema)
	}
}

func sanitizeGeminiFunctionsV2(functions any) any {
	switch typed := functions.(type) {
	case []Function:
		sanitized := make([]Function, len(typed))
		copy(sanitized, typed)
		for i := range sanitized {
			sanitized[i].Parameters = sanitizeGeminiSchemaValueV2(sanitized[i].Parameters)
		}
		return sanitized
	case []any:
		sanitized := make([]any, len(typed))
		for i, item := range typed {
			sanitized[i] = sanitizeGeminiFunctionValueV2(item)
		}
		return sanitized
	default:
		return sanitizeGeminiFunctionValueV2(functions)
	}
}

func sanitizeGeminiFunctionValueV2(functionValue any) any {
	functionMap, ok := functionValue.(map[string]any)
	if !ok {
		return functionValue
	}
	sanitized := make(map[string]any, len(functionMap))
	for key, value := range functionMap {
		if key == "parameters" {
			sanitized[key] = sanitizeGeminiSchemaValueV2(value)
			continue
		}
		sanitized[key] = sanitizeGeminiSchemaValueV2(value)
	}
	return sanitized
}

func sanitizeGeminiSchemaMapV2(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	sanitized, ok := sanitizeGeminiSchemaValueV2(schema).(map[string]any)
	if !ok {
		return schema
	}
	return sanitized
}

func sanitizeGeminiSchemaValueV2(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, item := range typed {
			if _, shouldDrop := geminiUnsupportedSchemaKeysV2[key]; shouldDrop {
				continue
			}
			sanitized[key] = sanitizeGeminiSchemaValueV2(item)
		}
		return sanitized
	case []any:
		sanitized := make([]any, len(typed))
		for i, item := range typed {
			sanitized[i] = sanitizeGeminiSchemaValueV2(item)
		}
		return sanitized
	default:
		return value
	}
}
