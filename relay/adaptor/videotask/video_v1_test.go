package videotask

import "testing"

func TestParseCreateTaskResponseV1NormalizesTask(t *testing.T) {
	task, err := ParseCreateTaskResponseV1([]byte(`{
		"task": {
			"id": "task-123",
			"status": "submitted",
			"model": "video-fast",
			"created_at": "2026-04-13T00:00:00Z"
		}
	}`))
	if err != nil {
		t.Fatalf("expected successful normalization, got %v", err)
	}
	if task.ID != "task-123" {
		t.Fatalf("unexpected task id %q", task.ID)
	}
	if task.Status != "queued" {
		t.Fatalf("unexpected normalized status %q", task.Status)
	}
}

func TestParseTaskQueryResponseV1NormalizesStatuses(t *testing.T) {
	tasks, err := ParseTaskQueryResponseV1([]byte(`{
		"tasks": [
			{"id": "task-1", "status": "running"},
			{"id": "task-2", "status": "success", "video_urls": ["https://example.com/video.mp4"]}
		]
	}`))
	if err != nil {
		t.Fatalf("expected query payload to normalize, got %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Status != "processing" {
		t.Fatalf("unexpected status for task-1: %q", tasks[0].Status)
	}
	if tasks[1].Status != "completed" {
		t.Fatalf("unexpected status for task-2: %q", tasks[1].Status)
	}
}

func TestParseTaskQueryResponseV1RejectsCompletedWithoutVideo(t *testing.T) {
	_, err := ParseTaskQueryResponseV1([]byte(`{
		"data": {
			"id": "task-1",
			"status": "completed"
		}
	}`))
	if err == nil {
		t.Fatalf("expected completed-without-video error")
	}
}

func TestParseTaskQueryResponseV1RejectsMalformedPayload(t *testing.T) {
	_, err := ParseTaskQueryResponseV1([]byte(`{"message":"not-a-task-payload"}`))
	if err == nil {
		t.Fatalf("expected malformed payload error")
	}
}
