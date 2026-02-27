package adapter

import (
	"database/sql"
	"fmt"
	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/logger"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteRepository implements IssueStore and AnalysisResultStore using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	logger.Debug("NewSQLiteRepository: opening database at %s", dbPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		logger.Debug("NewSQLiteRepository: failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(); err != nil {
		db.Close()
		logger.Debug("NewSQLiteRepository: migration failed: %v", err)
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Debug("NewSQLiteRepository: database opened and migrated successfully")
	return repo, nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	logger.Debug("SQLiteRepository: closing database connection")
	err := r.db.Close()
	if err != nil {
		logger.Debug("SQLiteRepository: close failed: %v", err)
	} else {
		logger.Debug("SQLiteRepository: database closed successfully")
	}
	return err
}

// migrate runs database migrations
func (r *SQLiteRepository) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS issues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issue_key TEXT NOT NULL,
			summary TEXT,
			description TEXT,
			jira_url TEXT,
			md_path TEXT,
			phase INTEGER DEFAULT 1,
			status TEXT DEFAULT 'active',
			channel_index INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(issue_key, channel_index)
		)`,
		`CREATE TABLE IF NOT EXISTS analysis_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issue_id INTEGER NOT NULL,
			analysis_phase INTEGER NOT NULL,
			result_path TEXT,
			plan_path TEXT,
			execution_path TEXT,
			status TEXT DEFAULT 'pending',
			started_at DATETIME,
			completed_at DATETIME,
			error_message TEXT,
			FOREIGN KEY (issue_id) REFERENCES issues(id)
		)`,
		`CREATE TABLE IF NOT EXISTS attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issue_id INTEGER NOT NULL,
			filename TEXT,
			local_path TEXT,
			mime_type TEXT,
			is_video BOOLEAN DEFAULT FALSE,
			FOREIGN KEY (issue_id) REFERENCES issues(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_key ON issues(issue_key)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_phase ON issues(phase)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_channel ON issues(channel_index)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_issues_key_channel ON issues(issue_key, channel_index)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_issue_id ON analysis_results(issue_id)`,
	}

	for _, migration := range migrations {
		if _, err := r.db.Exec(migration); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	if err := r.migrateIssueUniqueConstraint(); err != nil {
		return err
	}

	return nil
}

// migrateIssueUniqueConstraint는 legacy 단일 issue_key UNIQUE 제약을 복합 UNIQUE(issue_key, channel_index)로 교체한다.
func (r *SQLiteRepository) migrateIssueUniqueConstraint() error {
	var tableSQL string
	err := r.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='issues'`).Scan(&tableSQL)
	if err != nil {
		return fmt.Errorf("failed to inspect issues table schema: %w", err)
	}

	normalized := strings.ToLower(tableSQL)
	hasSingleUnique := strings.Contains(normalized, "unique(issue_key)") ||
		strings.Contains(normalized, "issue_key text unique") ||
		strings.Contains(normalized, "issue_key text not null unique")
	hasCompositeUnique := strings.Contains(normalized, "unique(issue_key, channel_index)")
	if !hasSingleUnique || hasCompositeUnique {
		return nil
	}

	logger.Debug("migrateIssueUniqueConstraint: legacy schema detected, migrating issues table")

	// PRAGMA는 트랜잭션 밖에서 실행해야 효과가 있다
	r.db.Exec(`PRAGMA foreign_keys = OFF`)

	tx, err := r.db.Begin()
	if err != nil {
		r.db.Exec(`PRAGMA foreign_keys = ON`)
		return fmt.Errorf("failed to begin schema migration tx: %w", err)
	}
	defer tx.Rollback()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS issues_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issue_key TEXT NOT NULL,
			summary TEXT,
			description TEXT,
			jira_url TEXT,
			md_path TEXT,
			phase INTEGER DEFAULT 1,
			status TEXT DEFAULT 'active',
			channel_index INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(issue_key, channel_index)
		)`,
		`INSERT INTO issues_new (id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at)
		 SELECT id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at
		 FROM (
		     SELECT *, ROW_NUMBER() OVER (
		         PARTITION BY issue_key, COALESCE(channel_index, -1)
		         ORDER BY updated_at DESC
		     ) as rn
		     FROM issues
		 ) WHERE rn = 1`,
		`DROP TABLE issues`,
		`ALTER TABLE issues_new RENAME TO issues`,
		`CREATE INDEX IF NOT EXISTS idx_issues_key ON issues(issue_key)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_phase ON issues(phase)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_channel ON issues(channel_index)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_issues_key_channel ON issues(issue_key, channel_index)`,
	}
	for _, stmt := range statements {
		if _, execErr := tx.Exec(stmt); execErr != nil {
			return fmt.Errorf("failed to execute schema migration statement: %w", execErr)
		}
	}

	if err := tx.Commit(); err != nil {
		r.db.Exec(`PRAGMA foreign_keys = ON`)
		return fmt.Errorf("failed to commit schema migration tx: %w", err)
	}

	r.db.Exec(`PRAGMA foreign_keys = ON`)

	logger.Debug("migrateIssueUniqueConstraint: migration completed")
	return nil
}

// CreateIssue creates a new issue record
func (r *SQLiteRepository) CreateIssue(issue *domain.IssueRecord) error {
	logger.Debug("CreateIssue: issueKey=%s, channel=%d", issue.IssueKey, issue.ChannelIndex)
	query := `INSERT INTO issues (issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := r.db.Exec(query,
		issue.IssueKey,
		issue.Summary,
		issue.Description,
		issue.JiraURL,
		issue.MDPath,
		issue.Phase,
		issue.Status,
		issue.ChannelIndex,
		now,
		now,
	)
	if err != nil {
		logger.Debug("CreateIssue: failed: %v", err)
		return fmt.Errorf("failed to create issue: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	issue.ID = id
	issue.CreatedAt = now
	issue.UpdatedAt = now

	logger.Debug("CreateIssue: success, ID=%d", id)
	return nil
}

// UpsertIssue creates or updates an issue record by (issue_key, channel_index).
func (r *SQLiteRepository) UpsertIssue(issue *domain.IssueRecord) error {
	if issue == nil {
		return fmt.Errorf("issue is nil")
	}

	existing, err := r.GetIssueByKeyAndChannel(issue.IssueKey, issue.ChannelIndex)
	if err == nil && existing != nil {
		issue.ID = existing.ID
		if issue.CreatedAt.IsZero() {
			issue.CreatedAt = existing.CreatedAt
		}
		return r.UpdateIssue(issue)
	}

	return r.CreateIssue(issue)
}

// GetIssue retrieves an issue by key
func (r *SQLiteRepository) GetIssue(issueKey string) (*domain.IssueRecord, error) {
	logger.Debug("GetIssue: issueKey=%s", issueKey)
	query := `SELECT id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at
		FROM issues WHERE issue_key = ? ORDER BY updated_at DESC LIMIT 1`

	var issue domain.IssueRecord
	err := r.db.QueryRow(query, issueKey).Scan(
		&issue.ID,
		&issue.IssueKey,
		&issue.Summary,
		&issue.Description,
		&issue.JiraURL,
		&issue.MDPath,
		&issue.Phase,
		&issue.Status,
		&issue.ChannelIndex,
		&issue.CreatedAt,
		&issue.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		logger.Debug("GetIssue: issue not found: %s", issueKey)
		return nil, fmt.Errorf("issue not found: %s", issueKey)
	}
	if err != nil {
		logger.Debug("GetIssue: failed: %v", err)
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	logger.Debug("GetIssue: success, ID=%d, phase=%d", issue.ID, issue.Phase)
	return &issue, nil
}

// GetIssueByKeyAndChannel retrieves an issue by key and channel index.
func (r *SQLiteRepository) GetIssueByKeyAndChannel(issueKey string, channelIndex int) (*domain.IssueRecord, error) {
	logger.Debug("GetIssueByKeyAndChannel: issueKey=%s, channel=%d", issueKey, channelIndex)
	query := `SELECT id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at
		FROM issues WHERE issue_key = ? AND channel_index = ?`

	var issue domain.IssueRecord
	err := r.db.QueryRow(query, issueKey, channelIndex).Scan(
		&issue.ID,
		&issue.IssueKey,
		&issue.Summary,
		&issue.Description,
		&issue.JiraURL,
		&issue.MDPath,
		&issue.Phase,
		&issue.Status,
		&issue.ChannelIndex,
		&issue.CreatedAt,
		&issue.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("issue not found: %s (channel=%d)", issueKey, channelIndex)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get issue by key and channel: %w", err)
	}
	return &issue, nil
}

// UpdateIssue updates an existing issue
func (r *SQLiteRepository) UpdateIssue(issue *domain.IssueRecord) error {
	logger.Debug("UpdateIssue: issueKey=%s, phase=%d, status=%s", issue.IssueKey, issue.Phase, issue.Status)
	query := `UPDATE issues SET summary = ?, description = ?, jira_url = ?, md_path = ?, phase = ?, status = ?, channel_index = ?, updated_at = ?
		WHERE issue_key = ? AND channel_index = ?`

	now := time.Now()
	_, err := r.db.Exec(query,
		issue.Summary,
		issue.Description,
		issue.JiraURL,
		issue.MDPath,
		issue.Phase,
		issue.Status,
		issue.ChannelIndex,
		now,
		issue.IssueKey,
		issue.ChannelIndex,
	)
	if err != nil {
		logger.Debug("UpdateIssue: failed: %v", err)
		return fmt.Errorf("failed to update issue: %w", err)
	}

	issue.UpdatedAt = now
	logger.Debug("UpdateIssue: success")
	return nil
}

// ListIssuesByPhase lists all issues in a specific phase
func (r *SQLiteRepository) ListIssuesByPhase(phase int) ([]*domain.IssueRecord, error) {
	logger.Debug("ListIssuesByPhase: phase=%d", phase)
	query := `SELECT id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at
		FROM issues WHERE phase = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, phase)
	if err != nil {
		logger.Debug("ListIssuesByPhase: query failed: %v", err)
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}
	defer rows.Close()

	var issues []*domain.IssueRecord
	for rows.Next() {
		var issue domain.IssueRecord
		err := rows.Scan(
			&issue.ID,
			&issue.IssueKey,
			&issue.Summary,
			&issue.Description,
			&issue.JiraURL,
			&issue.MDPath,
			&issue.Phase,
			&issue.Status,
			&issue.ChannelIndex,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		issues = append(issues, &issue)
	}

	logger.Debug("ListIssuesByPhase: found %d issues", len(issues))
	return issues, nil
}

// ListIssuesByChannel lists all issues for a specific channel
func (r *SQLiteRepository) ListIssuesByChannel(channelIndex int) ([]*domain.IssueRecord, error) {
	query := `SELECT id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at
		FROM issues WHERE channel_index = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, channelIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}
	defer rows.Close()

	var issues []*domain.IssueRecord
	for rows.Next() {
		var issue domain.IssueRecord
		err := rows.Scan(
			&issue.ID,
			&issue.IssueKey,
			&issue.Summary,
			&issue.Description,
			&issue.JiraURL,
			&issue.MDPath,
			&issue.Phase,
			&issue.Status,
			&issue.ChannelIndex,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		issues = append(issues, &issue)
	}

	return issues, nil
}

// ListIssuesByChannelAndPhase lists all issues for a specific channel and phase
func (r *SQLiteRepository) ListIssuesByChannelAndPhase(channelIndex, phase int) ([]*domain.IssueRecord, error) {
	logger.Debug("ListIssuesByChannelAndPhase: channelIndex=%d, phase=%d", channelIndex, phase)
	query := `SELECT id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at
		FROM issues WHERE channel_index = ? AND phase = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, channelIndex, phase)
	if err != nil {
		logger.Debug("ListIssuesByChannelAndPhase: query failed: %v", err)
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}
	defer rows.Close()

	var issues []*domain.IssueRecord
	for rows.Next() {
		var issue domain.IssueRecord
		err := rows.Scan(
			&issue.ID,
			&issue.IssueKey,
			&issue.Summary,
			&issue.Description,
			&issue.JiraURL,
			&issue.MDPath,
			&issue.Phase,
			&issue.Status,
			&issue.ChannelIndex,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		issues = append(issues, &issue)
	}

	logger.Debug("ListIssuesByChannelAndPhase: found %d issues", len(issues))
	return issues, nil
}

// CreateAnalysisResult creates a new analysis result
func (r *SQLiteRepository) CreateAnalysisResult(result *domain.AnalysisResult) error {
	query := `INSERT INTO analysis_results (issue_id, analysis_phase, result_path, plan_path, execution_path, status, started_at, completed_at, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := r.db.Exec(query,
		result.IssueID,
		result.AnalysisPhase,
		result.ResultPath,
		result.PlanPath,
		result.ExecutionPath,
		result.Status,
		result.StartedAt,
		result.CompletedAt,
		result.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to create analysis result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	result.ID = id
	return nil
}

// GetAnalysisResult retrieves an analysis result by issue ID and phase
func (r *SQLiteRepository) GetAnalysisResult(issueID int64, phase int) (*domain.AnalysisResult, error) {
	query := `SELECT id, issue_id, analysis_phase, result_path, plan_path, execution_path, status, started_at, completed_at, error_message
		FROM analysis_results WHERE issue_id = ? AND analysis_phase = ?`

	var result domain.AnalysisResult
	var startedAt, completedAt sql.NullTime

	err := r.db.QueryRow(query, issueID, phase).Scan(
		&result.ID,
		&result.IssueID,
		&result.AnalysisPhase,
		&result.ResultPath,
		&result.PlanPath,
		&result.ExecutionPath,
		&result.Status,
		&startedAt,
		&completedAt,
		&result.ErrorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("analysis result not found for issue %d phase %d", issueID, phase)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis result: %w", err)
	}

	if startedAt.Valid {
		result.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		result.CompletedAt = &completedAt.Time
	}

	return &result, nil
}

// UpdateAnalysisResult updates an existing analysis result
func (r *SQLiteRepository) UpdateAnalysisResult(result *domain.AnalysisResult) error {
	logger.Debug("UpdateAnalysisResult: ID=%d, status=%s", result.ID, result.Status)
	query := `UPDATE analysis_results SET result_path = ?, plan_path = ?, execution_path = ?, status = ?, started_at = ?, completed_at = ?, error_message = ?
		WHERE id = ?`

	_, err := r.db.Exec(query,
		result.ResultPath,
		result.PlanPath,
		result.ExecutionPath,
		result.Status,
		result.StartedAt,
		result.CompletedAt,
		result.ErrorMessage,
		result.ID,
	)
	if err != nil {
		logger.Debug("UpdateAnalysisResult: failed: %v", err)
		return fmt.Errorf("failed to update analysis result: %w", err)
	}

	logger.Debug("UpdateAnalysisResult: success")
	return nil
}

// DeleteIssue deletes an issue by key
func (r *SQLiteRepository) DeleteIssue(issueKey string) error {
	query := `DELETE FROM issues WHERE issue_key = ?`
	_, err := r.db.Exec(query, issueKey)
	if err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}
	return nil
}

// DeleteIssueByIDAndChannel은 지정 채널의 단일 이슈를 연관 데이터와 함께 삭제한다.
func (r *SQLiteRepository) DeleteIssueByIDAndChannel(issueID int64, channelIndex int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin delete transaction: %w", err)
	}
	defer tx.Rollback()

	// 분석 결과를 먼저 삭제해 참조 무결성을 보장한다.
	if _, err := tx.Exec(`DELETE FROM analysis_results WHERE issue_id = ?`, issueID); err != nil {
		return fmt.Errorf("failed to delete analysis results: %w", err)
	}

	// 첨부파일 메타데이터를 삭제한다.
	if _, err := tx.Exec(`DELETE FROM attachments WHERE issue_id = ?`, issueID); err != nil {
		return fmt.Errorf("failed to delete attachments: %w", err)
	}

	// 채널 조건을 포함해 대상 이슈 레코드만 삭제한다.
	result, err := tx.Exec(`DELETE FROM issues WHERE id = ? AND channel_index = ?`, issueID, channelIndex)
	if err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check deleted rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("issue not found for delete: id=%d, channel=%d", issueID, channelIndex)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit delete transaction: %w", err)
	}
	return nil
}

// ListAllIssues lists all issues
func (r *SQLiteRepository) ListAllIssues() ([]*domain.IssueRecord, error) {
	query := `SELECT id, issue_key, summary, description, jira_url, md_path, phase, status, channel_index, created_at, updated_at
		FROM issues ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}
	defer rows.Close()

	var issues []*domain.IssueRecord
	for rows.Next() {
		var issue domain.IssueRecord
		err := rows.Scan(
			&issue.ID,
			&issue.IssueKey,
			&issue.Summary,
			&issue.Description,
			&issue.JiraURL,
			&issue.MDPath,
			&issue.Phase,
			&issue.Status,
			&issue.ChannelIndex,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		issues = append(issues, &issue)
	}

	return issues, nil
}

// ListAnalysisResultsByIssue lists all analysis results for an issue
func (r *SQLiteRepository) ListAnalysisResultsByIssue(issueID int64) ([]*domain.AnalysisResult, error) {
	query := `SELECT id, issue_id, analysis_phase, result_path, plan_path, execution_path, status, started_at, completed_at, error_message
		FROM analysis_results WHERE issue_id = ? ORDER BY analysis_phase`

	rows, err := r.db.Query(query, issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis results: %w", err)
	}
	defer rows.Close()

	var results []*domain.AnalysisResult
	for rows.Next() {
		var result domain.AnalysisResult
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&result.ID,
			&result.IssueID,
			&result.AnalysisPhase,
			&result.ResultPath,
			&result.PlanPath,
			&result.ExecutionPath,
			&result.Status,
			&startedAt,
			&completedAt,
			&result.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analysis result: %w", err)
		}

		if startedAt.Valid {
			result.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			result.CompletedAt = &completedAt.Time
		}

		results = append(results, &result)
	}

	return results, nil
}

// CreateAttachment creates a new attachment record
func (r *SQLiteRepository) CreateAttachment(attachment *domain.AttachmentRecord) error {
	query := `INSERT INTO attachments (issue_id, filename, local_path, mime_type, is_video)
		VALUES (?, ?, ?, ?, ?)`

	res, err := r.db.Exec(query,
		attachment.IssueID,
		attachment.Filename,
		attachment.LocalPath,
		attachment.MimeType,
		attachment.IsVideo,
	)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	attachment.ID = id
	return nil
}

// ListAttachmentsByIssue lists all attachments for an issue
func (r *SQLiteRepository) ListAttachmentsByIssue(issueID int64) ([]*domain.AttachmentRecord, error) {
	query := `SELECT id, issue_id, filename, local_path, mime_type, is_video
		FROM attachments WHERE issue_id = ?`

	rows, err := r.db.Query(query, issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*domain.AttachmentRecord
	for rows.Next() {
		var attachment domain.AttachmentRecord
		err := rows.Scan(
			&attachment.ID,
			&attachment.IssueID,
			&attachment.Filename,
			&attachment.LocalPath,
			&attachment.MimeType,
			&attachment.IsVideo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, &attachment)
	}

	return attachments, nil
}

// DeleteAttachmentsByIssue deletes all attachments for an issue
func (r *SQLiteRepository) DeleteAttachmentsByIssue(issueID int64) error {
	query := `DELETE FROM attachments WHERE issue_id = ?`
	_, err := r.db.Exec(query, issueID)
	if err != nil {
		return fmt.Errorf("failed to delete attachments: %w", err)
	}
	return nil
}
