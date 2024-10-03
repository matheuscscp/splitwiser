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
			name:     "simple",
			content:  "```json\n{\"text\":\"hello\"}\n```",
			expected: "{\"text\":\"hello\"}",
		},
		{
			name:     "simple_no_json",
			content:  "```{\"text\":\"hello\"}\n```",
			expected: "{\"text\":\"hello\"}",
		},
		{
			name:     "simple_no_backticks",
			content:  "{\"text\":\"hello\"}",
			expected: "{\"text\":\"hello\"}",
		},
		{
			name:     "simple_no_backticks_no_json",
			content:  "{\"text\":\"hello\"}",
			expected: "{\"text\":\"hello\"}",
		},
		{
			name:     "complex",
			content:  "```json\n{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}\n```",
			expected: "{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}",
		},
		{
			name:     "complex_no_json",
			content:  "```{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}\n```",
			expected: "{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}",
		},
		{
			name:     "complex_no_backticks",
			content:  "{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}",
			expected: "{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}",
		},
		{
			name:     "complex_no_backticks_no_json",
			content:  "{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}",
			expected: "{\"text\":\"hello\",\"choices\":[\"yes\",\"no\"]}",
		},
		{
			name:     "simple_with_newlines",
			content:  "\n```json\n{\n\"text\":\"hello\"\n}\n```",
			expected: "{\n\"text\":\"hello\"\n}",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := openaipkg.CleanOpenAIJSONResponse(tt.content); got != tt.expected {
				t.Errorf("CleanOpenAIJSONResponse() = %v, want %v", got, tt.expected)
			}
		})
	}
}
