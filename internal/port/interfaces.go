package port

import "jira-ai-generator/internal/domain"

// JiraRepository defines the interface for Jira operations
type JiraRepository interface {
	// GetIssue fetches a Jira issue by its key
	GetIssue(issueKey string) (*domain.JiraIssue, error)
	// DownloadAttachment downloads an attachment and returns its data
	DownloadAttachment(url string) ([]byte, error)
}

// AttachmentDownloader defines the interface for downloading attachments
type AttachmentDownloader interface {
	// DownloadAll downloads all media attachments for an issue
	DownloadAll(issueKey string, attachments []domain.Attachment) ([]domain.DownloadResult, error)
}

// VideoProcessor defines the interface for video processing
type VideoProcessor interface {
	// IsAvailable checks if video processing is available
	IsAvailable() bool
	// ExtractFrames extracts frames from a video file
	ExtractFrames(videoPath, outputDir string, interval float64, maxFrames int) ([]string, error)
}

// DocumentGenerator defines the interface for document generation
type DocumentGenerator interface {
	// Generate creates a document from a Jira issue
	Generate(issue *domain.JiraIssue, imagePaths, framePaths []string, outputDir string) (*domain.GeneratedDocument, error)
	// SaveToFile saves the document to a file
	SaveToFile(doc *domain.GeneratedDocument) (string, error)
	// GenerateClipboardContent creates content for clipboard
	GenerateClipboardContent(doc *domain.GeneratedDocument) string
}

// Clipboard defines the interface for clipboard operations
type Clipboard interface {
	// SetContent sets the clipboard content
	SetContent(content string)
}

// IssueStore defines the interface for persisting issue records
type IssueStore interface {
	CreateIssue(issue *domain.IssueRecord) error
	UpsertIssue(issue *domain.IssueRecord) error
	GetIssue(issueKey string) (*domain.IssueRecord, error)
	GetIssueByKeyAndChannel(issueKey string, channelIndex int) (*domain.IssueRecord, error)
	UpdateIssue(issue *domain.IssueRecord) error
	DeleteIssue(issueKey string) error
	DeleteIssueByIDAndChannel(issueID int64, channelIndex int) error
	ListIssuesByPhase(phase int) ([]*domain.IssueRecord, error)
	ListIssuesByChannel(channelIndex int) ([]*domain.IssueRecord, error)
	ListIssuesByChannelAndPhase(channelIndex, phase int) ([]*domain.IssueRecord, error)
	ListAllIssues() ([]*domain.IssueRecord, error)
}

// AnalysisResultStore defines the interface for persisting analysis results
type AnalysisResultStore interface {
	CreateAnalysisResult(result *domain.AnalysisResult) error
	GetAnalysisResult(issueID int64, phase int) (*domain.AnalysisResult, error)
	UpdateAnalysisResult(result *domain.AnalysisResult) error
	ListAnalysisResultsByIssue(issueID int64) ([]*domain.AnalysisResult, error)
}

// AttachmentStore defines the interface for persisting attachment records
type AttachmentStore interface {
	CreateAttachment(attachment *domain.AttachmentRecord) error
	ListAttachmentsByIssue(issueID int64) ([]*domain.AttachmentRecord, error)
	DeleteAttachmentsByIssue(issueID int64) error
}
