package state

import (
	"sync"
	"time"

	"jira-ai-generator/internal/logger"
)

// EventType 이벤트 유형 정의
type EventType string

const (
	// 진행 상황 관련 이벤트
	EventProgressUpdate EventType = "progress.update"
	EventPhaseChange    EventType = "phase.change"
	EventLogAdded       EventType = "log.added"

	// 작업 관련 이벤트
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
	EventJobFailed    EventType = "job.failed"

	// UI 관련 이벤트
	EventChannelSwitch EventType = "channel.switch"
	EventQueueUpdated  EventType = "queue.updated"
	EventHistoryAdded  EventType = "history.added"

	// 신규 이벤트 타입
	EventSidebarAction    EventType = "sidebar.action"       // Sidebar에서 액션 발생
	EventPhase1Complete   EventType = "phase1.complete"      // 1차 분석 완료
	EventPhase2Complete   EventType = "phase2.complete"      // 2차 분석 완료 (Plan 준비됨)
	EventPhase3Complete   EventType = "phase3.complete"      // 3차 분석 완료 (실행 완료)
	EventDBSync           EventType = "db.sync"              // DB 동기화 완료
	EventIssueListRefresh EventType = "issue.list.refresh"   // 이슈 목록 갱신 필요
)

// ProcessPhase 처리 단계 정의
type ProcessPhase int

const (
	PhaseIdle ProcessPhase = iota
	PhaseFetchingIssue
	PhaseDownloadingAttachments
	PhaseExtractingFrames
	PhaseGeneratingDocument
	PhasePhase1Complete   // 1차 완료 (MD 생성됨)
	PhaseAIPlanGeneration // 2차: AI Plan 생성 중
	PhaseAIPlanReady      // 2차 완료: Plan 준비됨
	PhaseAIExecution      // 3차: Plan 실행 중
	PhaseAnalyzing        // 기존 유지 (하위 호환)
	PhaseCompleted
	PhaseFailed
)

// String ProcessPhase의 문자열 표현
func (p ProcessPhase) String() string {
	switch p {
	case PhaseIdle:
		return "대기"
	case PhaseFetchingIssue:
		return "이슈 조회"
	case PhaseDownloadingAttachments:
		return "첨부파일 다운로드"
	case PhaseExtractingFrames:
		return "프레임 추출"
	case PhaseGeneratingDocument:
		return "문서 생성"
	case PhasePhase1Complete:
		return "1차 분석 완료"
	case PhaseAIPlanGeneration:
		return "AI 플랜 생성 중"
	case PhaseAIPlanReady:
		return "AI 플랜 준비됨"
	case PhaseAIExecution:
		return "AI 플랜 실행 중"
	case PhaseAnalyzing:
		return "AI 분석"
	case PhaseCompleted:
		return "완료"
	case PhaseFailed:
		return "실패"
	default:
		return "알 수 없음"
	}
}

// Progress ProcessPhase의 기본 진행률 (0.0 ~ 1.0)
func (p ProcessPhase) Progress() float64 {
	switch p {
	case PhaseIdle:
		return 0.0
	case PhaseFetchingIssue:
		return 0.1
	case PhaseDownloadingAttachments:
		return 0.3
	case PhaseExtractingFrames:
		return 0.5
	case PhaseGeneratingDocument:
		return 0.6
	case PhasePhase1Complete:
		return 0.65
	case PhaseAIPlanGeneration:
		return 0.7
	case PhaseAIPlanReady:
		return 0.75
	case PhaseAIExecution:
		return 0.85
	case PhaseAnalyzing:
		return 0.8
	case PhaseCompleted:
		return 1.0
	case PhaseFailed:
		return 0.0
	default:
		return 0.0
	}
}

// Event 이벤트 구조체
type Event struct {
	Type      EventType
	Channel   int
	Data      interface{}
	Timestamp time.Time
}

// ProgressData 진행률 데이터
type ProgressData struct {
	Phase    ProcessPhase
	Step     int
	Total    int
	Progress float64
	Message  string
}

// LogData 로그 데이터
type LogData struct {
	Level   LogLevel
	Message string
	Source  string
}

// LogLevel 로그 레벨
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarning
	LogError
)

// String LogLevel의 문자열 표현
func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarning:
		return "WARNING"
	case LogError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// EventHandler 이벤트 핸들러 함수 타입
type EventHandler func(event Event)

// EventBus 이벤트 버스
type EventBus struct {
	subscribers map[EventType][]EventHandler
	mu          sync.RWMutex
}

// NewEventBus 새 EventBus 생성
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]EventHandler),
	}
}

// Subscribe 이벤트 구독
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// SubscribeMultiple 여러 이벤트 타입 구독
func (eb *EventBus) SubscribeMultiple(eventTypes []EventType, handler EventHandler) {
	for _, eventType := range eventTypes {
		eb.Subscribe(eventType, handler)
	}
}

// Publish 이벤트 발행
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	// 타임스탬프 설정
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	logger.Debug("Publish: type=%s, channel=%d, handler_count=%d", event.Type, event.Channel, len(handlers))

	// 모든 핸들러에게 이벤트 전달 (비동기)
	for _, handler := range handlers {
		go handler(event)
	}
}

// PublishSync 동기적으로 이벤트 발행 (UI 업데이트용)
func (eb *EventBus) PublishSync(event Event) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	logger.Debug("PublishSync: type=%s, channel=%d, handler_count=%d", event.Type, event.Channel, len(handlers))

	for _, handler := range handlers {
		handler(event)
	}
}

// PublishProgress 진행률 이벤트 발행 헬퍼
func (eb *EventBus) PublishProgress(channel int, phase ProcessPhase, step, total int, message string) {
	progress := float64(step) / float64(total)
	if total == 0 {
		progress = phase.Progress()
	}

	eb.Publish(Event{
		Type:    EventProgressUpdate,
		Channel: channel,
		Data: ProgressData{
			Phase:    phase,
			Step:     step,
			Total:    total,
			Progress: progress,
			Message:  message,
		},
	})
}

// PublishPhaseChange 단계 변경 이벤트 발행 헬퍼
func (eb *EventBus) PublishPhaseChange(channel int, phase ProcessPhase) {
	eb.Publish(Event{
		Type:    EventPhaseChange,
		Channel: channel,
		Data:    phase,
	})
}

// PublishLog 로그 이벤트 발행 헬퍼
func (eb *EventBus) PublishLog(channel int, level LogLevel, message, source string) {
	eb.Publish(Event{
		Type:    EventLogAdded,
		Channel: channel,
		Data: LogData{
			Level:   level,
			Message: message,
			Source:  source,
		},
	})
}

// PublishJobCompleted 작업 완료 이벤트 발행 헬퍼
func (eb *EventBus) PublishJobCompleted(channel int, jobID string, result interface{}) {
	eb.Publish(Event{
		Type:    EventJobCompleted,
		Channel: channel,
		Data: map[string]interface{}{
			"jobID":  jobID,
			"result": result,
		},
	})
}

// PublishJobFailed 작업 실패 이벤트 발행 헬퍼
func (eb *EventBus) PublishJobFailed(channel int, jobID string, err error) {
	eb.Publish(Event{
		Type:    EventJobFailed,
		Channel: channel,
		Data: map[string]interface{}{
			"jobID": jobID,
			"error": err.Error(),
		},
	})
}

// Clear 모든 구독 해제
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers = make(map[EventType][]EventHandler)
}

// Unsubscribe 특정 이벤트 타입의 모든 핸들러 제거
func (eb *EventBus) Unsubscribe(eventType EventType) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.subscribers, eventType)
}
