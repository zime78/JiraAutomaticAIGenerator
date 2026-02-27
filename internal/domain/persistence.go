package domain

import "time"

// IssueRecord represents a persisted Jira issue record
type IssueRecord struct {
	ID           int64     `json:"id"`
	IssueKey     string    `json:"issue_key"`
	Summary      string    `json:"summary"`
	Description  string    `json:"description"`
	JiraURL      string    `json:"jira_url"`
	MDPath       string    `json:"md_path"`
	Phase        int       `json:"phase"`         // 1: 1차완료, 2: 2차완료
	Status       string    `json:"status"`        // active, archived
	ChannelIndex int       `json:"channel_index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AnalysisResult represents the result of AI analysis
type AnalysisResult struct {
	ID            int64      `json:"id"`
	IssueID       int64      `json:"issue_id"`
	AnalysisPhase int        `json:"analysis_phase"` // 1: PhaseAnalyze
	ResultPath    string     `json:"result_path"`
	PlanPath      string     `json:"plan_path"`
	Status        string     `json:"status"` // pending, running, completed, failed
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	ErrorMessage  string     `json:"error_message"`
}

// AttachmentRecord represents a persisted attachment
type AttachmentRecord struct {
	ID        int64  `json:"id"`
	IssueID   int64  `json:"issue_id"`
	Filename  string `json:"filename"`
	LocalPath string `json:"local_path"`
	MimeType  string `json:"mime_type"`
	IsVideo   bool   `json:"is_video"`
}
