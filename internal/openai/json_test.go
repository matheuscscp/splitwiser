package openaipkg_test

import (
	"testing"

	openaipkg "github.com/matheuscscp/splitwiser/internal/openai"
)

func TestCleanOpenAIJSONResponse(t *testing.T) {
	for _, tt := range []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty",
			content:  "",
			expected: "",
		},
		{
			name:     "no array",
			content:  "{}",
			expected: "{}",
		},
		{
			name:     "array",
			content:  "[{\"key\": \"value\"}]",
			expected: "[{\"key\": \"value\"}]",
		},
		{
			name:     "array with extra",
			content:  "extra[{\"key\": \"value\"}]extra",
			expected: "[{\"key\": \"value\"}]",
		},
		{
			name:     "array with extra and nested",
			content:  "extra[{\"key\": [{\"key\": \"value\"}]}]extra",
			expected: "[{\"key\": [{\"key\": \"value\"}]}]",
		},
		{
			name:     "array with extra and nested and extra",
			content:  "extra[{\"key\": [{\"key\": \"value\"}]}]extra",
			expected: "[{\"key\": [{\"key\": \"value\"}]}]",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := openaipkg.CleanOpenAIJSONResponse(tt.content)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
