package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/adapter"
	"jira-ai-generator/internal/config"
	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/usecase"
)

// App represents the main application
type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	config     *config.Config

	// Use cases
	processIssueUC *usecase.ProcessIssueUseCase
	docGenerator   *adapter.MarkdownGenerator
	claudeAdapter  *adapter.ClaudeCodeAdapter

	// UI components
	urlEntry         *widget.Entry
	projectPathEntry *widget.Entry
	resultText       *widget.Entry
	analysisText     *widget.Entry
	progressBar      *widget.ProgressBar
	statusLabel      *widget.Label
	processBtn       *widget.Button
	copyBtn          *widget.Button
	stopAnalysisBtn  *widget.Button
	copyAnalysisBtn  *widget.Button
	jobsList         *widget.List
	tabs             *container.AppTabs
	queueLists       [3]*widget.List
	historyList      *widget.List

	// Processing state
	currentDoc          *domain.GeneratedDocument
	currentMDPath       string
	currentAnalysisPath string
	currentPlanPath     string // Phase 1 결과 plan 파일 경로
	currentScriptPath   string
	runningJobs         []*AnalysisJob
	selectedJobIndex    int
	queues              [3]*AnalysisQueue
	completedJobs       []*AnalysisJob
	executePlanBtn      *widget.Button // "계획 실행" 버튼
}

// NewApp creates a new application instance with dependency injection
func NewApp(cfg *config.Config) *App {
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(NewKoreanTheme())

	// Create adapters
	jiraClient := adapter.NewJiraClient(cfg.Jira.URL, cfg.Jira.Email, cfg.Jira.APIKey)
	docGenerator := adapter.NewMarkdownGenerator(cfg.AI.PromptTemplate)
	claudeAdapter := adapter.NewClaudeCodeAdapter(cfg.Claude.CLIPath, cfg.Claude.WorkDir, cfg.Claude.Enabled)
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
