package components

import (
	"testing"
	"time"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/ui/state"
)

func TestNewAnalysisSelector(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	channelIdx := 0

	// Act
	selector := NewAnalysisSelector(eventBus, channelIdx)

	// Assert
	if selector == nil {
		t.Fatal("Expected selector to be created")
	}

	if selector.eventBus != eventBus {
		t.Error("Expected eventBus to be set")
	}

	if selector.channelIdx != channelIdx {
		t.Errorf("Expected channelIdx to be %d, got %d", channelIdx, selector.channelIdx)
	}

	if selector.phase2List == nil {
		t.Error("Expected phase2List to be initialized")
	}

	if selector.phase3List == nil {
		t.Error("Expected phase3List to be initialized")
	}

	if selector.startPhase2 == nil {
		t.Error("Expected startPhase2 button to be initialized")
	}

	if selector.startPhase3 == nil {
		t.Error("Expected startPhase3 button to be initialized")
	}

	if !selector.startPhase2.Disabled() {
		t.Error("Expected startPhase2 button to be initially disabled")
	}

	if !selector.startPhase3.Disabled() {
		t.Error("Expected startPhase3 button to be initially disabled")
	}
}

func TestAnalysisSelector_SetPhase1Items(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 0)

	items := []*domain.IssueRecord{
		{
			ID:       1,
			IssueKey: "ISSUE-1",
			Summary:  "Test Issue 1",
			Phase:    1,
		},
		{
			ID:       2,
			IssueKey: "ISSUE-2",
			Summary:  "Test Issue 2",
			Phase:    1,
		},
	}

	// Act
	selector.SetPhase1Items(items)

	// Assert
	if len(selector.phase2List.items) != 2 {
		t.Errorf("Expected 2 items in phase2List, got %d", len(selector.phase2List.items))
	}

	if selector.selectedPhase2Item != nil {
		t.Error("Expected selectedPhase2Item to be nil after setting items")
	}

	if !selector.startPhase2.Disabled() {
		t.Error("Expected startPhase2 button to be disabled after setting items")
	}
}

func TestAnalysisSelector_SetPhase2Items(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 0)

	items := []*domain.IssueRecord{
		{
			ID:       1,
			IssueKey: "ISSUE-1",
			Summary:  "Test Issue 1",
			Phase:    2,
		},
	}

	// Act
	selector.SetPhase2Items(items)

	// Assert
	if len(selector.phase3List.items) != 1 {
		t.Errorf("Expected 1 item in phase3List, got %d", len(selector.phase3List.items))
	}

	if selector.selectedPhase3Item != nil {
		t.Error("Expected selectedPhase3Item to be nil after setting items")
	}

	if !selector.startPhase3.Disabled() {
		t.Error("Expected startPhase3 button to be disabled after setting items")
	}
}

func TestAnalysisSelector_OnStartPhase2(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 1)

	testItem := &domain.IssueRecord{
		ID:       1,
		IssueKey: "ISSUE-1",
		Summary:  "Test Issue 1",
		Phase:    1,
	}

	// 항목 설정 및 선택
	selector.SetPhase1Items([]*domain.IssueRecord{testItem})
	selector.phase2List.selected[1] = true // ID 1 선택

	// 이벤트 수신 확인용 채널
	phaseChangeReceived := make(chan bool, 1)
	jobStartedReceived := make(chan bool, 1)
	var receivedPhaseEvent state.Event
	var receivedJobEvent state.Event

	eventBus.Subscribe(state.EventPhaseChange, func(event state.Event) {
		receivedPhaseEvent = event
		phaseChangeReceived <- true
	})

	eventBus.Subscribe(state.EventJobStarted, func(event state.Event) {
		receivedJobEvent = event
		jobStartedReceived <- true
	})

	// Act
	selector.onStartPhase2()

	// Assert - Phase Change Event
	select {
	case <-phaseChangeReceived:
		if receivedPhaseEvent.Type != state.EventPhaseChange {
			t.Errorf("Expected event type %v, got %v", state.EventPhaseChange, receivedPhaseEvent.Type)
		}

		if receivedPhaseEvent.Channel != 1 {
			t.Errorf("Expected channel 1, got %d", receivedPhaseEvent.Channel)
		}

		if receivedPhaseEvent.Data.(state.ProcessPhase) != state.PhaseAIPlanGeneration {
			t.Errorf("Expected PhaseAIPlanGeneration, got %v", receivedPhaseEvent.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected EventPhaseChange to be published")
	}

	// Assert - Job Started Event
	select {
	case <-jobStartedReceived:
		if receivedJobEvent.Type != state.EventJobStarted {
			t.Errorf("Expected event type %v, got %v", state.EventJobStarted, receivedJobEvent.Type)
		}

		if receivedJobEvent.Channel != 1 {
			t.Errorf("Expected channel 1, got %d", receivedJobEvent.Channel)
		}

		data := receivedJobEvent.Data.(map[string]interface{})
		if data["phase"] != "phase2" {
			t.Errorf("Expected phase2, got %v", data["phase"])
		}

		// 다중 선택 지원으로 issueRecords 배열 확인
		if issueRecords, ok := data["issueRecords"].([]*domain.IssueRecord); ok {
			if len(issueRecords) != 1 {
				t.Errorf("Expected 1 issue record, got %d", len(issueRecords))
			}
			if issueRecords[0].IssueKey != "ISSUE-1" {
				t.Errorf("Expected ISSUE-1, got %v", issueRecords[0].IssueKey)
			}
		} else {
			t.Error("Expected issueRecords to be []*domain.IssueRecord")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected EventJobStarted to be published")
	}

	// Assert - Button disabled
	if !selector.startPhase2.Disabled() {
		t.Error("Expected startPhase2 button to be disabled after execution")
	}

	// Assert - Status updated
	if selector.phase2Status.Text != "AI 플랜 생성 중..." {
		t.Errorf("Expected status 'AI 플랜 생성 중...', got '%s'", selector.phase2Status.Text)
	}
}

func TestAnalysisSelector_OnStartPhase3(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 2)

	testItem := &domain.IssueRecord{
		ID:       1,
		IssueKey: "ISSUE-1",
		Summary:  "Test Issue 1",
		Phase:    2,
	}

	// 항목 설정 및 선택
	selector.SetPhase2Items([]*domain.IssueRecord{testItem})
	selector.phase3List.selected[1] = true // ID 1 선택

	// 이벤트 수신 확인용 채널
	phaseChangeReceived := make(chan bool, 1)
	jobStartedReceived := make(chan bool, 1)
	var receivedPhaseEvent state.Event
	var receivedJobEvent state.Event

	eventBus.Subscribe(state.EventPhaseChange, func(event state.Event) {
		receivedPhaseEvent = event
		phaseChangeReceived <- true
	})

	eventBus.Subscribe(state.EventJobStarted, func(event state.Event) {
		receivedJobEvent = event
		jobStartedReceived <- true
	})

	// Act
	selector.onStartPhase3()

	// Assert - Phase Change Event
	select {
	case <-phaseChangeReceived:
		if receivedPhaseEvent.Type != state.EventPhaseChange {
			t.Errorf("Expected event type %v, got %v", state.EventPhaseChange, receivedPhaseEvent.Type)
		}

		if receivedPhaseEvent.Channel != 2 {
			t.Errorf("Expected channel 2, got %d", receivedPhaseEvent.Channel)
		}

		if receivedPhaseEvent.Data.(state.ProcessPhase) != state.PhaseAIExecution {
			t.Errorf("Expected PhaseAIExecution, got %v", receivedPhaseEvent.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected EventPhaseChange to be published")
	}

	// Assert - Job Started Event
	select {
	case <-jobStartedReceived:
		if receivedJobEvent.Type != state.EventJobStarted {
			t.Errorf("Expected event type %v, got %v", state.EventJobStarted, receivedJobEvent.Type)
		}

		if receivedJobEvent.Channel != 2 {
			t.Errorf("Expected channel 2, got %d", receivedJobEvent.Channel)
		}

		data := receivedJobEvent.Data.(map[string]interface{})
		if data["phase"] != "phase3" {
			t.Errorf("Expected phase3, got %v", data["phase"])
		}

		// 다중 선택 지원으로 issueRecords 배열 확인
		if issueRecords, ok := data["issueRecords"].([]*domain.IssueRecord); ok {
			if len(issueRecords) != 1 {
				t.Errorf("Expected 1 issue record, got %d", len(issueRecords))
			}
			if issueRecords[0].IssueKey != "ISSUE-1" {
				t.Errorf("Expected ISSUE-1, got %v", issueRecords[0].IssueKey)
			}
		} else {
			t.Error("Expected issueRecords to be []*domain.IssueRecord")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected EventJobStarted to be published")
	}

	// Assert - Button disabled
	if !selector.startPhase3.Disabled() {
		t.Error("Expected startPhase3 button to be disabled after execution")
	}

	// Assert - Status updated
	if selector.phase3Status.Text != "AI 플랜 실행 중..." {
		t.Errorf("Expected status 'AI 플랜 실행 중...', got '%s'", selector.phase3Status.Text)
	}
}

func TestAnalysisSelector_NoEventWhenNoSelection(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 0)

	// selectedPhase2Item이 nil인 상태

	eventReceived := make(chan bool, 1)
	eventBus.Subscribe(state.EventJobStarted, func(event state.Event) {
		eventReceived <- true
	})

	// Act
	selector.onStartPhase2()

	// Assert
	select {
	case <-eventReceived:
		t.Error("Expected no event to be published when no item is selected")
	case <-time.After(50 * time.Millisecond):
		// 정상: 이벤트가 발행되지 않아야 함
	}
}

func TestAnalysisSelector_PhaseChangeUpdatesUI(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 0)

	testItem := &domain.IssueRecord{
		ID:       1,
		IssueKey: "ISSUE-1",
		Summary:  "Test Issue 1",
		Phase:    1,
	}

	selector.selectedPhase2Item = testItem
	selector.startPhase2.Enable()

	// Act - Phase 변경 이벤트 발행
	eventBus.PublishSync(state.Event{
		Type:    state.EventPhaseChange,
		Channel: 0,
		Data:    state.PhaseAIPlanGeneration,
	})

	// Wait for event processing
	time.Sleep(50 * time.Millisecond)

	// Assert
	if selector.phase2Status.Text != "AI 플랜 생성 중..." {
		t.Errorf("Expected status 'AI 플랜 생성 중...', got '%s'", selector.phase2Status.Text)
	}

	if !selector.startPhase2.Disabled() {
		t.Error("Expected startPhase2 button to be disabled during generation")
	}

	// Act - Phase 완료 이벤트 발행
	eventBus.PublishSync(state.Event{
		Type:    state.EventPhaseChange,
		Channel: 0,
		Data:    state.PhaseAIPlanReady,
	})

	time.Sleep(50 * time.Millisecond)

	// Assert
	if selector.phase2Status.Text != "🟢 AI 플랜 준비 완료" {
		t.Errorf("Expected status 'AI 플랜 준비됨', got '%s'", selector.phase2Status.Text)
	}

	if selector.startPhase2.Disabled() {
		t.Error("Expected startPhase2 button to be enabled after plan ready")
	}
}

func TestAnalysisSelector_Phase1CompleteRefreshesLists(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 1)

	// Act - Phase1 완료 이벤트 발행
	eventBus.PublishSync(state.Event{
		Type:    state.EventPhase1Complete,
		Channel: 1,
		Data: map[string]interface{}{
			"issueKey": "ISSUE-1",
		},
	})

	time.Sleep(50 * time.Millisecond)

	// Assert
	if selector.phase2Status.Text != "새 항목이 반영되었습니다 (체크 후 AI 플랜 생성)" {
		t.Errorf("Expected refresh message, got '%s'", selector.phase2Status.Text)
	}
}

func TestAnalysisSelector_Phase2CompleteRefreshesPhase3List(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 2)

	testItem := &domain.IssueRecord{
		ID:       1,
		IssueKey: "ISSUE-1",
		Summary:  "Test Issue 1",
		Phase:    1,
	}

	selector.selectedPhase2Item = testItem

	// Act - Phase2 완료 이벤트 발행
	eventBus.PublishSync(state.Event{
		Type:    state.EventPhase2Complete,
		Channel: 2,
		Data:    testItem,
	})

	time.Sleep(50 * time.Millisecond)

	// Assert
	if selector.phase3Status.Text != "새 항목이 반영되었습니다 (체크 후 AI 실행)" {
		t.Errorf("Expected refresh message, got '%s'", selector.phase3Status.Text)
	}

	if selector.startPhase2.Disabled() {
		t.Error("Expected startPhase2 button to be re-enabled after phase2 completion")
	}
}

func TestAnalysisSelector_OnDeletePhase2Item_PublishesEvent(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 0)
	testItem := &domain.IssueRecord{
		ID:           11,
		IssueKey:     "ISSUE-DEL-1",
		Summary:      "Delete Test Item",
		Phase:        2,
		ChannelIndex: 0,
	}

	eventReceived := make(chan state.Event, 1)
	eventBus.Subscribe(state.EventIssueDeleteRequest, func(event state.Event) {
		eventReceived <- event
	})

	// Act
	selector.onDeletePhase2Item(testItem)

	// Assert
	select {
	case event := <-eventReceived:
		if event.Channel != 0 {
			t.Fatalf("expected channel 0, got %d", event.Channel)
		}
		payload, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected payload map, got %T", event.Data)
		}
		if payload["listPhase"] != 2 {
			t.Fatalf("expected listPhase=2, got %v", payload["listPhase"])
		}
		issue, ok := payload["issueRecord"].(*domain.IssueRecord)
		if !ok {
			t.Fatalf("expected issueRecord type *domain.IssueRecord, got %T", payload["issueRecord"])
		}
		if issue.ID != testItem.ID {
			t.Fatalf("expected issue id %d, got %d", testItem.ID, issue.ID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected delete request event to be published")
	}
}

func TestAnalysisSelector_OnDeletePhase3Item_PublishesEvent(t *testing.T) {
	// Arrange
	eventBus := state.NewEventBus()
	selector := NewAnalysisSelector(eventBus, 2)
	testItem := &domain.IssueRecord{
		ID:           22,
		IssueKey:     "ISSUE-DEL-2",
		Summary:      "Delete Test Item 2",
		Phase:        3,
		ChannelIndex: 2,
	}

	eventReceived := make(chan state.Event, 1)
	eventBus.Subscribe(state.EventIssueDeleteRequest, func(event state.Event) {
		eventReceived <- event
	})

	// Act
	selector.onDeletePhase3Item(testItem)

	// Assert
	select {
	case event := <-eventReceived:
		if event.Channel != 2 {
			t.Fatalf("expected channel 2, got %d", event.Channel)
		}
		payload, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected payload map, got %T", event.Data)
		}
		if payload["listPhase"] != 3 {
			t.Fatalf("expected listPhase=3, got %v", payload["listPhase"])
		}
		issue, ok := payload["issueRecord"].(*domain.IssueRecord)
		if !ok {
			t.Fatalf("expected issueRecord type *domain.IssueRecord, got %T", payload["issueRecord"])
		}
		if issue.IssueKey != testItem.IssueKey {
			t.Fatalf("expected issue key %s, got %s", testItem.IssueKey, issue.IssueKey)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected delete request event to be published")
	}
}
