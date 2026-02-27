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

// queueListItemView는 큐 목록 한 줄에 표시할 아이콘/텍스트를 표현한다.
type queueListItemView struct {
	Icon string
	Text string
}

// queueListItemCount는 큐 목록에 표시할 전체 아이템 수를 반환한다.
func queueListItemCount(q *AnalysisQueue) int {
	if q == nil {
		return 0
	}

	count := len(q.Pending) + len(q.Completed) + len(q.Failed) + len(q.Cancelled)
	if q.Current != nil {
		count++
	}
	return count
}

// resolveQueueListItemView는 인덱스에 해당하는 큐 아이템의 표시 정보를 계산한다.
// 표시 순서는 Current -> Pending -> Completed -> Failed -> Cancelled 이다.
func resolveQueueListItemView(q *AnalysisQueue, id int) (queueListItemView, bool) {
	if q == nil || id < 0 {
		return queueListItemView{}, false
	}

	currentCount := 0
	if q.Current != nil {
		currentCount = 1
	}

	if id == 0 && q.Current != nil {
		return queueListItemView{
			Icon: "▶",
			Text: q.Current.IssueKey,
		}, true
	}

	pendingCount := len(q.Pending)
	if id < currentCount+pendingCount {
		pendingIdx := id - currentCount
		if pendingIdx >= 0 && pendingIdx < len(q.Pending) {
			return queueListItemView{
				Icon: "⏳",
				Text: q.Pending[pendingIdx].IssueKey,
			}, true
		}
	}

	completedStart := currentCount + pendingCount
	if id < completedStart+len(q.Completed) {
		completedIdx := id - completedStart
		if completedIdx >= 0 && completedIdx < len(q.Completed) {
			return queueListItemView{
				Icon: "✓",
				Text: q.Completed[completedIdx].IssueKey + " (완료)",
			}, true
		}
	}

	failedStart := completedStart + len(q.Completed)
	if id < failedStart+len(q.Failed) {
		failedIdx := id - failedStart
		if failedIdx >= 0 && failedIdx < len(q.Failed) {
			return queueListItemView{
				Icon: "✗",
				Text: q.Failed[failedIdx].IssueKey + " (실패)",
			}, true
		}
	}

	cancelledStart := failedStart + len(q.Failed)
	if id < cancelledStart+len(q.Cancelled) {
		cancelledIdx := id - cancelledStart
		if cancelledIdx >= 0 && cancelledIdx < len(q.Cancelled) {
			return queueListItemView{
				Icon: "⏹",
				Text: q.Cancelled[cancelledIdx].IssueKey + " (중단)",
			}, true
		}
	}

	return queueListItemView{}, false
}

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

	refreshBtn := widget.NewButtonWithIcon("새로고침", theme.ViewRefreshIcon(), func() {
		a.onRefreshChannelAnalysis(channelIndex)
	})

	buttonRow := container.NewHBox(ch.ProcessBtn, addBtn, stopBtn, refreshBtn)

	// 진행바
	ch.ProgressBar = widget.NewProgressBar()
	ch.ProgressBar.Hide()

	// 큐 목록
	ch.QueueList = widget.NewList(
		func() int {
			return queueListItemCount(a.queues[channelIndex])
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
			iconLabel := box.Objects[0].(*widget.Label)
			textLabel := box.Objects[1].(*widget.Label)

			item, ok := resolveQueueListItemView(q, int(id))
			if !ok {
				iconLabel.SetText("")
				textLabel.SetText("")
				return
			}
			iconLabel.SetText(item.Icon)
			textLabel.SetText(item.Text)
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

	return container.NewBorder(topSection, nil, nil, nil, issuePanel)
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
