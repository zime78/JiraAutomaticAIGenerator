package state

import (
	"sync"
	"time"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/port"
)

// AppState 전체 애플리케이션 상태 관리
type AppState struct {
	mu sync.RWMutex

	// 채널 상태 (단일 채널)
	Channel *ChannelStateData

	// 글로벌 상태
	GlobalStatus string
	IsProcessing bool

	// 이벤트 버스
	EventBus *EventBus

	// 완료된 작업 이력 (메모리 캐시)
	CompletedJobs []*JobData

	// Database stores (영속성 레이어)
	IssueStore    port.IssueStore
	AnalysisStore port.AnalysisResultStore
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
	ScriptPath   string
	LogPath      string
	Phase        ProcessPhase
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
func NewAppState(issueStore port.IssueStore, analysisStore port.AnalysisResultStore) *AppState {
	state := &AppState{
		Channel:       NewChannelStateData(0, "기본"),
		GlobalStatus:  "준비됨",
		EventBus:      NewEventBus(),
		CompletedJobs: make([]*JobData, 0),
		IssueStore:    issueStore,
		AnalysisStore: analysisStore,
	}

	// DB에서 기존 데이터 로드
	if err := state.LoadFromDB(); err != nil {
		// 로드 실패해도 앱은 시작 (로그만 남김)
		state.AddLog(LogError, "DB 로드 실패: "+err.Error(), "AppState")
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

// GetChannel 채널 상태 조회 (하위 호환: index 무시)
func (s *AppState) GetChannel(index int) *ChannelStateData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Channel
}

// GetActiveChannel 활성 채널 상태 조회
func (s *AppState) GetActiveChannel() *ChannelStateData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Channel
}

// UpdatePhase 처리 단계 업데이트
func (s *AppState) UpdatePhase(channelIndex int, phase ProcessPhase) {
	s.mu.Lock()
	ch := s.Channel
	ch.Phase = phase
	ch.Progress = phase.Progress()

	// 단계별 상태 업데이트
	stepIndex := int(phase) - 1
	if stepIndex >= 0 && stepIndex < len(ch.Steps) {
		for i := 0; i < stepIndex; i++ {
			ch.Steps[i].Status = StepCompleted
			ch.Steps[i].Progress = 1.0
		}
		ch.Steps[stepIndex].Status = StepRunning
	}
	s.mu.Unlock()

	s.EventBus.PublishPhaseChange(0, phase)
}

// UpdateProgress 진행률 업데이트
func (s *AppState) UpdateProgress(channelIndex int, step, total int, message string) {
	s.mu.Lock()
	ch := s.Channel
	progress := float64(step) / float64(total)
	ch.Progress = progress

	for _, st := range ch.Steps {
		if st.Status == StepRunning {
			st.Progress = progress
			st.Message = message
			break
		}
	}
	phase := ch.Phase
	s.mu.Unlock()

	s.EventBus.PublishProgress(0, phase, step, total, message)
}

// AddLog 로그 추가
func (s *AppState) AddLog(level LogLevel, message, source string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
	}

	s.mu.Lock()
	ch := s.Channel
	ch.Logs = append(ch.Logs, entry)
	if len(ch.Logs) > 1000 {
		ch.Logs = ch.Logs[len(ch.Logs)-1000:]
	}
	s.mu.Unlock()

	s.EventBus.PublishLog(0, level, message, source)
}

// AddJob 대기열에 작업 추가
func (s *AppState) AddJob(job *JobData) {
	s.mu.Lock()
	s.Channel.Queue = append(s.Channel.Queue, job)
	s.mu.Unlock()

	s.EventBus.Publish(Event{
		Type:    EventQueueUpdated,
		Channel: 0,
		Data:    job,
	})
}

// CompleteJob 작업 완료 처리
func (s *AppState) CompleteJob(jobID string, result interface{}) {
	s.mu.Lock()
	ch := s.Channel
	if ch.CurrentJob != nil && ch.CurrentJob.ID == jobID {
		ch.CurrentJob.Status = JobCompleted
		ch.CurrentJob.EndTime = time.Now()
		s.CompletedJobs = append([]*JobData{ch.CurrentJob}, s.CompletedJobs...)
		ch.CurrentJob = nil
		ch.IsRunning = false
	}
	s.mu.Unlock()

	s.EventBus.PublishJobCompleted(0, jobID, result)
	s.EventBus.Publish(Event{
		Type:    EventHistoryAdded,
		Channel: 0,
		Data:    jobID,
	})
}

// FailJob 작업 실패 처리
func (s *AppState) FailJob(jobID string, err error) {
	s.mu.Lock()
	ch := s.Channel
	if ch.CurrentJob != nil && ch.CurrentJob.ID == jobID {
		ch.CurrentJob.Status = JobFailed
		ch.CurrentJob.EndTime = time.Now()
		ch.CurrentJob.Error = err.Error()
		s.CompletedJobs = append([]*JobData{ch.CurrentJob}, s.CompletedJobs...)
		ch.CurrentJob = nil
		ch.IsRunning = false
	}

	ch.Phase = PhaseFailed
	for _, st := range ch.Steps {
		if st.Status == StepRunning {
			st.Status = StepFailed
			break
		}
	}
	s.mu.Unlock()

	s.EventBus.PublishJobFailed(0, jobID, err)
}

// ResetChannel 채널 상태 초기화
func (s *AppState) ResetChannel() {
	s.mu.Lock()
	ch := s.Channel
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

// LoadFromDB DB에서 기존 데이터 로드
func (s *AppState) LoadFromDB() error {
	if s.IssueStore == nil {
		return nil // DB가 설정되지 않은 경우 스킵
	}

	// 모든 이슈 로드
	issues, err := s.IssueStore.ListAllIssues()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 완료된 작업 목록 재구성
	s.CompletedJobs = make([]*JobData, 0, len(issues))

	for _, issue := range issues {
		job := &JobData{
			ID:        issue.IssueKey,
			IssueKey:  issue.IssueKey,
			MDPath:    issue.MDPath,
			StartTime: issue.CreatedAt,
			EndTime:   issue.UpdatedAt,
			Status:    JobCompleted,
		}

		// Phase에 따라 상태 설정
		switch issue.Phase {
		case 1:
			job.Phase = PhasePhase1Complete
		case 2, 3:
			job.Phase = PhaseCompleted
		default:
			job.Phase = PhaseIdle
		}

		// 분석 결과 로드 (있는 경우)
		if s.AnalysisStore != nil {
			results, err := s.AnalysisStore.ListAnalysisResultsByIssue(issue.ID)
			if err == nil && len(results) > 0 {
				for _, result := range results {
					if result.PlanPath != "" {
						job.PlanPath = result.PlanPath
					}
					if result.ResultPath != "" {
						job.AnalysisPath = result.ResultPath
					}
				}
			}
		}

		s.CompletedJobs = append(s.CompletedJobs, job)
	}

	return nil
}

// SaveIssueToDBAfterPhase1 1차 분석 완료 후 DB에 저장한다.
func (s *AppState) SaveIssueToDBAfterPhase1(issueKey, summary, description, jiraURL, mdPath string) (*domain.IssueRecord, error) {
	if s.IssueStore == nil {
		return nil, nil
	}

	issue := &domain.IssueRecord{
		IssueKey:     issueKey,
		Summary:      summary,
		Description:  description,
		JiraURL:      jiraURL,
		MDPath:       mdPath,
		Phase:        1,
		Status:       "active",
		ChannelIndex: 0,
	}

	if err := s.IssueStore.UpsertIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

// UpdateIssuePhase 이슈 단계 업데이트
func (s *AppState) UpdateIssuePhase(issueKey string, phase int) error {
	if s.IssueStore == nil {
		return nil
	}

	issue, err := s.IssueStore.GetIssue(issueKey)
	if err != nil {
		return err
	}

	issue.Phase = phase
	return s.IssueStore.UpdateIssue(issue)
}

// SaveAnalysisResult 분석 결과 저장
func (s *AppState) SaveAnalysisResult(issueKey string, analysisPhase int, planPath, resultPath string, status string) error {
	if s.IssueStore == nil || s.AnalysisStore == nil {
		return nil
	}

	// 이슈 ID 조회
	issue, err := s.IssueStore.GetIssue(issueKey)
	if err != nil {
		return err
	}

	now := time.Now()
	result := &domain.AnalysisResult{
		IssueID:       issue.ID,
		AnalysisPhase: analysisPhase,
		PlanPath:      planPath,
		ResultPath: resultPath,
		Status:        status,
		CompletedAt:   &now,
	}

	return s.AnalysisStore.CreateAnalysisResult(result)
}
