package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/ui/components"
	"jira-ai-generator/internal/ui/state"
)

// AppV2State 새 UI 상태
type AppV2State struct {
	appState *state.AppState

	// 새 UI 컴포넌트
	sidebar        *components.Sidebar
	progressPanels [3]*components.ProgressPanel
	resultPanels   [3]*components.ResultPanel
	logViewers     [3]*components.LogViewer
	statusBar      *components.StatusBar
}

// initV2State V2 상태 초기화
func (a *App) initV2State() *AppV2State {
	v2 := &AppV2State{
		appState:  state.NewAppState(),
		sidebar:   components.NewSidebar(),
		statusBar: components.NewStatusBar(),
	}

	for i := 0; i < 3; i++ {
		v2.progressPanels[i] = components.NewProgressPanel()
		v2.resultPanels[i] = components.NewResultPanel()
		v2.logViewers[i] = components.NewLogViewer()
	}

	// 이벤트 구독
	v2.subscribeEvents(a)

	return v2
}

// subscribeEvents 이벤트 구독 설정
func (v2 *AppV2State) subscribeEvents(a *App) {
	eb := v2.appState.EventBus

	// 진행률 업데이트
	eb.Subscribe(state.EventProgressUpdate, func(event state.Event) {
		if data, ok := event.Data.(state.ProgressData); ok {
			if event.Channel >= 0 && event.Channel < 3 {
				v2.progressPanels[event.Channel].SetProgress(data.Progress, data.Message)
			}
		}
	})

	// 단계 변경
	eb.Subscribe(state.EventPhaseChange, func(event state.Event) {
		if phase, ok := event.Data.(state.ProcessPhase); ok {
			if event.Channel >= 0 && event.Channel < 3 {
				v2.progressPanels[event.Channel].SetPhase(phase)

				// 사이드바 채널 상태 업데이트
				ch := v2.appState.GetChannel(event.Channel)
				if ch != nil {
					v2.sidebar.UpdateChannel(event.Channel, phase.String(), len(ch.Queue))
				}

				// 상태바 채널 상태 업데이트
				v2.statusBar.SetChannelStatus(event.Channel, phase)
			}
		}
	})

	// 로그 추가
	eb.Subscribe(state.EventLogAdded, func(event state.Event) {
		if data, ok := event.Data.(state.LogData); ok {
			if event.Channel >= 0 && event.Channel < 3 {
				v2.logViewers[event.Channel].AddLog(data.Level, data.Message, data.Source)
			}
		}
	})

	// 작업 완료
	eb.Subscribe(state.EventJobCompleted, func(event state.Event) {
		if data, ok := event.Data.(map[string]interface{}); ok {
			jobID := fmt.Sprintf("%v", data["jobID"])
			if event.Channel >= 0 && event.Channel < 3 {
				v2.progressPanels[event.Channel].SetComplete()
				v2.sidebar.AddHistoryItem(jobID, jobID, "completed", "")
				v2.statusBar.SetRecentActivity(fmt.Sprintf("✅ %s 완료", jobID))
			}
		}
	})

	// 작업 실패
	eb.Subscribe(state.EventJobFailed, func(event state.Event) {
		if data, ok := event.Data.(map[string]interface{}); ok {
			if event.Channel >= 0 && event.Channel < 3 {
				errMsg := fmt.Sprintf("%v", data["error"])
				v2.progressPanels[event.Channel].SetError(errMsg)
				v2.statusBar.SetGlobalStatus("오류 발생", true)
			}
		}
	})

	// 채널 전환
	eb.Subscribe(state.EventChannelSwitch, func(event state.Event) {
		v2.sidebar.SetActiveChannel(event.Channel)
	})
}

// createMainContentV2 새 레이아웃으로 메인 콘텐츠 생성
func (a *App) createMainContentV2(v2 *AppV2State) fyne.CanvasObject {
	// 헤더
	title := widget.NewLabelWithStyle(
		"🔧 Jira AI Generator",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	a.stopAllBtn = widget.NewButtonWithIcon("전체 중지", theme.MediaStopIcon(), a.onStopAllQueues)
	a.stopAllBtn.Importance = widget.DangerImportance
	if !a.claudeAdapter.IsEnabled() {
		a.stopAllBtn.Hide()
	}

	a.statusLabel = widget.NewLabel("준비됨")

	header := container.NewBorder(nil, nil, title, a.stopAllBtn)

	// 사이드바 콜백 설정
	v2.sidebar.SetOnChannelSelect(func(index int) {
		a.tabs.SelectIndex(index)
		v2.appState.SetActiveChannel(index)
	})

	v2.sidebar.SetOnHistorySelect(func(jobID string) {
		a.mu.Lock()
		var job *AnalysisJob
		for _, j := range a.completedJobs {
			if j.IssueKey == jobID {
				job = j
				break
			}
		}
		a.mu.Unlock()

		if job != nil {
			a.loadJobResultToChannel(job)
		}
	})

	// 채널 탭 생성 (새 컴포넌트 사용)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("채널 1", a.createChannelTabV2(0, v2)),
		container.NewTabItem("채널 2", a.createChannelTabV2(1, v2)),
		container.NewTabItem("채널 3", a.createChannelTabV2(2, v2)),
	)
	a.tabs.SetTabLocation(container.TabLocationTop)

	a.tabs.OnChanged = func(tab *container.TabItem) {
		for i, t := range a.tabs.Items {
			if t == tab {
				v2.appState.SetActiveChannel(i)
				break
			}
		}
	}

	// 사이드바 + 메인 콘텐츠 레이아웃
	sidebarContainer := container.NewVBox(v2.sidebar)
	sidebarScroll := container.NewScroll(sidebarContainer)
	sidebarScroll.SetMinSize(fyne.NewSize(200, 0))

	mainArea := container.NewBorder(
		container.NewVBox(header, a.statusLabel),
		nil,
		nil,
		nil,
		a.tabs,
	)

	// HSplit으로 사이드바와 메인 영역 분리
	split := container.NewHSplit(sidebarScroll, mainArea)
	split.SetOffset(0.15) // 사이드바 15%

	// 메인 콘텐츠 + 상태바
	mainWithStatusBar := container.NewBorder(
		nil,
		v2.statusBar, // 하단에 상태바
		nil,
		nil,
		split,
	)

	return container.NewPadded(mainWithStatusBar)
}

// createChannelTabV2 새 컴포넌트를 사용한 채널 탭 생성
func (a *App) createChannelTabV2(channelIndex int, v2 *AppV2State) fyne.CanvasObject {
	ch := a.channels[channelIndex]
	queue := a.queues[channelIndex]

	// URL 입력
	ch.UrlEntry = widget.NewEntry()
	ch.UrlEntry.SetPlaceHolder("Jira URL (예: https://domain.atlassian.net/browse/PROJ-123)")

	// 프로젝트 경로
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

	// 버튼들
	ch.ProcessBtn = widget.NewButtonWithIcon("분석 시작", theme.MediaPlayIcon(), func() {
		a.onChannelProcessV2(channelIndex, v2)
	})
	ch.ProcessBtn.Importance = widget.HighImportance

	addBtn := widget.NewButtonWithIcon("큐 추가", theme.ContentAddIcon(), func() {
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

	buttonRow := container.NewHBox(ch.ProcessBtn, addBtn, stopBtn, ch.ExecutePlanBtn)

	// 상태 라벨
	ch.StatusLabel = widget.NewLabel(fmt.Sprintf("%s 대기 중...", queue.Name))

	// 입력 섹션
	inputSection := container.NewVBox(
		widget.NewLabelWithStyle("📥 Jira URL", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ch.UrlEntry,
		widget.NewLabel("프로젝트 경로:"),
		projectPathRow,
		buttonRow,
	)

	// 진행률 패널
	progressPanel := v2.progressPanels[channelIndex]

	// 결과 패널
	resultPanel := v2.resultPanels[channelIndex]

	// 결과 패널 콜백 설정
	resultPanel.SetOnCopyIssue(func() {
		a.onCopyChannelResult(channelIndex)
	})
	resultPanel.SetOnCopyAnalysis(func() {
		a.onCopyChannelAnalysis(channelIndex)
	})
	resultPanel.SetOnExecutePlan(func() {
		a.onExecuteChannelPlan(channelIndex)
	})

	// 기존 위젯 참조 연결 (호환성)
	ch.ProgressBar = widget.NewProgressBar()
	ch.ResultText = widget.NewMultiLineEntry()
	ch.AnalysisText = widget.NewMultiLineEntry()

	// 로그 뷰어
	logViewer := v2.logViewers[channelIndex]

	// 레이아웃 구성
	topSection := container.NewVBox(
		inputSection,
		widget.NewSeparator(),
		progressPanel,
		ch.StatusLabel,
	)

	// 결과 + 로그 영역 (수직 분할)
	resultLogSplit := container.NewVSplit(resultPanel, logViewer)
	resultLogSplit.SetOffset(0.7) // 결과 70%, 로그 30%

	return container.NewBorder(topSection, nil, nil, nil, resultLogSplit)
}

// onChannelProcessV2 V2용 채널 처리 핸들러
func (a *App) onChannelProcessV2(channelIndex int, v2 *AppV2State) {
	ch := a.channels[channelIndex]
	url := ch.UrlEntry.Text

	if url == "" {
		dialog.ShowError(fmt.Errorf("Jira URL을 입력해주세요"), a.mainWindow)
		return
	}

	// 이전 이슈 정보 및 AI 분석 결과 초기화
	ch.CurrentDoc = nil
	ch.CurrentMDPath = ""
	ch.CurrentAnalysisPath = ""
	ch.CurrentPlanPath = ""
	ch.CurrentScriptPath = ""
	v2.resultPanels[channelIndex].Reset()
	v2.progressPanels[channelIndex].Reset()

	// 상태 업데이트
	v2.appState.UpdatePhase(channelIndex, state.PhaseFetchingIssue)
	v2.appState.AddLog(channelIndex, state.LogInfo, "분석 시작: "+url, "App")

	ch.StatusLabel.SetText("분석 중...")

	go func() {
		// 진행률 콜백 (usecase.ProgressCallback 형식)
		onProgress := func(progress float64, status string) {
			// 진행률에 따라 단계 결정
			var phase state.ProcessPhase
			switch {
			case progress < 0.2:
				phase = state.PhaseFetchingIssue
			case progress < 0.4:
				phase = state.PhaseDownloadingAttachments
			case progress < 0.6:
				phase = state.PhaseExtractingFrames
			case progress < 0.8:
				phase = state.PhaseGeneratingDocument
			default:
				phase = state.PhaseAnalyzing
			}
			v2.appState.UpdatePhase(channelIndex, phase)
			v2.progressPanels[channelIndex].SetProgress(progress, status)
		}

		result, err := a.processIssueUC.Execute(url, onProgress)
		if err != nil {
			v2.appState.FailJob(channelIndex, "", err)
			ch.StatusLabel.SetText(fmt.Sprintf("오류: %v", err))
			v2.progressPanels[channelIndex].SetError(err.Error())
			return
		}

		if !result.Success {
			v2.appState.FailJob(channelIndex, "", fmt.Errorf(result.ErrorMessage))
			ch.StatusLabel.SetText(fmt.Sprintf("오류: %s", result.ErrorMessage))
			v2.progressPanels[channelIndex].SetError(result.ErrorMessage)
			return
		}

		v2.appState.UpdatePhase(channelIndex, state.PhaseCompleted)

		ch.CurrentDoc = result.Document
		ch.CurrentMDPath = result.MDPath

		// 결과 표시
		if result.Document != nil {
			v2.resultPanels[channelIndex].SetIssueInfo(result.Document.Content)
			ch.StatusLabel.SetText(fmt.Sprintf("✅ %s 분석 완료", result.Document.IssueKey))
			v2.appState.AddLog(channelIndex, state.LogInfo, "분석 완료: "+result.Document.IssueKey, "App")
		}
		v2.progressPanels[channelIndex].SetComplete()

		// 사이드바 업데이트
		v2.sidebar.UpdateChannel(channelIndex, "완료", 0)
	}()
}

// UseV2UI V2 UI 사용 여부 (환경변수나 설정으로 제어 가능)
func (a *App) UseV2UI() bool {
	// V2 UI 활성화
	return true
}

// RunV2 V2 UI로 앱 실행
func (a *App) RunV2() {
	a.mainWindow = a.fyneApp.NewWindow("Jira AI Generator v2")
	a.mainWindow.Resize(fyne.NewSize(1600, 1000))
	a.mainWindow.CenterOnScreen()

	v2 := a.initV2State()
	content := a.createMainContentV2(v2)
	a.mainWindow.SetContent(content)

	a.loadPreviousAnalysis()

	a.mainWindow.ShowAndRun()
}
