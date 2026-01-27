package mock

import "jira-ai-generator/internal/domain"

// JiraRepository is a mock implementation of port.JiraRepository
type JiraRepository struct {
	GetIssueFunc           func(issueKey string) (*domain.JiraIssue, error)
	DownloadAttachmentFunc func(url string) ([]byte, error)
}

func (m *JiraRepository) GetIssue(issueKey string) (*domain.JiraIssue, error) {
	if m.GetIssueFunc != nil {
		return m.GetIssueFunc(issueKey)
	}
	return nil, nil
}

func (m *JiraRepository) DownloadAttachment(url string) ([]byte, error) {
	if m.DownloadAttachmentFunc != nil {
		return m.DownloadAttachmentFunc(url)
	}
	return nil, nil
}

// AttachmentDownloader is a mock implementation of port.AttachmentDownloader
type AttachmentDownloader struct {
	DownloadAllFunc func(issueKey string, attachments []domain.Attachment) ([]domain.DownloadResult, error)
}

func (m *AttachmentDownloader) DownloadAll(issueKey string, attachments []domain.Attachment) ([]domain.DownloadResult, error) {
	if m.DownloadAllFunc != nil {
		return m.DownloadAllFunc(issueKey, attachments)
	}
	return nil, nil
}

// VideoProcessor is a mock implementation of port.VideoProcessor
type VideoProcessor struct {
	IsAvailableFunc   func() bool
	ExtractFramesFunc func(videoPath, outputDir string, interval float64, maxFrames int) ([]string, error)
}

func (m *VideoProcessor) IsAvailable() bool {
	if m.IsAvailableFunc != nil {
		return m.IsAvailableFunc()
	}
	return false
}

func (m *VideoProcessor) ExtractFrames(videoPath, outputDir string, interval float64, maxFrames int) ([]string, error) {
	if m.ExtractFramesFunc != nil {
		return m.ExtractFramesFunc(videoPath, outputDir, interval, maxFrames)
	}
	return nil, nil
}

// DocumentGenerator is a mock implementation of port.DocumentGenerator
type DocumentGenerator struct {
	GenerateFunc                 func(issue *domain.JiraIssue, imagePaths, framePaths []string, outputDir string) (*domain.GeneratedDocument, error)
	SaveToFileFunc               func(doc *domain.GeneratedDocument) (string, error)
	GenerateClipboardContentFunc func(doc *domain.GeneratedDocument) string
}

func (m *DocumentGenerator) Generate(issue *domain.JiraIssue, imagePaths, framePaths []string, outputDir string) (*domain.GeneratedDocument, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(issue, imagePaths, framePaths, outputDir)
	}
	return nil, nil
}

func (m *DocumentGenerator) SaveToFile(doc *domain.GeneratedDocument) (string, error) {
	if m.SaveToFileFunc != nil {
		return m.SaveToFileFunc(doc)
	}
	return "", nil
}

func (m *DocumentGenerator) GenerateClipboardContent(doc *domain.GeneratedDocument) string {
	if m.GenerateClipboardContentFunc != nil {
		return m.GenerateClipboardContentFunc(doc)
	}
	return ""
}

// Clipboard is a mock implementation of port.Clipboard
type Clipboard struct {
	SetContentFunc func(content string)
	Content        string
}

func (m *Clipboard) SetContent(content string) {
	m.Content = content
	if m.SetContentFunc != nil {
		m.SetContentFunc(content)
	}
}
