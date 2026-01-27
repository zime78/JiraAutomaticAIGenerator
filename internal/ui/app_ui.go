package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (a *App) createMainContent() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Jira AI Generator",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	a.urlEntry = widget.NewEntry()
	a.urlEntry.SetPlaceHolder("Jira URL을 입력하세요 (예: https://domain.atlassian.net/browse/PROJ-123)")

	// Project path input
	a.projectPathEntry = widget.NewEntry()
	a.projectPathEntry.SetPlaceHolder("프로젝트 경로를 입력하세요 (예: /Users/user/MyProject)")
	if a.config.Claude.ProjectPath != "" {
		a.projectPathEntry.SetText(a.config.Claude.ProjectPath)
	}

	browseBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				a.projectPathEntry.SetText(uri.Path())
			}
		}, a.mainWindow)
	})

	projectPathRow := container.NewBorder(nil, nil, nil, browseBtn, a.projectPathEntry)

	a.processBtn = widget.NewButtonWithIcon("분석 시작", theme.MediaPlayIcon(), a.onProcess)
	a.processBtn.Importance = widget.HighImportance

	a.copyBtn = widget.NewButtonWithIcon("결과 복사", theme.ContentCopyIcon(), a.onCopy)
	a.copyBtn.Disable()

	a.stopAnalysisBtn = widget.NewButtonWithIcon("전체 중지", theme.MediaStopIcon(), a.onStopAllQueues)
	a.stopAnalysisBtn.Importance = widget.DangerImportance
	if !a.claudeAdapter.IsEnabled() {
		a.stopAnalysisBtn.Hide()
	}

	buttonRow := container.NewHBox(
		a.processBtn,
		a.copyBtn,
		a.stopAnalysisBtn,
	)

	a.progressBar = widget.NewProgressBar()
	a.progressBar.Hide()

	a.statusLabel = widget.NewLabel("대기 중...")

	// Issue info tab
	a.resultText = widget.NewMultiLineEntry()
	a.resultText.SetPlaceHolder("분석 결과가 여기에 표시됩니다...")
	a.resultText.Wrapping = fyne.TextWrapWord
	issueScroll := container.NewScroll(a.resultText)
	issueScroll.SetMinSize(fyne.NewSize(760, 400))

	// AI Analysis tab
	a.analysisText = widget.NewMultiLineEntry()
	a.analysisText.SetPlaceHolder("AI 분석 결과가 여기에 표시됩니다...")
	a.analysisText.Wrapping = fyne.TextWrapWord

	a.copyAnalysisBtn = widget.NewButtonWithIcon("분석 결과 복사", theme.ContentCopyIcon(), a.onCopyAnalysis)
	a.copyAnalysisBtn.Disable()

	refreshBtn := widget.NewButtonWithIcon("새로고침", theme.ViewRefreshIcon(), a.onRefreshAnalysis)

	// Create 3 channel queue panels
	channelPanels := a.createChannelPanels()

	// 3 channels side by side
	channelsRow := container.NewGridWithColumns(3,
		channelPanels[0],
		channelPanels[1],
		channelPanels[2],
	)

	// Completion history list
	historyContainer := a.createHistoryPanel()

	analysisScroll := container.NewScroll(a.analysisText)
	analysisScroll.SetMinSize(fyne.NewSize(760, 180))

	analysisTab := container.NewBorder(
		container.NewVBox(
			container.NewHBox(a.copyAnalysisBtn, refreshBtn),
			channelsRow,
			historyContainer,
		),
		nil, nil, nil,
		analysisScroll,
	)

	// Create tabs
	a.tabs = container.NewAppTabs(
		container.NewTabItem("📋 이슈 정보", issueScroll),
		container.NewTabItem("🤖 AI 분석 결과", analysisTab),
	)
	a.tabs.SetTabLocation(container.TabLocationTop)

	inputSection := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Jira URL:"),
		a.urlEntry,
		widget.NewLabel("프로젝트 경로 (AI 분석용):"),
		projectPathRow,
		buttonRow,
		a.progressBar,
		a.statusLabel,
	)

	mainLayout := container.NewBorder(
		inputSection,
		nil,
		nil,
		nil,
		a.tabs,
	)

	return container.NewPadded(mainLayout)
}

func (a *App) createChannelPanels() [3]fyne.CanvasObject {
	var channelPanels [3]fyne.CanvasObject

	for i := 0; i < 3; i++ {
		channelIndex := i
		queue := a.queues[i]

		// Queue list for this channel
		a.queueLists[i] = widget.NewList(
			func() int {
				count := len(a.queues[channelIndex].Pending)
				if a.queues[channelIndex].Current != nil {
					count++
				}
				return count
			},
			func() fyne.CanvasObject {
				return container.NewHBox(
					widget.NewLabel("▶"),
					widget.NewLabel("ITSM-0000"),
				)
			},
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				q := a.queues[channelIndex]
				box := obj.(*fyne.Container)
				if id == 0 && q.Current != nil {
					box.Objects[0].(*widget.Label).SetText("▶")
					box.Objects[1].(*widget.Label).SetText(q.Current.IssueKey)
				} else {
					pendingIdx := id
					if q.Current != nil {
						pendingIdx = id - 1
					}
					if pendingIdx < len(q.Pending) {
						box.Objects[0].(*widget.Label).SetText("  ")
						box.Objects[1].(*widget.Label).SetText(q.Pending[pendingIdx].IssueKey)
					}
				}
			},
		)

		addBtn := widget.NewButtonWithIcon("추가", theme.ContentAddIcon(), func() {
			a.addToQueue(channelIndex)
		})

		stopBtn := widget.NewButtonWithIcon("중지", theme.MediaStopIcon(), func() {
			a.stopQueueCurrent(channelIndex)
		})
		stopBtn.Importance = widget.DangerImportance

		channelLabel := widget.NewLabel(fmt.Sprintf("📊 %s", queue.Name))
		channelLabel.TextStyle = fyne.TextStyle{Bold: true}

		buttonRow := container.NewHBox(addBtn, stopBtn)
		queueScroll := container.NewScroll(a.queueLists[i])
		queueScroll.SetMinSize(fyne.NewSize(200, 120))

		channelPanels[i] = container.NewBorder(
			container.NewVBox(channelLabel, buttonRow),
			nil, nil, nil,
			queueScroll,
		)
	}

	return channelPanels
}

func (a *App) createHistoryPanel() fyne.CanvasObject {
	historyLabel := widget.NewLabel("✅ 완료 이력:")
	historyLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.historyList = widget.NewList(
		func() int { return len(a.completedJobs) },
		func() fyne.CanvasObject {
			return widget.NewLabel("✅ ITSM-0000 (0m 0s)")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(a.completedJobs) {
				job := a.completedJobs[id]
				obj.(*widget.Label).SetText(fmt.Sprintf("✅ %s - %s", job.IssueKey, job.StartTime))
			}
		},
	)
	a.historyList.OnSelected = func(id widget.ListItemID) {
		if id < len(a.completedJobs) {
			job := a.completedJobs[id]
			a.loadJobResult(job)
		}
	}

	historyScroll := container.NewScroll(a.historyList)
	historyScroll.SetMinSize(fyne.NewSize(760, 80))

	return container.NewBorder(historyLabel, nil, nil, nil, historyScroll)
}
