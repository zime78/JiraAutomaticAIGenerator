package usecase_test

import (
	"errors"
	"testing"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/mock"
	"jira-ai-generator/internal/usecase"
)

func TestProcessIssueUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockJira := &mock.JiraRepository{
		GetIssueFunc: func(issueKey string) (*domain.JiraIssue, error) {
			return &domain.JiraIssue{
				Key:         issueKey,
				Summary:     "Test Issue",
				Description: "Test Description",
				Link:        "https://example.atlassian.net/browse/" + issueKey,
			}, nil
		},
	}

	mockDownloader := &mock.AttachmentDownloader{
		DownloadAllFunc: func(issueKey string, attachments []domain.Attachment) ([]domain.DownloadResult, error) {
			return []domain.DownloadResult{}, nil
		},
	}

	mockVideoProcessor := &mock.VideoProcessor{
		IsAvailableFunc: func() bool { return false },
	}

	mockDocGenerator := &mock.DocumentGenerator{
		GenerateFunc: func(issue *domain.JiraIssue, imagePaths, framePaths []string, outputDir string) (*domain.GeneratedDocument, error) {
			return &domain.GeneratedDocument{
				IssueKey: issue.Key,
				Title:    issue.Summary,
				Content:  "Generated Content",
			}, nil
		},
		SaveToFileFunc: func(doc *domain.GeneratedDocument) (string, error) {
			return "/output/" + doc.IssueKey + ".md", nil
		},
	}

	uc := usecase.NewProcessIssueUseCase(
		mockJira,
		mockDownloader,
		mockVideoProcessor,
		mockDocGenerator,
		"/output",
	)

	progressCalled := false
	onProgress := func(progress float64, status string) {
		progressCalled = true
	}

	// Act
	result, err := uc.Execute("TEST-123", onProgress)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Document == nil {
		t.Fatal("expected document to not be nil")
	}
	if result.Document.IssueKey != "TEST-123" {
		t.Errorf("expected issue key TEST-123, got %s", result.Document.IssueKey)
	}
	if !progressCalled {
		t.Error("expected progress callback to be called")
	}
}

func TestProcessIssueUseCase_Execute_JiraError(t *testing.T) {
	// Arrange
	mockJira := &mock.JiraRepository{
		GetIssueFunc: func(issueKey string) (*domain.JiraIssue, error) {
			return nil, errors.New("jira connection failed")
		},
	}

	mockDownloader := &mock.AttachmentDownloader{}
	mockVideoProcessor := &mock.VideoProcessor{}
	mockDocGenerator := &mock.DocumentGenerator{}

	uc := usecase.NewProcessIssueUseCase(
		mockJira,
		mockDownloader,
		mockVideoProcessor,
		mockDocGenerator,
		"/output",
	)

	// Act
	result, err := uc.Execute("TEST-123", func(float64, string) {})

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.Success {
		t.Error("expected success to be false")
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message to be set")
	}
}

func TestProcessIssueUseCase_Execute_WithAttachments(t *testing.T) {
	// Arrange
	mockJira := &mock.JiraRepository{
		GetIssueFunc: func(issueKey string) (*domain.JiraIssue, error) {
			return &domain.JiraIssue{
				Key:     issueKey,
				Summary: "Issue with attachments",
				Attachments: []domain.Attachment{
					{ID: "1", Filename: "image.png", MimeType: "image/png"},
					{ID: "2", Filename: "video.mp4", MimeType: "video/mp4"},
				},
			}, nil
		},
	}

	mockDownloader := &mock.AttachmentDownloader{
		DownloadAllFunc: func(issueKey string, attachments []domain.Attachment) ([]domain.DownloadResult, error) {
			return []domain.DownloadResult{
				{Attachment: attachments[0], LocalPath: "/output/image.png", IsVideo: false},
				{Attachment: attachments[1], LocalPath: "/output/video.mp4", IsVideo: true},
			}, nil
		},
	}

	var receivedImagePaths []string
	mockDocGenerator := &mock.DocumentGenerator{
		GenerateFunc: func(issue *domain.JiraIssue, imagePaths, framePaths []string, outputDir string) (*domain.GeneratedDocument, error) {
			receivedImagePaths = imagePaths
			return &domain.GeneratedDocument{IssueKey: issue.Key}, nil
		},
		SaveToFileFunc: func(doc *domain.GeneratedDocument) (string, error) {
			return "/output/doc.md", nil
		},
	}

	mockVideoProcessor := &mock.VideoProcessor{
		IsAvailableFunc: func() bool { return true },
		ExtractFramesFunc: func(videoPath, outputDir string, interval float64, maxFrames int) ([]string, error) {
			return []string{"/output/frame1.png", "/output/frame2.png"}, nil
		},
	}

	uc := usecase.NewProcessIssueUseCase(
		mockJira,
		mockDownloader,
		mockVideoProcessor,
		mockDocGenerator,
		"/output",
	)

	// Act
	result, err := uc.Execute("TEST-456", func(float64, string) {})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(receivedImagePaths) != 1 {
		t.Errorf("expected 1 image path, got %d", len(receivedImagePaths))
	}
}
