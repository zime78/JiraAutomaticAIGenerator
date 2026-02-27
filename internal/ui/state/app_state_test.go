package state

import (
	"errors"
	"sync"
	"testing"
)

func TestNewAppState(t *testing.T) {
	state := NewAppState(nil, nil)

	if state == nil {
		t.Fatal("NewAppState returned nil")
	}

	if state.Channel == nil {
		t.Error("Channel not initialized")
	}

	if state.EventBus == nil {
		t.Error("EventBus not initialized")
	}

	if state.GlobalStatus != "준비됨" {
		t.Errorf("expected '준비됨', got '%s'", state.GlobalStatus)
	}
}

func TestNewChannelStateData(t *testing.T) {
	ch := NewChannelStateData(0, "테스트 채널")

	if ch.Index != 0 {
		t.Errorf("expected index 0, got %d", ch.Index)
	}
	if ch.Name != "테스트 채널" {
		t.Errorf("expected '테스트 채널', got '%s'", ch.Name)
	}
	if ch.Phase != PhaseIdle {
		t.Errorf("expected PhaseIdle, got %v", ch.Phase)
	}
	if len(ch.Steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(ch.Steps))
	}
}

func TestAppState_GetChannel(t *testing.T) {
	state := NewAppState(nil, nil)

	ch := state.GetChannel(0)
	if ch == nil {
		t.Fatal("GetChannel(0) returned nil")
	}

	// 인덱스와 무관하게 단일 채널을 반환한다
	ch2 := state.GetChannel(1)
	if ch2 != ch {
		t.Error("GetChannel should return the same channel regardless of index")
	}
}

func TestAppState_UpdatePhase(t *testing.T) {
	state := NewAppState(nil, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedPhase ProcessPhase
	state.EventBus.Subscribe(EventPhaseChange, func(event Event) {
		if phase, ok := event.Data.(ProcessPhase); ok {
			receivedPhase = phase
		}
		wg.Done()
	})

	state.UpdatePhase(0, PhaseExtractingFrames)

	wg.Wait()

	ch := state.GetChannel(0)
	if ch.Phase != PhaseExtractingFrames {
		t.Errorf("expected PhaseExtractingFrames, got %v", ch.Phase)
	}
	if receivedPhase != PhaseExtractingFrames {
		t.Errorf("expected event phase PhaseExtractingFrames, got %v", receivedPhase)
	}

	// 이전 단계들이 완료로 표시되었는지 확인
	for i := 0; i < 2; i++ {
		if ch.Steps[i].Status != StepCompleted {
			t.Errorf("step %d should be completed", i)
		}
	}
	if ch.Steps[2].Status != StepRunning {
		t.Error("step 2 should be running")
	}
}

func TestAppState_UpdateProgress(t *testing.T) {
	state := NewAppState(nil, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	state.EventBus.Subscribe(EventProgressUpdate, func(event Event) {
		wg.Done()
	})

	// 먼저 단계 설정
	state.UpdatePhase(0, PhaseDownloadingAttachments)

	// 진행률 업데이트
	state.UpdateProgress(0, 5, 10, "다운로드 중...")

	wg.Wait()

	ch := state.GetChannel(0)
	if ch.Progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", ch.Progress)
	}
}

func TestAppState_AddLog(t *testing.T) {
	state := NewAppState(nil, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	state.EventBus.Subscribe(EventLogAdded, func(event Event) {
		wg.Done()
	})

	state.AddLog(LogInfo, "테스트 로그", "test")

	wg.Wait()

	ch := state.GetChannel(0)
	if len(ch.Logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(ch.Logs))
	}
	if ch.Logs[0].Message != "테스트 로그" {
		t.Errorf("expected '테스트 로그', got '%s'", ch.Logs[0].Message)
	}
}

func TestAppState_AddJob(t *testing.T) {
	state := NewAppState(nil, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	state.EventBus.Subscribe(EventQueueUpdated, func(event Event) {
		wg.Done()
	})

	job := &JobData{
		ID:       "job-1",
		IssueKey: "TEST-123",
	}
	state.AddJob(job)

	wg.Wait()

	ch := state.GetChannel(0)
	if len(ch.Queue) != 1 {
		t.Errorf("expected 1 job in queue, got %d", len(ch.Queue))
	}
	if ch.Queue[0].IssueKey != "TEST-123" {
		t.Errorf("expected 'TEST-123', got '%s'", ch.Queue[0].IssueKey)
	}
}

func TestAppState_CompleteJob(t *testing.T) {
	state := NewAppState(nil, nil)

	// 현재 작업 설정
	state.Channel.CurrentJob = &JobData{
		ID:       "job-1",
		IssueKey: "TEST-123",
		Status:   JobRunning,
	}
	state.Channel.IsRunning = true

	var wg sync.WaitGroup
	wg.Add(2)

	state.EventBus.Subscribe(EventJobCompleted, func(event Event) {
		wg.Done()
	})
	state.EventBus.Subscribe(EventHistoryAdded, func(event Event) {
		wg.Done()
	})

	state.CompleteJob("job-1", nil)

	wg.Wait()

	ch := state.GetChannel(0)
	if ch.CurrentJob != nil {
		t.Error("CurrentJob should be nil after completion")
	}
	if ch.IsRunning {
		t.Error("IsRunning should be false after completion")
	}
	if len(state.CompletedJobs) != 1 {
		t.Errorf("expected 1 completed job, got %d", len(state.CompletedJobs))
	}
}

func TestAppState_FailJob(t *testing.T) {
	state := NewAppState(nil, nil)

	// 현재 작업 설정
	state.Channel.CurrentJob = &JobData{
		ID:       "job-1",
		IssueKey: "TEST-123",
		Status:   JobRunning,
	}
	state.Channel.IsRunning = true
	state.Channel.Phase = PhaseAnalyzing
	state.Channel.Steps[4].Status = StepRunning

	var wg sync.WaitGroup
	wg.Add(1)

	state.EventBus.Subscribe(EventJobFailed, func(event Event) {
		wg.Done()
	})

	testErr := errors.New("테스트 에러")
	state.FailJob("job-1", testErr)

	wg.Wait()

	ch := state.GetChannel(0)
	if ch.CurrentJob != nil {
		t.Error("CurrentJob should be nil after failure")
	}
	if ch.Phase != PhaseFailed {
		t.Errorf("Phase should be PhaseFailed, got %v", ch.Phase)
	}
	if len(state.CompletedJobs) != 1 {
		t.Errorf("expected 1 completed job (failed), got %d", len(state.CompletedJobs))
	}
	if state.CompletedJobs[0].Error != "테스트 에러" {
		t.Errorf("expected error '테스트 에러', got '%s'", state.CompletedJobs[0].Error)
	}
}

func TestAppState_ResetChannel(t *testing.T) {
	state := NewAppState(nil, nil)

	// 상태 변경
	ch := state.GetChannel(0)
	ch.Phase = PhaseCompleted
	ch.Progress = 1.0
	ch.IssueInfo = "테스트 정보"

	// 리셋
	state.ResetChannel()

	ch = state.GetChannel(0)
	if ch.Phase != PhaseIdle {
		t.Errorf("expected PhaseIdle after reset, got %v", ch.Phase)
	}
	if ch.Progress != 0 {
		t.Errorf("expected progress 0 after reset, got %f", ch.Progress)
	}
	if ch.IssueInfo != "" {
		t.Errorf("expected empty IssueInfo after reset, got '%s'", ch.IssueInfo)
	}
}

func TestAppState_ConcurrentAccess(t *testing.T) {
	state := NewAppState(nil, nil)

	var wg sync.WaitGroup

	// 동시에 여러 업데이트
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			state.UpdatePhase(0, ProcessPhase((i%7)+1))
		}(i)
		go func(i int) {
			defer wg.Done()
			state.AddLog(LogInfo, "로그", "test")
		}(i)
		go func(i int) {
			defer wg.Done()
			state.UpdateProgress(0, i, 100, "진행 중")
		}(i)
	}

	wg.Wait()

	// 패닉 없이 완료되면 성공
}

func TestStepStatus_String(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepPending, "대기"},
		{StepRunning, "진행 중"},
		{StepCompleted, "완료"},
		{StepFailed, "실패"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("StepStatus.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		status   JobStatus
		expected string
	}{
		{JobPending, "대기"},
		{JobRunning, "실행 중"},
		{JobCompleted, "완료"},
		{JobFailed, "실패"},
		{JobCancelled, "취소됨"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("JobStatus.String() = %v, want %v", got, tt.expected)
		}
	}
}
