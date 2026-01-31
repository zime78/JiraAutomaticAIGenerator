package state

import (
	"errors"
	"sync"
	"testing"
)

func TestNewAppState(t *testing.T) {
	state := NewAppState()

	if state == nil {
		t.Fatal("NewAppState returned nil")
	}

	if len(state.Channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(state.Channels))
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
	state := NewAppState()

	ch := state.GetChannel(1)
	if ch == nil {
		t.Fatal("GetChannel(1) returned nil")
	}
	if ch.Index != 1 {
		t.Errorf("expected index 1, got %d", ch.Index)
	}

	// 범위 밖 인덱스
	if state.GetChannel(-1) != nil {
		t.Error("GetChannel(-1) should return nil")
	}
	if state.GetChannel(3) != nil {
		t.Error("GetChannel(3) should return nil")
	}
}

func TestAppState_SetActiveChannel(t *testing.T) {
	state := NewAppState()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedChannel int
	state.EventBus.Subscribe(EventChannelSwitch, func(event Event) {
		receivedChannel = event.Channel
		wg.Done()
	})

	state.SetActiveChannel(2)

	wg.Wait()

	if state.ActiveChannel != 2 {
		t.Errorf("expected ActiveChannel 2, got %d", state.ActiveChannel)
	}
	if receivedChannel != 2 {
		t.Errorf("expected event channel 2, got %d", receivedChannel)
	}
}

func TestAppState_UpdatePhase(t *testing.T) {
	state := NewAppState()

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
	state := NewAppState()

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
	state := NewAppState()

	var wg sync.WaitGroup
	wg.Add(1)

	state.EventBus.Subscribe(EventLogAdded, func(event Event) {
		wg.Done()
	})

	state.AddLog(1, LogInfo, "테스트 로그", "test")

	wg.Wait()

	ch := state.GetChannel(1)
	if len(ch.Logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(ch.Logs))
	}
	if ch.Logs[0].Message != "테스트 로그" {
		t.Errorf("expected '테스트 로그', got '%s'", ch.Logs[0].Message)
	}
}

func TestAppState_AddJob(t *testing.T) {
	state := NewAppState()

	var wg sync.WaitGroup
	wg.Add(1)

	state.EventBus.Subscribe(EventQueueUpdated, func(event Event) {
		wg.Done()
	})

	job := &JobData{
		ID:       "job-1",
		IssueKey: "TEST-123",
	}
	state.AddJob(0, job)

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
	state := NewAppState()

	// 현재 작업 설정
	state.Channels[0].CurrentJob = &JobData{
		ID:       "job-1",
		IssueKey: "TEST-123",
		Status:   JobRunning,
	}
	state.Channels[0].IsRunning = true

	var wg sync.WaitGroup
	wg.Add(2)

	state.EventBus.Subscribe(EventJobCompleted, func(event Event) {
		wg.Done()
	})
	state.EventBus.Subscribe(EventHistoryAdded, func(event Event) {
		wg.Done()
	})

	state.CompleteJob(0, "job-1", nil)

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
	state := NewAppState()

	// 현재 작업 설정
	state.Channels[0].CurrentJob = &JobData{
		ID:       "job-1",
		IssueKey: "TEST-123",
		Status:   JobRunning,
	}
	state.Channels[0].IsRunning = true
	state.Channels[0].Phase = PhaseAnalyzing
	state.Channels[0].Steps[4].Status = StepRunning

	var wg sync.WaitGroup
	wg.Add(1)

	state.EventBus.Subscribe(EventJobFailed, func(event Event) {
		wg.Done()
	})

	testErr := errors.New("테스트 에러")
	state.FailJob(0, "job-1", testErr)

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
	state := NewAppState()

	// 상태 변경
	ch := state.GetChannel(0)
	ch.Phase = PhaseCompleted
	ch.Progress = 1.0
	ch.IssueInfo = "테스트 정보"

	// 리셋
	state.ResetChannel(0)

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
	state := NewAppState()

	var wg sync.WaitGroup

	// 동시에 여러 채널 업데이트
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			state.UpdatePhase(i%3, ProcessPhase((i%7)+1))
		}(i)
		go func(i int) {
			defer wg.Done()
			state.AddLog(i%3, LogInfo, "로그", "test")
		}(i)
		go func(i int) {
			defer wg.Done()
			state.UpdateProgress(i%3, i, 100, "진행 중")
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
