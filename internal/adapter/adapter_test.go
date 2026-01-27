package adapter_test

import (
	"testing"

	"jira-ai-generator/internal/adapter"
)

func TestExtractIssueKeyFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard browse URL",
			url:      "https://example.atlassian.net/browse/PROJ-123",
			expected: "PROJ-123",
		},
		{
			name:     "URL with query parameters",
			url:      "https://example.atlassian.net/browse/PROJ-456?filter=123",
			expected: "PROJ-456",
		},
		{
			name:     "software project URL",
			url:      "https://example.atlassian.net/jira/software/projects/PROJ/issues/PROJ-789",
			expected: "PROJ-789",
		},
		{
			name:     "invalid URL - no issue key",
			url:      "https://example.atlassian.net/browse/",
			expected: "",
		},
		{
			name:     "issue key only",
			url:      "ITSM-5239",
			expected: "ITSM-5239",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.ExtractIssueKeyFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractIssueKeyFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestMarkdownGenerator_Generate(t *testing.T) {
	gen := adapter.NewMarkdownGenerator("테스트 프롬프트")

	// This is a simplified test - full test would use domain.JiraIssue
	if gen == nil {
		t.Fatal("generator should not be nil")
	}
}

func TestFFmpegVideoProcessor_IsAvailable(t *testing.T) {
	processor := adapter.NewFFmpegVideoProcessor()

	// Just verify it doesn't panic
	available := processor.IsAvailable()
	t.Logf("FFmpeg available: %v", available)
}
