package state

import (
	"sync"
	"time"

	"jira-ai-generator/internal/domain"
)

// AppState 전체 애플리케이션 상태 관리
type AppState struct {
	mu sync.RWMutex

	// 채널 상태
	Channels      [3]*ChannelStateData
	ActiveChannel int

	// 글로벌 상태
	GlobalStatus string
	IsProcessing bool

	// 이벤트 버스
	EventBus *EventBus

	// 완료된 작업 이력
	CompletedJobs []*JobData
}

// ChannelStateData 채널별 상태 데이터 (UI 위젯 참조 제외)
type ChannelStateData struct {
	Index int
	Name  string

	// 입력 상태
	URL         string
	ProjectPath string

	// 진행 상태
	Phase    ProcessPhase
	Progress float64
	Steps    []*StepState
	Logs     []LogEntry

	// 결과 상태
	IssueInfo string
	Analysis  string

	// 현재 문서 상태
	CurrentDoc          *domain.GeneratedDocument
	CurrentMDPath       string
	CurrentAnalysisPath string
	CurrentPlanPath     string
	CurrentScriptPath   string

	// 대기열
	Queue      []*JobData
	CurrentJob *JobData
	IsRunning  bool
}

// StepState 단계별 상태
type StepState struct {
	Name     string
	Status   StepStatus
	Progress float64
	Message  string
}

// StepStatus 단계 상태
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepCompleted
	StepFailed
)

// String StepStatus의 문자열 표현
func (s StepStatus) String() string {
	switch s {
	case StepPending:
		return "대기"
	case StepRunning:
		return "진행 중"
	case StepCompleted:
		return "완료"
	case StepFailed:
		return "실패"
	default:
		return "알 수 없음"
	}
}

// JobData 작업 데이터
type JobData struct {
	ID           string
	IssueKey     string
	MDPath       string
	PlanPath     string
	AnalysisPath string
	ExecutionPath string
	ScriptPath   string
	LogPath      string
	Phase        ProcessPhase
	ChannelIndex int
	StartTime    time.Time
	EndTime      time.Time
	Duration     string
	Status       JobStatus
	Error        string
}

// JobStatus 작업 상태
type JobStatus int

const (
	JobPending JobStatus = iota
	JobRunning
	JobCompleted
	JobFailed
	JobCancelled
)

// String JobStatus의 문자열 표현
func (s JobStatus) String() string {
	switch s {
	case JobPending:
		return "대기"
	case JobRunning:
		return "실행 중"
	case JobCompleted:
		return "완료"
	case JobFailed:
		return "실패"
	case JobCancelled:
		return "취소됨"
	default:
		return "알 수 없음"
	}
}

// LogEntry 로그 항목
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Source    string
}

// NewAppState 새 AppState 생성
func NewAppState() *AppState {
	state := &AppState{
		Channels: [3]*ChannelStateData{
			NewChannelStateData(0, "채널 1"),
			NewChannelStateData(1, "채널 2"),
			NewChannelStateData(2, "채널 3"),
		},
		ActiveChannel: 0,
		GlobalStatus:  "준비됨",
		EventBus:      NewEventBus(),
		CompletedJobs: make([]*JobData, 0),
	}

	return state
}

// NewChannelStateData 새 채널 상태 생성
func NewChannelStateData(index int, name string) *ChannelStateData {
	return &ChannelStateData{
		Index:    index,
		Name:     name,
		Phase:    PhaseIdle,
		Progress: 0,
		Steps:    createDefaultSteps(),
		Logs:     make([]LogEntry, 0),
		Queue:    make([]*JobData, 0),
	}
}

// createDefaultSteps 기본 단계 목록 생성
func createDefaultSteps() []*StepState {
	return []*StepState{
		{Name: "이슈 조회", Status: StepPending, Progress: 0, Message: ""},
		{Name: "첨부파일 다운로드", Status: StepPending, Progress: 0, Message: ""},
		{Name: "프레임 추출", Status: StepPending, Progress: 0, Message: ""},
		{Name: "문서 생성", Status: StepPending, Progress: 0, Message: ""},
		{Name: "AI 분석", Status: StepPending, Progress: 0, Message: ""},
	}
}

// GetChannel 특정 채널 상태 조회
func (s *AppState) GetChannel(index int) *ChannelStateData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= 3 {
		return nil
	}
	return s.Channels[index]
}

// GetActiveChannel 활성 채널 상태 조회
func (s *AppState) GetActiveChannel() *ChannelStateData {
	return s.GetChannel(s.ActiveChannel)
}

// SetActiveChannel 활성 채널 변경
func (s *AppState) SetActiveChannel(index int) {
	s.mu.Lock()
	s.ActiveChannel = index
	s.mu.Unlock()

	s.EventBus.Publish(Event{
		Type:    EventChannelSwitch,
		Channel: index,
		Data:    index,
	})
}

// UpdatePhase 채널의 처리 단계 업데이트
func (s *AppState) UpdatePhase(channelIndex int, phase ProcessPhase) {
	s.mu.Lock()
	if channelIndex >= 0 && channelIndex < 3 {
		ch := s.Channels[channelIndex]
		ch.Phase = phase
		ch.Progress = phase.Progress()

		// 단계별 상태 업데이트
		stepIndex := int(phase) - 1
		if stepIndex >= 0 && stepIndex < len(ch.Steps) {
			// 이전 단계들은 완료로 표시
			for i := 0; i < stepIndex; i++ {
				ch.Steps[i].Status = StepCompleted
				ch.Steps[i].Progress = 1.0
			}
			// 현재 단계는 진행 중으로 표시
			ch.Steps[stepIndex].Status = StepRunning
		}
	}
	s.mu.Unlock()

	s.EventBus.PublishPhaseChange(channelIndex, phase)
}

// UpdateProgress 채널의 진행률 업데이트
func (s *AppState) UpdateProgress(channelIndex int, step, total int, message string) {
	s.mu.Lock()
	if channelIndex >= 0 && channelIndex < 3 {
		ch := s.Channels[channelIndex]
		progress := float64(step) / float64(total)
		ch.Progress = progress

		// 현재 진행 중인 단계의 진행률 업데이트
		for _, st := range ch.Steps {
			if st.Status == StepRunning {
				st.Progress = progress
				st.Message = message
				break
			}
		}
	}
	s.mu.Unlock()

	phase := PhaseIdle
	s.mu.RLock()
	if channelIndex >= 0 && channelIndex < 3 {
		phase = s.Channels[channelIndex].Phase
	}
	s.mu.RUnlock()

	s.EventBus.PublishProgress(channelIndex, phase, step, total, message)
}

// AddLog 채널에 로그 추가
func (s *AppState) AddLog(channelIndex int, level LogLevel, message, source string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
	}

	s.mu.Lock()
	if channelIndex >= 0 && channelIndex < 3 {
		ch := s.Channels[channelIndex]
		ch.Logs = append(ch.Logs, entry)

		// 로그 개수 제한 (최대 1000개)
		if len(ch.Logs) > 1000 {
			ch.Logs = ch.Logs[len(ch.Logs)-1000:]
		}
	}
	s.mu.Unlock()

	s.EventBus.PublishLog(channelIndex, level, message, source)
}

// AddJob 채널 대기열에 작업 추가
func (s *AppState) AddJob(channelIndex int, job *JobData) {
	s.mu.Lock()
	if channelIndex >= 0 && channelIndex < 3 {
		ch := s.Channels[channelIndex]
		ch.Queue = append(ch.Queue, job)
	}
	s.mu.Unlock()

	s.EventBus.Publish(Event{
		Type:    EventQueueUpdated,
		Channel: channelIndex,
		Data:    job,
	})
}

// CompleteJob 작업 완료 처리
func (s *AppState) CompleteJob(channelIndex int, jobID string, result interface{}) {
	s.mu.Lock()
	if channelIndex >= 0 && channelIndex < 3 {
		ch := s.Channels[channelIndex]

		// 현재 작업 완료 처리
		if ch.CurrentJob != nil && ch.CurrentJob.ID == jobID {
			ch.CurrentJob.Status = JobCompleted
			ch.CurrentJob.EndTime = time.Now()

			// 완료 목록에 추가
			s.CompletedJobs = append([]*JobData{ch.CurrentJob}, s.CompletedJobs...)

			ch.CurrentJob = nil
			ch.IsRunning = false
		}
	}
	s.mu.Unlock()

	s.EventBus.PublishJobCompleted(channelIndex, jobID, result)

	s.EventBus.Publish(Event{
		Type:    EventHistoryAdded,
		Channel: channelIndex,
		Data:    jobID,
	})
}

// FailJob 작업 실패 처리
func (s *AppState) FailJob(channelIndex int, jobID string, err error) {
	s.mu.Lock()
	if channelIndex >= 0 && channelIndex < 3 {
		ch := s.Channels[channelIndex]

		if ch.CurrentJob != nil && ch.CurrentJob.ID == jobID {
			ch.CurrentJob.Status = JobFailed
			ch.CurrentJob.EndTime = time.Now()
			ch.CurrentJob.Error = err.Error()

			// 완료 목록에 추가 (실패도 이력에 남김)
			s.CompletedJobs = append([]*JobData{ch.CurrentJob}, s.CompletedJobs...)

			ch.CurrentJob = nil
			ch.IsRunning = false
		}

		// 단계 상태를 실패로 변경
		ch.Phase = PhaseFailed
		for _, st := range ch.Steps {
			if st.Status == StepRunning {
				st.Status = StepFailed
				break
			}
		}
	}
	s.mu.Unlock()

	s.EventBus.PublishJobFailed(channelIndex, jobID, err)
}

// ResetChannel 채널 상태 초기화
func (s *AppState) ResetChannel(channelIndex int) {
	s.mu.Lock()
	if channelIndex >= 0 && channelIndex < 3 {
		ch := s.Channels[channelIndex]
		ch.Phase = PhaseIdle
		ch.Progress = 0
		ch.Steps = createDefaultSteps()
		ch.Logs = make([]LogEntry, 0)
		ch.IssueInfo = ""
		ch.Analysis = ""
		ch.CurrentDoc = nil
		ch.CurrentMDPath = ""
		ch.CurrentAnalysisPath = ""
		ch.CurrentPlanPath = ""
		ch.CurrentScriptPath = ""
	}
	s.mu.Unlock()
}

// SetGlobalStatus 글로벌 상태 메시지 설정
func (s *AppState) SetGlobalStatus(status string) {
	s.mu.Lock()
	s.GlobalStatus = status
	s.mu.Unlock()
}

// GetCompletedJobs 완료된 작업 목록 조회
func (s *AppState) GetCompletedJobs() []*JobData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 복사본 반환
	result := make([]*JobData, len(s.CompletedJobs))
	copy(result, s.CompletedJobs)
	return result
}
