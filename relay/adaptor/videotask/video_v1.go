package videotask

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func ParseCreateTaskResponseV1(body []byte) (*relaymodel.VideoTaskResponseV1, error) {
	payload, err := decodeJSONObjectV1(body)
	if err != nil {
		return nil, err
	}
	task, err := normalizeVideoTaskObjectV1(extractTaskObjectV1(payload))
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func ParseTaskQueryResponseV1(body []byte) ([]relaymodel.VideoTaskResponseV1, error) {
	payload, err := decodeJSONObjectV1(body)
	if err != nil {
		return nil, err
	}

	taskObjects := extractTaskObjectListV1(payload)
	if len(taskObjects) == 0 {
		return nil, fmt.Errorf("bad upstream video task response: no task object found")
	}

	result := make([]relaymodel.VideoTaskResponseV1, 0, len(taskObjects))
	for _, taskObject := range taskObjects {
		task, normalizeErr := normalizeVideoTaskObjectV1(taskObject)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		result = append(result, task)
	}
	return result, nil
}

func decodeJSONObjectV1(body []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("bad upstream video task response: %w", err)
	}
	return payload, nil
}

func extractTaskObjectV1(payload map[string]any) map[string]any {
	if task, ok := payload["task"].(map[string]any); ok {
		return task
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return data
	}
	return payload
}

func extractTaskObjectListV1(payload map[string]any) []map[string]any {
	for _, key := range []string{"data", "tasks", "items"} {
		if rawItems, ok := payload[key].([]any); ok {
			items := make([]map[string]any, 0, len(rawItems))
			for _, rawItem := range rawItems {
				if item, ok := rawItem.(map[string]any); ok {
					items = append(items, item)
				}
			}
			if len(items) > 0 {
				return items
			}
		}
	}
	if task, ok := payload["task"].(map[string]any); ok {
		return []map[string]any{task}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return []map[string]any{data}
	}
	if payload != nil {
		return []map[string]any{payload}
	}
	return nil
}

func normalizeVideoTaskObjectV1(payload map[string]any) (relaymodel.VideoTaskResponseV1, error) {
	if payload == nil {
		return relaymodel.VideoTaskResponseV1{}, fmt.Errorf("bad upstream video task response: empty payload")
	}
	taskID := firstStringV1(payload, "id", "task_id", "taskId")
	if taskID == "" {
		return relaymodel.VideoTaskResponseV1{}, fmt.Errorf("bad upstream video task response: missing task id")
	}

	status := relaymodel.NormalizeVideoTaskStatusV1(firstStringV1(payload, "status", "state", "task_status", "taskStatus"))
	if status == "" {
		status = relaymodel.VideoTaskStatusQueuedV1
	}

	videoURLs := dedupeStringsV1(
		append(
			append(stringSliceFromAnyV1(payload["video_urls"]), stringSliceFromAnyV1(payload["videos"])...),
			stringSliceFromAnyV1(payload["output"])...,
		)...,
	)
	if singleVideoURL := firstStringV1(payload, "video_url", "videoUrl"); singleVideoURL != "" {
		videoURLs = dedupeStringsV1(append(videoURLs, singleVideoURL)...)
	}

	thumbnailURLs := dedupeStringsV1(
		append(
			append(stringSliceFromAnyV1(payload["thumbnail_urls"]), stringSliceFromAnyV1(payload["thumbnails"])...),
			stringSliceFromAnyV1(payload["thumbnail"])...,
		)...,
	)
	if singleThumbURL := firstStringV1(payload, "thumbnail_url", "thumbnailUrl", "poster_url"); singleThumbURL != "" {
		thumbnailURLs = dedupeStringsV1(append(thumbnailURLs, singleThumbURL)...)
	}

	if status == relaymodel.VideoTaskStatusCompletedV1 && len(videoURLs) == 0 {
		return relaymodel.VideoTaskResponseV1{}, fmt.Errorf("bad upstream video task response: completed task has no video urls")
	}

	metadata := make(map[string]any)
	if pollingURL := firstStringV1(payload, "polling_url", "poll_url"); pollingURL != "" {
		metadata["upstream_polling_url"] = pollingURL
	}
	if providerStatus := firstStringV1(payload, "status", "state", "task_status", "taskStatus"); providerStatus != "" {
		metadata["upstream_status"] = providerStatus
	}
	if upstreamModel := firstStringV1(payload, "model", "model_name", "modelName"); upstreamModel != "" {
		metadata["upstream_model"] = upstreamModel
	}

	task := relaymodel.VideoTaskResponseV1{
		ID:            taskID,
		Status:        status,
		Model:         firstStringV1(payload, "model", "model_name", "modelName"),
		CreatedAt:     normalizeTimestampV1(firstValueV1(payload, "created_at", "createdAt", "created_time", "submit_time")),
		CompletedAt:   normalizeTimestampV1(firstValueV1(payload, "completed_at", "completedAt", "finished_at", "finish_time")),
		VideoURLs:     videoURLs,
		ThumbnailURLs: thumbnailURLs,
		Error:         extractVideoTaskErrorV1(payload),
	}
	if len(metadata) > 0 {
		task.Metadata = metadata
	}
	return task, nil
}

func extractVideoTaskErrorV1(payload map[string]any) *relaymodel.VideoTaskErrorV1 {
	if payload == nil {
		return nil
	}
	if rawError, ok := payload["error"]; ok {
		switch value := rawError.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return &relaymodel.VideoTaskErrorV1{Message: strings.TrimSpace(value)}
			}
		case map[string]any:
			message := firstStringV1(value, "message", "msg", "error")
			if message != "" {
				return &relaymodel.VideoTaskErrorV1{
					Message: message,
					Code:    firstValueV1(value, "code", "error_code"),
				}
			}
		}
	}
	if failureReason := firstStringV1(payload, "error_message", "errorMessage", "fail_reason", "failure_reason", "reason"); failureReason != "" {
		return &relaymodel.VideoTaskErrorV1{Message: failureReason}
	}
	return nil
}

func firstValueV1(payload map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}

func firstStringV1(payload map[string]any, keys ...string) string {
	value := firstValueV1(payload, keys...)
	switch converted := value.(type) {
	case string:
		return strings.TrimSpace(converted)
	default:
		return ""
	}
}

func stringSliceFromAnyV1(raw any) []string {
	switch value := raw.(type) {
	case nil:
		return nil
	case string:
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return []string{trimmed}
		}
	case []string:
		return dedupeStringsV1(value...)
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			switch converted := item.(type) {
			case string:
				if trimmed := strings.TrimSpace(converted); trimmed != "" {
					result = append(result, trimmed)
				}
			case map[string]any:
				if nestedURL := firstStringV1(converted, "url", "video_url", "thumbnail_url"); nestedURL != "" {
					result = append(result, nestedURL)
				}
			}
		}
		return dedupeStringsV1(result...)
	}
	return nil
}

func dedupeStringsV1(values ...string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeTimestampV1(raw any) string {
	switch value := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case float64:
		return time.Unix(int64(value), 0).UTC().Format(time.RFC3339)
	case int64:
		return time.Unix(value, 0).UTC().Format(time.RFC3339)
	case int:
		return time.Unix(int64(value), 0).UTC().Format(time.RFC3339)
	case json.Number:
		if unixValue, err := value.Int64(); err == nil {
			return time.Unix(unixValue, 0).UTC().Format(time.RFC3339)
		}
		if floatValue, err := strconv.ParseFloat(value.String(), 64); err == nil {
			return time.Unix(int64(floatValue), 0).UTC().Format(time.RFC3339)
		}
	}
	return ""
}
