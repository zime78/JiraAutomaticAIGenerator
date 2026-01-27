package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/port"
)

// AttachmentDownloader implements port.AttachmentDownloader
type AttachmentDownloader struct {
	jiraRepo  port.JiraRepository
	outputDir string
}

// NewAttachmentDownloader creates a new attachment downloader
func NewAttachmentDownloader(jiraRepo port.JiraRepository, outputDir string) *AttachmentDownloader {
	return &AttachmentDownloader{
		jiraRepo:  jiraRepo,
		outputDir: outputDir,
	}
}

// DownloadAll downloads all media attachments for an issue
func (d *AttachmentDownloader) DownloadAll(issueKey string, attachments []domain.Attachment) ([]domain.DownloadResult, error) {
	// Convert to absolute path for AI accessibility
	absOutputDir, err := filepath.Abs(d.outputDir)
	if err != nil {
		absOutputDir = d.outputDir
	}

	issueDir := filepath.Join(absOutputDir, issueKey)
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	var results []domain.DownloadResult

	for _, att := range attachments {
		if !isMediaFile(att.MimeType) {
			continue
		}

		result := domain.DownloadResult{
			Attachment: att,
			IsVideo:    isVideoFile(att.MimeType),
		}

		data, err := d.jiraRepo.DownloadAttachment(att.URL)
		if err != nil {
			result.Error = err
			results = append(results, result)
			continue
		}

		localPath := filepath.Join(issueDir, att.Filename)
		if err := os.WriteFile(localPath, data, 0644); err != nil {
			result.Error = err
			results = append(results, result)
			continue
		}

		result.LocalPath = localPath
		results = append(results, result)
	}

	return results, nil
}

func isMediaFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/")
}

func isVideoFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}
