package ui

import (
	"fmt"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/adapter"
	"jira-ai-generator/internal/config"
	"jira-ai-generator/internal/port"
	"jira-ai-generator/internal/usecase"
)

// App represents the main application
type App struct {
	mu         sync.Mutex // 동시 접근 보호 (completedJobs 전용)
	fyneApp    fyne.App
	mainWindow fyne.Window
	config     *config.Config

	// Use cases
	processIssueUC *usecase.ProcessIssueUseCase
	docGenerator   *adapter.MarkdownGenerator
	claudeAdapter  *adapter.ClaudeCodeAdapter

	// Database stores
	issueStore      port.IssueStore
	analysisStore   port.AnalysisResultStore
	attachmentStore port.AttachmentStore
	repository      *adapter.SQLiteRepository // For Close()

	// UI components (글로벌)
	statusLabel *widget.Label
	stopAllBtn  *widget.Button
	tabs        *container.AppTabs // 채널 탭 (채널1/채널2/채널3)
	historyList *widget.List

	// 채널별 독립 UI 및 상태
	channels [3]*ChannelState

	// Processing state
	queues        [3]*AnalysisQueue
	completedJobs []*AnalysisJob

	// V2 실행 작업 추적
	runningTasksMu sync.Mutex
	runningTasks   [3]map[string]*RunningTask

	// 채널별 이슈 목록 로딩 요청 추적 (최신 요청만 UI 반영)
	issueListLoadMu  sync.Mutex
	issueListLoadSeq [3]uint64

	// UI version control
	useV2UI bool // V2 UI 사용 여부 (기본값: true, V1은 deprecated)

	// V2 상태 참조
	v2State *AppV2State
}

// RunningTask는 V2에서 실행 중인 2차 작업의 런타임 상태를 보관한다.
type RunningTask struct {
	TaskID          string
	IssueID         int64
	IssueKey        string
	ChannelIndex    int
	PhaseLabel      string
	PID             int
	ScriptPath      string
	LogPath         string
	CancelRequested bool
}

// NewApp creates a new application instance with dependency injection
func NewApp(cfg *config.Config) (*App, error) {
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(NewKoreanTheme())

	// Initialize SQLite repository
	dbPath := filepath.Join(cfg.Output.Dir, "jira.db")
	repo, err := adapter.NewSQLiteRepository(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init DB: %w", err)
	}

	// Create adapters
	jiraClient := adapter.NewJiraClient(cfg.Jira.URL, cfg.Jira.Email, cfg.Jira.APIKey)
	docGenerator := adapter.NewMarkdownGenerator(cfg.AI.PromptTemplate)
	claudeAdapter := adapter.NewClaudeCodeAdapter(cfg.Claude.CLIPath, cfg.Claude.Enabled, cfg.Claude.Model, cfg.Claude.HookScriptPath)
	videoProcessor := adapter.NewFFmpegVideoProcessor()
	downloader := adapter.NewAttachmentDownloader(jiraClient, cfg.Output.Dir)

	// Create use cases
	processIssueUC := usecase.NewProcessIssueUseCase(jiraClient, downloader, videoProcessor, docGenerator, cfg.Output.Dir)

	appInstance := &App{
		fyneApp:         fyneApp,
		config:          cfg,
		processIssueUC:  processIssueUC,
		docGenerator:    docGenerator,
		claudeAdapter:   claudeAdapter,
		issueStore:      repo,
		analysisStore:   repo,
		attachmentStore: repo,
		repository:      repo,
		useV2UI:         true, // V2 UI를 기본으로 활성화
		channels: [3]*ChannelState{
			{Index: 0, Name: "채널 1"},
			{Index: 1, Name: "채널 2"},
			{Index: 2, Name: "채널 3"},
		},
		queues: [3]*AnalysisQueue{
			{Name: "채널 1", Pending: []*AnalysisJob{}},
			{Name: "채널 2", Pending: []*AnalysisJob{}},
			{Name: "채널 3", Pending: []*AnalysisJob{}},
		},
	}
	for i := 0; i < 3; i++ {
		appInstance.runningTasks[i] = make(map[string]*RunningTask)
	}

	return appInstance, nil
}

// UseV2UI returns whether V2 UI is enabled
func (a *App) UseV2UI() bool {
	return a.useV2UI
}

// Run starts the application
func (a *App) Run() {
	// V2 UI가 기본값이므로 항상 V2로 실행
	a.RunV2()
}

// Close closes the database connection
func (a *App) Close() error {
	if a.repository != nil {
		return a.repository.Close()
	}
	return nil
}
