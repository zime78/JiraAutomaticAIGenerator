package domain

// JiraIssue represents a Jira issue with its details
type JiraIssue struct {
	Key         string       `json:"key"`
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	Attachments []Attachment `json:"attachments"`
	Link        string       `json:"link"`
}

// Attachment represents a file attached to a Jira issue
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
	URL      string `json:"content"`
}

// GeneratedDocument represents the output document for AI processing
type GeneratedDocument struct {
	IssueKey   string
	Title      string
	Content    string
	OutputDir  string
	ImagePaths []string
	FramePaths []string
}

// ProcessResult represents the result of processing a Jira issue
type ProcessResult struct {
	Success      bool
	Document     *GeneratedDocument
	MDPath       string
	ErrorMessage string
}

// DownloadResult represents the result of downloading an attachment
type DownloadResult struct {
	Attachment Attachment
	LocalPath  string
	Error      error
	IsVideo    bool
}
