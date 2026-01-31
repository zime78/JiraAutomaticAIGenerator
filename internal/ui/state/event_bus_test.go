package state

import (
	"sync"
	"testing"
	"time"
)

func TestNewEventBus(t *testing.T) {
	eb := NewEventBus()
	if eb == nil {
		t.Fatal("NewEventBus returned nil")
	}
	if eb.subscribers == nil {
		t.Fatal("subscribers map not initialized")
	}
}

func TestEventBus_Subscribe(t *testing.T) {
	eb := NewEventBus()

	eb.Subscribe(EventProgressUpdate, func(event Event) {
		// handler registered
	})

	if len(eb.subscribers[EventProgressUpdate]) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(eb.subscribers[EventProgressUpdate]))
	}
}

func TestEventBus_Publish(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedEvent Event
	eb.Subscribe(EventProgressUpdate, func(event Event) {
		receivedEvent = event
		wg.Done()
	})

	eb.Publish(Event{
		Type:    EventProgressUpdate,
		Channel: 1,
		Data:    "test data",
	})

	wg.Wait()

	if receivedEvent.Type != EventProgressUpdate {
		t.Errorf("expected EventProgressUpdate, got %v", receivedEvent.Type)
	}
	if receivedEvent.Channel != 1 {
		t.Errorf("expected channel 1, got %d", receivedEvent.Channel)
	}
	if receivedEvent.Data != "test data" {
		t.Errorf("expected 'test data', got %v", receivedEvent.Data)
	}
	if receivedEvent.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
}

func TestEventBus_PublishSync(t *testing.T) {
	eb := NewEventBus()

	called := false
	eb.Subscribe(EventPhaseChange, func(event Event) {
		called = true
	})

	eb.PublishSync(Event{
		Type:    EventPhaseChange,
		Channel: 0,
		Data:    PhaseAnalyzing,
	})

	if !called {
		t.Error("handler was not called synchronously")
	}
}

func TestEventBus_SubscribeMultiple(t *testing.T) {
	eb := NewEventBus()

	callCount := 0
	eb.SubscribeMultiple([]EventType{EventProgressUpdate, EventPhaseChange}, func(event Event) {
		callCount++
	})

	if len(eb.subscribers[EventProgressUpdate]) != 1 {
		t.Error("EventProgressUpdate subscriber not added")
	}
	if len(eb.subscribers[EventPhaseChange]) != 1 {
		t.Error("EventPhaseChange subscriber not added")
	}
}

func TestEventBus_PublishProgress(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedData ProgressData
	eb.Subscribe(EventProgressUpdate, func(event Event) {
		if data, ok := event.Data.(ProgressData); ok {
			receivedData = data
		}
		wg.Done()
	})

	eb.PublishProgress(0, PhaseExtractingFrames, 3, 10, "프레임 추출 중...")

	wg.Wait()

	if receivedData.Phase != PhaseExtractingFrames {
		t.Errorf("expected PhaseExtractingFrames, got %v", receivedData.Phase)
	}
	if receivedData.Step != 3 {
		t.Errorf("expected step 3, got %d", receivedData.Step)
	}
	if receivedData.Total != 10 {
		t.Errorf("expected total 10, got %d", receivedData.Total)
	}
	if receivedData.Progress != 0.3 {
		t.Errorf("expected progress 0.3, got %f", receivedData.Progress)
	}
	if receivedData.Message != "프레임 추출 중..." {
		t.Errorf("expected '프레임 추출 중...', got %s", receivedData.Message)
	}
}

func TestEventBus_PublishLog(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedData LogData
	eb.Subscribe(EventLogAdded, func(event Event) {
		if data, ok := event.Data.(LogData); ok {
			receivedData = data
		}
		wg.Done()
	})

	eb.PublishLog(1, LogWarning, "테스트 경고", "test_source")

	wg.Wait()

	if receivedData.Level != LogWarning {
		t.Errorf("expected LogWarning, got %v", receivedData.Level)
	}
	if receivedData.Message != "테스트 경고" {
		t.Errorf("expected '테스트 경고', got %s", receivedData.Message)
	}
	if receivedData.Source != "test_source" {
		t.Errorf("expected 'test_source', got %s", receivedData.Source)
	}
}

func TestEventBus_Clear(t *testing.T) {
	eb := NewEventBus()

	eb.Subscribe(EventProgressUpdate, func(event Event) {})
	eb.Subscribe(EventPhaseChange, func(event Event) {})

	eb.Clear()

	if len(eb.subscribers) != 0 {
		t.Errorf("expected 0 subscribers after clear, got %d", len(eb.subscribers))
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus()

	eb.Subscribe(EventProgressUpdate, func(event Event) {})
	eb.Subscribe(EventPhaseChange, func(event Event) {})

	eb.Unsubscribe(EventProgressUpdate)

	if len(eb.subscribers[EventProgressUpdate]) != 0 {
		t.Error("EventProgressUpdate subscribers should be removed")
	}
	if len(eb.subscribers[EventPhaseChange]) != 1 {
		t.Error("EventPhaseChange subscribers should remain")
	}
}

func TestProcessPhase_String(t *testing.T) {
	tests := []struct {
		phase    ProcessPhase
		expected string
	}{
		{PhaseIdle, "대기"},
		{PhaseFetchingIssue, "이슈 조회"},
		{PhaseDownloadingAttachments, "첨부파일 다운로드"},
		{PhaseExtractingFrames, "프레임 추출"},
		{PhaseGeneratingDocument, "문서 생성"},
		{PhaseAnalyzing, "AI 분석"},
		{PhaseCompleted, "완료"},
		{PhaseFailed, "실패"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.phase.String(); got != tt.expected {
				t.Errorf("ProcessPhase.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProcessPhase_Progress(t *testing.T) {
	tests := []struct {
		phase    ProcessPhase
		expected float64
	}{
		{PhaseIdle, 0.0},
		{PhaseFetchingIssue, 0.1},
		{PhaseDownloadingAttachments, 0.3},
		{PhaseExtractingFrames, 0.5},
		{PhaseGeneratingDocument, 0.7},
		{PhaseAnalyzing, 0.8},
		{PhaseCompleted, 1.0},
		{PhaseFailed, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.phase.String(), func(t *testing.T) {
			if got := tt.phase.Progress(); got != tt.expected {
				t.Errorf("ProcessPhase.Progress() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogDebug, "DEBUG"},
		{LogInfo, "INFO"},
		{LogWarning, "WARNING"},
		{LogError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	eb := NewEventBus()

	var mu sync.Mutex
	callCount := 0

	eb.Subscribe(EventProgressUpdate, func(event Event) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			eb.Publish(Event{
				Type:    EventProgressUpdate,
				Channel: i % 3,
				Data:    i,
			})
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // 비동기 핸들러 완료 대기

	mu.Lock()
	if callCount != 100 {
		t.Errorf("expected 100 calls, got %d", callCount)
	}
	mu.Unlock()
}
