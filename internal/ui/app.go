package ui

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/adapter"
	"jira-ai-generator/internal/config"
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
}

// NewApp creates a new application instance with dependency injection
func NewApp(cfg *config.Config) *App {
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(NewKoreanTheme())

	// Create adapters
	jiraClient := adapter.NewJiraClient(cfg.Jira.URL, cfg.Jira.Email, cfg.Jira.APIKey)
	docGenerator := adapter.NewMarkdownGenerator(cfg.AI.PromptTemplate)
	claudeAdapter := adapter.NewClaudeCodeAdapter(cfg.Claude.CLIPath, cfg.Claude.Enabled)
	videoProcessor := adapter.NewFFmpegVideoProcessor()
	downloader := adapter.NewAttachmentDownloader(jiraClient, cfg.Output.Dir)

	// Create use cases
	processIssueUC := usecase.NewProcessIssueUseCase(jiraClient, downloader, videoProcessor, docGenerator, cfg.Output.Dir)

	return &App{
		fyneApp:        fyneApp,
		config:         cfg,
		processIssueUC: processIssueUC,
		docGenerator:   docGenerator,
		claudeAdapter:  claudeAdapter,
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
}

// Run starts the application
func (a *App) Run() {
	a.mainWindow = a.fyneApp.NewWindow("Jira AI Generator")
	a.mainWindow.Resize(fyne.NewSize(1500, 1000))
	a.mainWindow.CenterOnScreen()

	content := a.createMainContent()
	a.mainWindow.SetContent(content)

	// Load previous analysis results from output folder
	a.loadPreviousAnalysis()

	a.mainWindow.ShowAndRun()
}
