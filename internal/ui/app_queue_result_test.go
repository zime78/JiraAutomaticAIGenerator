package ui

import "testing"

// TestRecordQueueJobByOutcome_CompletedOnlyOnSuccess는 성공 시에만 Completed 목록이 증가하는지 검증한다.
func TestRecordQueueJobByOutcome_CompletedOnlyOnSuccess(t *testing.T) {
	queue := &AnalysisQueue{}

	recordQueueJobByOutcome(queue, &AnalysisJob{IssueKey: "ITSM-100"}, QueueJobOutcomeCompleted)

	if got := len(queue.Completed); got != 1 {
		t.Fatalf("completed length = %d, want 1", got)
	}
	if got := len(queue.Failed); got != 0 {
		t.Fatalf("failed length = %d, want 0", got)
	}
	if got := len(queue.Cancelled); got != 0 {
		t.Fatalf("cancelled length = %d, want 0", got)
	}
}

// TestRecordQueueJobByOutcome_FailedDoesNotIncreaseCompleted는 실패 시 Completed가 증가하지 않는지 검증한다.
func TestRecordQueueJobByOutcome_FailedDoesNotIncreaseCompleted(t *testing.T) {
	queue := &AnalysisQueue{
		Completed: []*AnalysisJob{{IssueKey: "ITSM-100"}},
	}

	recordQueueJobByOutcome(queue, &AnalysisJob{IssueKey: "ITSM-101"}, QueueJobOutcomeFailed)

	if got := len(queue.Completed); got != 1 {
		t.Fatalf("completed length = %d, want 1", got)
	}
	if got := len(queue.Failed); got != 1 {
		t.Fatalf("failed length = %d, want 1", got)
	}
	if got := queue.Failed[0].IssueKey; got != "ITSM-101" {
		t.Fatalf("failed[0].IssueKey = %s, want ITSM-101", got)
	}
}

// TestRecordQueueJobByOutcome_CancelledDoesNotIncreaseCompleted는 중단 시 Completed가 증가하지 않는지 검증한다.
func TestRecordQueueJobByOutcome_CancelledDoesNotIncreaseCompleted(t *testing.T) {
	queue := &AnalysisQueue{
		Completed: []*AnalysisJob{{IssueKey: "ITSM-100"}},
	}

	recordQueueJobByOutcome(queue, &AnalysisJob{IssueKey: "ITSM-102"}, QueueJobOutcomeCancelled)

	if got := len(queue.Completed); got != 1 {
		t.Fatalf("completed length = %d, want 1", got)
	}
	if got := len(queue.Cancelled); got != 1 {
		t.Fatalf("cancelled length = %d, want 1", got)
	}
	if got := queue.Cancelled[0].IssueKey; got != "ITSM-102" {
		t.Fatalf("cancelled[0].IssueKey = %s, want ITSM-102", got)
	}
}
