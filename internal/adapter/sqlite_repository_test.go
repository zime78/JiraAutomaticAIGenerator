package adapter

import (
	"jira-ai-generator/internal/domain"
	"os"
	"testing"
	"time"
)

func TestNewSQLiteRepository(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Act
	repo, err := NewSQLiteRepository(dbPath)

	// Assert
	if err != nil {
		t.Fatalf("NewSQLiteRepository failed: %v", err)
	}
	defer repo.Close()

	if repo == nil {
		t.Fatal("Expected repository to be created")
	}
}

func TestCreateAndGetIssue(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issue := &domain.IssueRecord{
		IssueKey:     "TEST-123",
		Summary:      "Test Issue",
		Description:  "Test Description",
		JiraURL:      "https://test.atlassian.net/browse/TEST-123",
		MDPath:       "/path/to/test.md",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 1,
	}

	// Act
	err = repo.CreateIssue(issue)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Assert
	retrieved, err := repo.GetIssue("TEST-123")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}

	if retrieved.IssueKey != issue.IssueKey {
		t.Errorf("Expected IssueKey %s, got %s", issue.IssueKey, retrieved.IssueKey)
	}
	if retrieved.Summary != issue.Summary {
		t.Errorf("Expected Summary %s, got %s", issue.Summary, retrieved.Summary)
	}
	if retrieved.Phase != issue.Phase {
		t.Errorf("Expected Phase %d, got %d", issue.Phase, retrieved.Phase)
	}
	if retrieved.ID == 0 {
		t.Error("Expected ID to be set")
	}
}

func TestUpdateIssue(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issue := &domain.IssueRecord{
		IssueKey:     "TEST-124",
		Summary:      "Original Summary",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 1,
	}

	err = repo.CreateIssue(issue)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Act
	issue.Summary = "Updated Summary"
	issue.Phase = 2
	err = repo.UpdateIssue(issue)
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	// Assert
	retrieved, err := repo.GetIssue("TEST-124")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}

	if retrieved.Summary != "Updated Summary" {
		t.Errorf("Expected Summary 'Updated Summary', got %s", retrieved.Summary)
	}
	if retrieved.Phase != 2 {
		t.Errorf("Expected Phase 2, got %d", retrieved.Phase)
	}
}

func TestListIssuesByPhase(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issues := []*domain.IssueRecord{
		{IssueKey: "TEST-1", Phase: 1, Status: "active", ChannelIndex: 1},
		{IssueKey: "TEST-2", Phase: 1, Status: "active", ChannelIndex: 1},
		{IssueKey: "TEST-3", Phase: 2, Status: "active", ChannelIndex: 1},
	}

	for _, issue := range issues {
		if err := repo.CreateIssue(issue); err != nil {
			t.Fatalf("CreateIssue failed: %v", err)
		}
	}

	// Act
	phase1Issues, err := repo.ListIssuesByPhase(1)
	if err != nil {
		t.Fatalf("ListIssuesByPhase failed: %v", err)
	}

	// Assert
	if len(phase1Issues) != 2 {
		t.Errorf("Expected 2 issues in phase 1, got %d", len(phase1Issues))
	}
}

func TestListIssuesByChannel(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issues := []*domain.IssueRecord{
		{IssueKey: "TEST-1", Phase: 1, Status: "active", ChannelIndex: 1},
		{IssueKey: "TEST-2", Phase: 1, Status: "active", ChannelIndex: 2},
		{IssueKey: "TEST-3", Phase: 1, Status: "active", ChannelIndex: 1},
	}

	for _, issue := range issues {
		if err := repo.CreateIssue(issue); err != nil {
			t.Fatalf("CreateIssue failed: %v", err)
		}
	}

	// Act
	channel1Issues, err := repo.ListIssuesByChannel(1)
	if err != nil {
		t.Fatalf("ListIssuesByChannel failed: %v", err)
	}

	// Assert
	if len(channel1Issues) != 2 {
		t.Errorf("Expected 2 issues in channel 1, got %d", len(channel1Issues))
	}
}

func TestCreateAndGetAnalysisResult(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issue := &domain.IssueRecord{
		IssueKey:     "TEST-200",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 1,
	}
	if err := repo.CreateIssue(issue); err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	now := time.Now()
	result := &domain.AnalysisResult{
		IssueID:       issue.ID,
		AnalysisPhase: 1,
		ResultPath:    "/path/to/result.md",
		PlanPath:      "/path/to/plan.md",
		Status:        "completed",
		StartedAt:     &now,
		CompletedAt:   &now,
	}

	// Act
	err = repo.CreateAnalysisResult(result)
	if err != nil {
		t.Fatalf("CreateAnalysisResult failed: %v", err)
	}

	// Assert
	retrieved, err := repo.GetAnalysisResult(issue.ID, 1)
	if err != nil {
		t.Fatalf("GetAnalysisResult failed: %v", err)
	}

	if retrieved.ResultPath != result.ResultPath {
		t.Errorf("Expected ResultPath %s, got %s", result.ResultPath, retrieved.ResultPath)
	}
	if retrieved.Status != result.Status {
		t.Errorf("Expected Status %s, got %s", result.Status, retrieved.Status)
	}
}

func TestCreateAndListAttachments(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issue := &domain.IssueRecord{
		IssueKey:     "TEST-300",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 1,
	}
	if err := repo.CreateIssue(issue); err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	attachments := []*domain.AttachmentRecord{
		{
			IssueID:   issue.ID,
			Filename:  "video.mp4",
			LocalPath: "/path/to/video.mp4",
			MimeType:  "video/mp4",
			IsVideo:   true,
		},
		{
			IssueID:   issue.ID,
			Filename:  "image.png",
			LocalPath: "/path/to/image.png",
			MimeType:  "image/png",
			IsVideo:   false,
		},
	}

	// Act
	for _, att := range attachments {
		if err := repo.CreateAttachment(att); err != nil {
			t.Fatalf("CreateAttachment failed: %v", err)
		}
	}

	// Assert
	retrieved, err := repo.ListAttachmentsByIssue(issue.ID)
	if err != nil {
		t.Fatalf("ListAttachmentsByIssue failed: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Expected 2 attachments, got %d", len(retrieved))
	}
}

func TestDeleteIssue(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issue := &domain.IssueRecord{
		IssueKey:     "TEST-400",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 1,
	}
	if err := repo.CreateIssue(issue); err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Act
	err = repo.DeleteIssue("TEST-400")
	if err != nil {
		t.Fatalf("DeleteIssue failed: %v", err)
	}

	// Assert
	_, err = repo.GetIssue("TEST-400")
	if err == nil {
		t.Error("Expected error when getting deleted issue")
	}
}

func TestCreateIssue_AllowsSameIssueKeyAcrossChannels(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issueChannel1 := &domain.IssueRecord{
		IssueKey:     "TEST-500",
		Summary:      "channel 1 issue",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 1,
	}
	issueChannel2 := &domain.IssueRecord{
		IssueKey:     "TEST-500",
		Summary:      "channel 2 issue",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 2,
	}

	// Act
	if err := repo.CreateIssue(issueChannel1); err != nil {
		t.Fatalf("CreateIssue(channel1) failed: %v", err)
	}
	if err := repo.CreateIssue(issueChannel2); err != nil {
		t.Fatalf("CreateIssue(channel2) failed: %v", err)
	}

	// Assert
	got1, err := repo.GetIssueByKeyAndChannel("TEST-500", 1)
	if err != nil {
		t.Fatalf("GetIssueByKeyAndChannel(channel1) failed: %v", err)
	}
	got2, err := repo.GetIssueByKeyAndChannel("TEST-500", 2)
	if err != nil {
		t.Fatalf("GetIssueByKeyAndChannel(channel2) failed: %v", err)
	}
	if got1.ID == got2.ID {
		t.Fatalf("Expected different records per channel, got same ID=%d", got1.ID)
	}
}

func TestUpsertIssue_UpdatesExistingByChannel(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	original := &domain.IssueRecord{
		IssueKey:     "TEST-501",
		Summary:      "original",
		Phase:        1,
		Status:       "active",
		ChannelIndex: 1,
	}
	if err := repo.CreateIssue(original); err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	upsert := &domain.IssueRecord{
		IssueKey:     "TEST-501",
		Summary:      "updated by upsert",
		Phase:        2,
		Status:       "active",
		ChannelIndex: 1,
	}

	// Act
	if err := repo.UpsertIssue(upsert); err != nil {
		t.Fatalf("UpsertIssue failed: %v", err)
	}

	// Assert
	got, err := repo.GetIssueByKeyAndChannel("TEST-501", 1)
	if err != nil {
		t.Fatalf("GetIssueByKeyAndChannel failed: %v", err)
	}
	if got.Summary != "updated by upsert" {
		t.Fatalf("Expected updated summary, got %s", got.Summary)
	}
	if got.Phase != 2 {
		t.Fatalf("Expected phase 2, got %d", got.Phase)
	}
}

func TestDeleteIssueByIDAndChannel_DeletesOnlyTargetIssueAndRelatedData(t *testing.T) {
	// Arrange
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	issueCh0 := &domain.IssueRecord{
		IssueKey:     "TEST-700",
		Summary:      "channel 0 issue",
		Phase:        2,
		Status:       "active",
		ChannelIndex: 0,
	}
	issueCh1 := &domain.IssueRecord{
		IssueKey:     "TEST-700",
		Summary:      "channel 1 issue",
		Phase:        2,
		Status:       "active",
		ChannelIndex: 1,
	}

	if err := repo.CreateIssue(issueCh0); err != nil {
		t.Fatalf("CreateIssue(channel0) failed: %v", err)
	}
	if err := repo.CreateIssue(issueCh1); err != nil {
		t.Fatalf("CreateIssue(channel1) failed: %v", err)
	}

	if err := repo.CreateAnalysisResult(&domain.AnalysisResult{
		IssueID:       issueCh0.ID,
		AnalysisPhase: 2,
		Status:        "completed",
	}); err != nil {
		t.Fatalf("CreateAnalysisResult failed: %v", err)
	}
	if err := repo.CreateAttachment(&domain.AttachmentRecord{
		IssueID:   issueCh0.ID,
		Filename:  "test.png",
		LocalPath: "/tmp/test.png",
		MimeType:  "image/png",
		IsVideo:   false,
	}); err != nil {
		t.Fatalf("CreateAttachment failed: %v", err)
	}

	// Act
	if err := repo.DeleteIssueByIDAndChannel(issueCh0.ID, 0); err != nil {
		t.Fatalf("DeleteIssueByIDAndChannel failed: %v", err)
	}

	// Assert - 대상 채널 이슈는 삭제되어야 한다.
	if _, err := repo.GetIssueByKeyAndChannel("TEST-700", 0); err == nil {
		t.Fatal("expected channel 0 issue to be deleted")
	}

	// Assert - 다른 채널 이슈는 유지되어야 한다.
	if _, err := repo.GetIssueByKeyAndChannel("TEST-700", 1); err != nil {
		t.Fatalf("expected channel 1 issue to remain, got error: %v", err)
	}

	// Assert - 연관된 분석 결과/첨부도 함께 삭제되어야 한다.
	results, err := repo.ListAnalysisResultsByIssue(issueCh0.ID)
	if err != nil {
		t.Fatalf("ListAnalysisResultsByIssue failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 analysis results after delete, got %d", len(results))
	}

	attachments, err := repo.ListAttachmentsByIssue(issueCh0.ID)
	if err != nil {
		t.Fatalf("ListAttachmentsByIssue failed: %v", err)
	}
	if len(attachments) != 0 {
		t.Fatalf("expected 0 attachments after delete, got %d", len(attachments))
	}
}
