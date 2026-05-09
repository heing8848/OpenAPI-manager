package relaymode

import "testing"

func TestGetByPathVideoV1(t *testing.T) {
	testCases := []struct {
		path string
		want int
	}{
		{path: "/v1/videos/generations", want: VideosGenerationsV1},
		{path: "/v1/videos/generations/tasks", want: VideoGenerationsTasksV1},
	}

	for _, tc := range testCases {
		if got := GetByPath(tc.path); got != tc.want {
			t.Fatalf("GetByPath(%q) = %d, want %d", tc.path, got, tc.want)
		}
	}
}
