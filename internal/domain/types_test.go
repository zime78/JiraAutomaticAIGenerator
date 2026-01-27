package domain_test

import (
	"testing"

	"jira-ai-generator/internal/domain"
)

func TestJiraIssue_Fields(t *testing.T) {
	issue := domain.JiraIssue{
		Key:         "TEST-123",
		Summary:     "Test Summary",
		Description: "Test Description",
		Link:        "https://example.com/browse/TEST-123",
		Attachments: []domain.Attachment{
			{
				ID:       "1",
				Filename: "test.png",
				MimeType: "image/png",
				Size:     1024,
				URL:      "https://example.com/attachment/1",
			},
		},
	}

	if issue.Key != "TEST-123" {
		t.Errorf("expected Key TEST-123, got %s", issue.Key)
	}
	if len(issue.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(issue.Attachments))
	}
}

func TestDownloadResult_IsVideo(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		isVideo  bool
	}{
		{"image file", "image/png", false},
		{"video file", "video/mp4", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.DownloadResult{
				Attachment: domain.Attachment{MimeType: tt.mimeType},
				IsVideo:    tt.isVideo,
			}
			if result.IsVideo != tt.isVideo {
				t.Errorf("expected IsVideo=%v, got %v", tt.isVideo, result.IsVideo)
			}
		})
	}
}

func TestProcessResult_Success(t *testing.T) {
	result := domain.ProcessResult{
		Success: true,
		Document: &domain.GeneratedDocument{
			IssueKey: "TEST-123",
		},
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Document == nil {
		t.Error("expected document to not be nil")
	}
}
