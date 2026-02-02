package ui

// Deprecated: This file contains V1 UI code.
// Use app_ui_v2.go instead.
// This file will be removed in the next major version.

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

	a.stopAllBtn = widget.NewButtonWithIcon("전체 중지", theme.MediaStopIcon(), a.onStopAllQueues)
	a.stopAllBtn.Importance = widget.DangerImportance
	if !a.claudeAdapter.IsEnabled() {
		a.stopAllBtn.Hide()
	}

	a.statusLabel = widget.NewLabel("대기 중...")

	headerRow := container.NewBorder(nil, nil, title, a.stopAllBtn)

	// 채널 탭 생성
	a.tabs = container.NewAppTabs(
		container.NewTabItem("채널 1", a.createChannelTab(0)),
		container.NewTabItem("채널 2", a.createChannelTab(1)),
		container.NewTabItem("채널 3", a.createChannelTab(2)),
	)
	a.tabs.SetTabLocation(container.TabLocationTop)

	// 공유 완료 이력
	historyContainer := a.createHistoryPanel()

	contentArea := container.NewBorder(nil, historyContainer, nil, nil, a.tabs)

	mainLayout := container.NewBorder(
		container.NewVBox(headerRow, a.statusLabel),
		nil, nil, nil,
		contentArea,
	)

	return container.NewPadded(mainLayout)
}

// createChannelTab은 채널별 전체 워크스페이스를 생성한다.
func (a *App) createChannelTab(channelIndex int) fyne.CanvasObject {
	ch := a.channels[channelIndex]
	queue := a.queues[channelIndex]

	// URL 입력
	ch.UrlEntry = widget.NewEntry()
	ch.UrlEntry.SetPlaceHolder("Jira URL (예: https://domain.atlassian.net/browse/PROJ-123)")

	// 프로젝트 경로 (채널별 설정)
	ch.ProjectPathEntry = widget.NewEntry()
	ch.ProjectPathEntry.SetPlaceHolder("프로젝트 경로 (예: /Users/user/MyProject)")
	if a.config.Claude.ChannelPaths[channelIndex] != "" {
		ch.ProjectPathEntry.SetText(a.config.Claude.ChannelPaths[channelIndex])
	}
	browseBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				ch.ProjectPathEntry.SetText(uri.Path())
			}
		}, a.mainWindow)
	})
	projectPathRow := container.NewBorder(nil, nil, nil, browseBtn, ch.ProjectPathEntry)

	// 분석 시작 버튼
	ch.ProcessBtn = widget.NewButtonWithIcon("분석 시작", theme.MediaPlayIcon(), func() {
		a.onChannelProcess(channelIndex)
	})
	ch.ProcessBtn.Importance = widget.HighImportance

	// 큐 컨트롤 버튼
	addBtn := widget.NewButtonWithIcon("추가", theme.ContentAddIcon(), func() {
		a.addToQueue(channelIndex)
	})
	stopBtn := widget.NewButtonWithIcon("중지", theme.MediaStopIcon(), func() {
		a.stopQueueCurrent(channelIndex)
	})
	stopBtn.Importance = widget.DangerImportance

	ch.ExecutePlanBtn = widget.NewButtonWithIcon("계획 실행", theme.MailForwardIcon(), func() {
		a.onExecuteChannelPlan(channelIndex)
	})
	ch.ExecutePlanBtn.Importance = widget.WarningImportance
	ch.ExecutePlanBtn.Disable()
	if !a.claudeAdapter.IsEnabled() {
		ch.ExecutePlanBtn.Hide()
	}

	refreshBtn := widget.NewButtonWithIcon("새로고침", theme.ViewRefreshIcon(), func() {
		a.onRefreshChannelAnalysis(channelIndex)
	})

	buttonRow := container.NewHBox(ch.ProcessBtn, addBtn, stopBtn, ch.ExecutePlanBtn, refreshBtn)

	// 진행바
	ch.ProgressBar = widget.NewProgressBar()
	ch.ProgressBar.Hide()

	// 큐 목록
	ch.QueueList = widget.NewList(
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

	queueScroll := container.NewScroll(ch.QueueList)
	queueScroll.SetMinSize(fyne.NewSize(760, 80))

	// 상태 라벨
	ch.StatusLabel = widget.NewLabel(fmt.Sprintf("%s 대기 중...", queue.Name))

	// === 내부 서브탭: 이슈 정보 + AI 분석 ===

	// 이슈 정보 패널
	ch.ResultText = widget.NewMultiLineEntry()
	ch.ResultText.SetPlaceHolder("이슈 정보가 여기에 표시됩니다...")
	ch.ResultText.Wrapping = fyne.TextWrapWord
	ch.CopyResultBtn = widget.NewButtonWithIcon("이슈 복사", theme.ContentCopyIcon(), func() {
		a.onCopyChannelResult(channelIndex)
	})
	ch.CopyResultBtn.Disable()
	issueScroll := container.NewScroll(ch.ResultText)
	issuePanel := container.NewBorder(
		container.NewHBox(ch.CopyResultBtn), nil, nil, nil, issueScroll,
	)

	// AI 분석 패널
	ch.AnalysisText = widget.NewMultiLineEntry()
	ch.AnalysisText.SetPlaceHolder(fmt.Sprintf("%s AI 분석 결과가 여기에 표시됩니다...", queue.Name))
	ch.AnalysisText.Wrapping = fyne.TextWrapWord
	ch.CopyAnalysisBtn = widget.NewButtonWithIcon("분석 복사", theme.ContentCopyIcon(), func() {
		a.onCopyChannelAnalysis(channelIndex)
	})
	ch.CopyAnalysisBtn.Disable()
	analysisScroll := container.NewScroll(ch.AnalysisText)
	analysisPanel := container.NewBorder(
		container.NewHBox(ch.CopyAnalysisBtn), nil, nil, nil, analysisScroll,
	)

	// 내부 서브탭
	ch.InnerTabs = container.NewAppTabs(
		container.NewTabItem("이슈 정보", issuePanel),
		container.NewTabItem("AI 분석 결과", analysisPanel),
	)
	ch.InnerTabs.SetTabLocation(container.TabLocationTop)

	// 최종 레이아웃
	topSection := container.NewVBox(
		widget.NewLabel("Jira URL:"),
		ch.UrlEntry,
		widget.NewLabel("프로젝트 경로 (AI 분석용):"),
		projectPathRow,
		buttonRow,
		ch.ProgressBar,
		queueScroll,
		ch.StatusLabel,
	)

	return container.NewBorder(topSection, nil, nil, nil, ch.InnerTabs)
}

// createHistoryPanel은 모든 채널의 완료 이력을 공유하는 패널을 생성한다.
func (a *App) createHistoryPanel() fyne.CanvasObject {
	historyLabel := widget.NewLabel("완료 이력:")
	historyLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.historyList = widget.NewList(
		func() int {
			a.mu.Lock()
			defer a.mu.Unlock()
			return len(a.completedJobs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("ITSM-0000 (0m 0s)")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			a.mu.Lock()
			defer a.mu.Unlock()
			if id < len(a.completedJobs) {
				job := a.completedJobs[id]
				obj.(*widget.Label).SetText(fmt.Sprintf("%s - %s", job.IssueKey, job.StartTime))
			}
		},
	)
	a.historyList.OnSelected = func(id widget.ListItemID) {
		a.mu.Lock()
		var job *AnalysisJob
		if id < len(a.completedJobs) {
			job = a.completedJobs[id]
		}
		a.mu.Unlock()

		if job != nil {
			a.loadJobResultToChannel(job)
		}
	}

	historyScroll := container.NewScroll(a.historyList)
	historyScroll.SetMinSize(fyne.NewSize(760, 80))

	return container.NewBorder(historyLabel, nil, nil, nil, historyScroll)
}
