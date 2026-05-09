package controller

import "testing"

func TestShouldRetryStatusV2(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "404 should not retry",
			statusCode: 404,
			expected:   false,
		},
		{
			name:       "400 should not retry",
			statusCode: 400,
			expected:   false,
		},
		{
			name:       "429 should retry",
			statusCode: 429,
			expected:   true,
		},
		{
			name:       "503 should retry",
			statusCode: 503,
			expected:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := shouldRetryStatusV2(testCase.statusCode)
			if actual != testCase.expected {
				t.Fatalf("expected retry=%v for status %d, got %v", testCase.expected, testCase.statusCode, actual)
			}
		})
	}
}
