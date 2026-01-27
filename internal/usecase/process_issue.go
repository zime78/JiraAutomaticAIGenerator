package usecase

import (
	"fmt"
	"path/filepath"
	"strings"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/port"
)

// ProcessIssueUseCase handles the business logic for processing Jira issues
type ProcessIssueUseCase struct {
	jiraRepo       port.JiraRepository
	downloader     port.AttachmentDownloader
	videoProcessor port.VideoProcessor
	docGenerator   port.DocumentGenerator
	outputDir      string
}

// NewProcessIssueUseCase creates a new ProcessIssueUseCase
func NewProcessIssueUseCase(
	jiraRepo port.JiraRepository,
	downloader port.AttachmentDownloader,
	videoProcessor port.VideoProcessor,
	docGenerator port.DocumentGenerator,
	outputDir string,
) *ProcessIssueUseCase {
	return &ProcessIssueUseCase{
		jiraRepo:       jiraRepo,
		downloader:     downloader,
		videoProcessor: videoProcessor,
		docGenerator:   docGenerator,
		outputDir:      outputDir,
	}
}

// ProgressCallback is called to report progress
type ProgressCallback func(progress float64, status string)

// Execute processes a Jira issue and generates a document
func (uc *ProcessIssueUseCase) Execute(issueKey string, onProgress ProgressCallback) (*domain.ProcessResult, error) {
	result := &domain.ProcessResult{}

	// Step 1: Fetch issue
	onProgress(0.1, "Jira 이슈 조회 중...")
	issue, err := uc.jiraRepo.GetIssue(issueKey)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("이슈 조회 실패: %v", err)
		return result, err
	}

	// Step 2: Download attachments
	onProgress(0.3, "첨부파일 다운로드 중...")
	mediaAttachments := filterMediaAttachments(issue.Attachments)
	downloadResults, err := uc.downloader.DownloadAll(issueKey, mediaAttachments)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("첨부파일 다운로드 실패: %v", err)
		return result, err
	}

	// Step 3: Collect images and process videos
	var imagePaths []string
	var framePaths []string

	for _, dr := range downloadResults {
		if dr.Error != nil {
			continue
		}
		if !dr.IsVideo {
			imagePaths = append(imagePaths, dr.LocalPath)
		}
	}

	// Step 4: Extract video frames
	onProgress(0.5, "동영상 프레임 추출 중...")
	if uc.videoProcessor.IsAvailable() {
		for _, dr := range downloadResults {
			if dr.Error != nil || !dr.IsVideo {
				continue
			}
			framesDir := filepath.Join(uc.outputDir, issueKey, "frames")
			frames, err := uc.videoProcessor.ExtractFrames(dr.LocalPath, framesDir, 1.0, 10)
			if err == nil {
				framePaths = append(framePaths, frames...)
			}
		}
	}

	// Step 5: Generate document
	onProgress(0.8, "문서 생성 중...")
	doc, err := uc.docGenerator.Generate(issue, imagePaths, framePaths, uc.outputDir)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("문서 생성 실패: %v", err)
		return result, err
	}

	// Step 6: Save to file
	mdPath, err := uc.docGenerator.SaveToFile(doc)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("파일 저장 실패: %v", err)
		return result, err
	}

	onProgress(1.0, fmt.Sprintf("완료! 저장 위치: %s", mdPath))
	result.Success = true
	result.Document = doc
	result.MDPath = mdPath

	return result, nil
}

// filterMediaAttachments filters attachments to only include images and videos
func filterMediaAttachments(attachments []domain.Attachment) []domain.Attachment {
	var media []domain.Attachment
	for _, att := range attachments {
		if strings.HasPrefix(att.MimeType, "image/") || strings.HasPrefix(att.MimeType, "video/") {
			media = append(media, att)
		}
	}
	return media
}
