package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	VideoContentTypeTextV1     = "text"
	VideoContentTypeImageURLV1 = "image_url"

	VideoTaskStatusQueuedV1     = "queued"
	VideoTaskStatusProcessingV1 = "processing"
	VideoTaskStatusCompletedV1  = "completed"
	VideoTaskStatusFailedV1     = "failed"
	VideoTaskStatusCanceledV1   = "canceled"

	videoTaskHandlePrefixV1 = "alfred-video-v1_"
)

type VideoGenerationRequestV1 struct {
	Model   string               `json:"model"`
	Content []VideoContentPartV1 `json:"content"`
}

type VideoContentPartV1 struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type VideoTaskErrorV1 struct {
	Message string `json:"message"`
	Code    any    `json:"code,omitempty"`
}

type VideoTaskResponseV1 struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	Model         string            `json:"model,omitempty"`
	CreatedAt     string            `json:"created_at,omitempty"`
	CompletedAt   string            `json:"completed_at,omitempty"`
	PollingURL    string            `json:"polling_url,omitempty"`
	VideoURLs     []string          `json:"video_urls,omitempty"`
	ThumbnailURLs []string          `json:"thumbnail_urls,omitempty"`
	Error         *VideoTaskErrorV1 `json:"error,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

type VideoTaskListResponseV1 struct {
	Object string                `json:"object"`
	Data   []VideoTaskResponseV1 `json:"data"`
}

type VideoTaskHandleV1 struct {
	Version        string `json:"version"`
	UserID         int    `json:"user_id"`
	ChannelID      int    `json:"channel_id"`
	ChannelType    int    `json:"channel_type"`
	RequestedModel string `json:"requested_model,omitempty"`
	UpstreamTaskID string `json:"upstream_task_id"`
}

func NormalizeVideoTaskStatusV1(rawStatus string) string {
	normalized := strings.ToLower(strings.TrimSpace(rawStatus))
	switch normalized {
	case "", "submitted", "pending", "queued", "created", "waiting":
		return VideoTaskStatusQueuedV1
	case "running", "processing", "in_progress", "generating":
		return VideoTaskStatusProcessingV1
	case "done", "succeeded", "success", "completed", "finished":
		return VideoTaskStatusCompletedV1
	case "failed", "error":
		return VideoTaskStatusFailedV1
	case "cancelled", "canceled":
		return VideoTaskStatusCanceledV1
	default:
		return normalized
	}
}

func EncodeVideoTaskHandleV1(handle VideoTaskHandleV1) (string, error) {
	handle.Version = "v1"
	payload, err := json.Marshal(handle)
	if err != nil {
		return "", err
	}
	return videoTaskHandlePrefixV1 + base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecodeVideoTaskHandleV1(value string) (*VideoTaskHandleV1, error) {
	if !strings.HasPrefix(value, videoTaskHandlePrefixV1) {
		return nil, fmt.Errorf("invalid video task id")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, videoTaskHandlePrefixV1))
	if err != nil {
		return nil, fmt.Errorf("invalid video task id")
	}
	var handle VideoTaskHandleV1
	if err = json.Unmarshal(payload, &handle); err != nil {
		return nil, fmt.Errorf("invalid video task id")
	}
	if handle.Version != "v1" || handle.ChannelID == 0 || handle.UpstreamTaskID == "" {
		return nil, fmt.Errorf("invalid video task id")
	}
	return &handle, nil
}
