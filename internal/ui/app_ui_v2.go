package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/logger"
	"jira-ai-generator/internal/ui/components"
	"jira-ai-generator/internal/ui/state"
)

// AppV2State 새 UI 상태
type AppV2State struct {
	appState *state.AppState

	// 새 UI 컴포넌트
	sidebar           *components.Sidebar
	progressPanels    [3]*components.ProgressPanel
	resultPanels      [3]*components.ResultPanel
	logViewers        [3]*components.LogViewer
	analysisSelectors [3]*components.AnalysisSelector
	statusBar         *components.StatusBar
}

// initV2State V2 상태 초기화
func (a *App) initV2State() *AppV2State {
	appState := state.NewAppState(a.issueStore, a.analysisStore)
	v2 := &AppV2State{
		appState:  appState,
		sidebar:   components.NewSidebar(appState.EventBus, 0), // 기본 채널 0
		statusBar: components.NewStatusBar(),
	}

	for i := 0; i < 3; i++ {
		v2.progressPanels[i] = components.NewProgressPanel()
		v2.resultPanels[i] = components.NewResultPanel()
		v2.logViewers[i] = components.NewLogViewer()
		v2.analysisSelectors[i] = components.NewAnalysisSelector(appState.EventBus, i)
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
		fyne.Do(func() {
			v2.sidebar.SetActiveChannel(event.Channel)
		})
	})

	// Sidebar 액션 (1차 분석 시작)
	eb.Subscribe(state.EventSidebarAction, func(event state.Event) {
		if data, ok := event.Data.(map[string]interface{}); ok {
			if url, exists := data["url"].(string); exists && url != "" {
				// URL을 채널의 입력창에 설정하고 분석 시작
				if event.Channel >= 0 && event.Channel < 3 {
					a.channels[event.Channel].UrlEntry.SetText(url)
					a.onChannelProcessV2(event.Channel, v2)
				}
			}
		}
	})

	// EventJobStarted 구독 - 2차/3차 분석 작업 실행
	eb.Subscribe(state.EventJobStarted, func(event state.Event) {
		if data, ok := event.Data.(map[string]interface{}); ok {
			phase, _ := data["phase"].(string)
			issueRecords, _ := data["issueRecords"].([]*domain.IssueRecord)

			if event.Channel >= 0 && event.Channel < 3 && len(issueRecords) > 0 {
				switch phase {
				case "phase2":
					// AI 플랜 생성 실행
					go a.executePhase2ForV2(event.Channel, issueRecords, v2)
				case "phase3":
					// AI 실행
					go a.executePhase3ForV2(event.Channel, issueRecords, v2)
				}
			}
		}
	})

	// 이슈 목록 새로고침
	eb.Subscribe(state.EventIssueListRefresh, func(event state.Event) {
		fyne.Do(func() {
			data, _ := event.Data.(map[string]interface{})
			phase, _ := data["phase"].(int)

			switch phase {
			case 1:
				// 1차 완료 목록 로드 (2차 분석 대상)
				issues, err := a.issueStore.ListIssuesByPhase(1)
				if err == nil {
					for i := 0; i < 3; i++ {
						v2.analysisSelectors[i].SetPhase1Items(issues)
					}
				}
			case 2:
				// 2차 완료 목록 로드 (3차 분석 대상)
				issues, err := a.issueStore.ListIssuesByPhase(2)
				if err == nil {
					for i := 0; i < 3; i++ {
						v2.analysisSelectors[i].SetPhase2Items(issues)
					}
				}
			default:
				// 모든 목록 새로고침
				issues1, _ := a.issueStore.ListIssuesByPhase(1)
				issues2, _ := a.issueStore.ListIssuesByPhase(2)
				for i := 0; i < 3; i++ {
					v2.analysisSelectors[i].SetPhase1Items(issues1)
					v2.analysisSelectors[i].SetPhase2Items(issues2)
				}
			}
		})
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

	v2.sidebar.SetOnSettingsClick(func() {
		a.showSettingsDialog()
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

	// URL 입력 위젯 초기화 (다른 곳에서 사용될 수 있으므로 유지)
	ch.UrlEntry = widget.NewEntry()
	ch.UrlEntry.SetPlaceHolder("Jira URL (예: https://domain.atlassian.net/browse/PROJ-123)")

	// 프로젝트 경로 위젯 초기화 (다른 곳에서 사용될 수 있으므로 유지)
	ch.ProjectPathEntry = widget.NewEntry()
	ch.ProjectPathEntry.SetPlaceHolder("프로젝트 경로 (예: /Users/user/MyProject)")
	if a.config.Claude.ChannelPaths[channelIndex] != "" {
		ch.ProjectPathEntry.SetText(a.config.Claude.ChannelPaths[channelIndex])
	}

	// 버튼들 - 중지와 계획 실행만 유지
	stopBtn := widget.NewButtonWithIcon("중지", theme.MediaStopIcon(), func() {
		a.stopQueueCurrent(channelIndex)
	})
	stopBtn.Importance = widget.DangerImportance

	ch.ExecutePlanBtn = widget.NewButtonWithIcon("계획 실행", theme.MailForwardIcon(), func() {
		a.onExecuteChannelPlan(channelIndex)
	})
	ch.ExecutePlanBtn.Importance = widget.WarningImportance
	ch.ExecutePlanBtn.Disable()

	buttonRow := container.NewHBox(stopBtn, ch.ExecutePlanBtn)

	// 상태 라벨
	ch.StatusLabel = widget.NewLabel(fmt.Sprintf("%s 대기 중...", queue.Name))

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

	// 분석 선택기
	analysisSelector := v2.analysisSelectors[channelIndex]

	// 간소화된 상단 섹션
	topSection := container.NewVBox(
		buttonRow,
		widget.NewSeparator(),
		progressPanel,
		ch.StatusLabel,
	)

	// 결과 + 분석 선택기 영역 (수평 분할)
	resultAnalysisSplit := container.NewHSplit(resultPanel, analysisSelector)
	resultAnalysisSplit.SetOffset(0.6) // 결과 60%, 분석 선택기 40%

	// 위 영역 + 로그 영역 (수직 분할)
	mainContentSplit := container.NewVSplit(resultAnalysisSplit, logViewer)
	mainContentSplit.SetOffset(0.7) // 상단 70%, 로그 30%

	return container.NewBorder(topSection, nil, nil, nil, mainContentSplit)
}

// onChannelProcessV2 V2용 채널 처리 핸들러
func (a *App) onChannelProcessV2(channelIndex int, v2 *AppV2State) {
	logger.Debug("onChannelProcessV2: start, channelIndex=%d", channelIndex)
	ch := a.channels[channelIndex]
	url := ch.UrlEntry.Text
	logger.Debug("onChannelProcessV2: url=%s", url)

	if url == "" {
		logger.Debug("onChannelProcessV2: empty URL, showing error dialog")
		dialog.ShowError(fmt.Errorf("Jira URL을 입력해주세요"), a.mainWindow)
		return
	}

	// 프로젝트 경로 확인 (config에서 가져오기)
	workDir := a.config.Claude.ChannelPaths[channelIndex]
	if workDir == "" {
		logger.Debug("onChannelProcessV2: 채널 %d 프로젝트 경로가 설정되지 않았습니다", channelIndex+1)
		dialog.ShowError(fmt.Errorf("채널 %d 프로젝트 경로가 config.ini에 설정되지 않았습니다", channelIndex+1), a.mainWindow)
		return
	}
	logger.Debug("onChannelProcessV2: workDir=%s", workDir)

	// 이전 이슈 정보 및 AI 분석 결과 초기화
	logger.Debug("onChannelProcessV2: resetting previous state")
	ch.CurrentDoc = nil
	ch.CurrentMDPath = ""
	ch.CurrentAnalysisPath = ""
	ch.CurrentPlanPath = ""
	ch.CurrentScriptPath = ""
	v2.resultPanels[channelIndex].Reset()
	v2.progressPanels[channelIndex].Reset()

	// 상태 업데이트
	logger.Debug("onChannelProcessV2: updating phase to PhaseFetchingIssue")
	v2.appState.UpdatePhase(channelIndex, state.PhaseFetchingIssue)
	v2.appState.AddLog(channelIndex, state.LogInfo, "분석 시작: "+url, "App")

	ch.StatusLabel.SetText("분석 중...")

	go func() {
		logger.Debug("onChannelProcessV2: goroutine started for url=%s", url)
		// 진행률 콜백 (usecase.ProgressCallback 형식)
		onProgress := func(progress float64, status string) {
			logger.Debug("onChannelProcessV2: progress=%.2f, status=%s", progress, status)
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
			// UI 업데이트는 메인 스레드에서 실행
			fyne.Do(func() {
				v2.appState.UpdatePhase(channelIndex, phase)
				v2.progressPanels[channelIndex].SetProgress(progress, status)
			})
		}

		logger.Debug("onChannelProcessV2: calling processIssueUC.Execute")
		result, err := a.processIssueUC.Execute(url, onProgress)
		if err != nil {
			logger.Debug("onChannelProcessV2: Execute error: %v", err)
			fyne.Do(func() {
				v2.appState.FailJob(channelIndex, "", err)
				ch.StatusLabel.SetText(fmt.Sprintf("오류: %v", err))
				v2.progressPanels[channelIndex].SetError(err.Error())
			})
			return
		}

		if !result.Success {
			logger.Debug("onChannelProcessV2: result not success: %s", result.ErrorMessage)
			fyne.Do(func() {
				v2.appState.FailJob(channelIndex, "", fmt.Errorf(result.ErrorMessage))
				ch.StatusLabel.SetText(fmt.Sprintf("오류: %s", result.ErrorMessage))
				v2.progressPanels[channelIndex].SetError(result.ErrorMessage)
			})
			return
		}

		logger.Debug("onChannelProcessV2: success, mdPath=%s", result.MDPath)

		// 모든 UI 업데이트를 메인 스레드에서 실행
		fyne.Do(func() {
			v2.appState.UpdatePhase(channelIndex, state.PhaseCompleted)

			ch.CurrentDoc = result.Document
			ch.CurrentMDPath = result.MDPath

			// 결과 표시
			if result.Document != nil {
				logger.Debug("onChannelProcessV2: setting result, issueKey=%s", result.Document.IssueKey)
				v2.resultPanels[channelIndex].SetIssueInfo(result.Document.Content)
				ch.StatusLabel.SetText(fmt.Sprintf("✅ %s 분석 완료", result.Document.IssueKey))
				v2.appState.AddLog(channelIndex, state.LogInfo, "분석 완료: "+result.Document.IssueKey, "App")

				// DB에 저장
				err := v2.appState.SaveIssueToDBAfterPhase1(
					channelIndex,
					result.Document.IssueKey,
					result.Document.Title,
					result.Document.Content,
					url,
					result.MDPath,
				)
				if err != nil {
					logger.Debug("onChannelProcessV2: DB save error: %v", err)
				}

				// 이력에 추가
				v2.sidebar.AddHistoryItem(result.Document.IssueKey, result.Document.IssueKey, "완료", "")

				// 모든 채널의 AnalysisSelector 갱신 (1차 완료 목록)
				allIssues, loadErr := a.issueStore.ListIssuesByPhase(1)
				if loadErr == nil {
					for i := 0; i < 3; i++ {
						v2.analysisSelectors[i].SetPhase1Items(allIssues)
					}
				}
			}
			v2.progressPanels[channelIndex].SetComplete()

			// 사이드바 업데이트
			v2.sidebar.UpdateChannel(channelIndex, "완료", 0)
			logger.Debug("onChannelProcessV2: completed")
		})
	}()
}

// RunV2 V2 UI로 앱 실행
func (a *App) RunV2() {
	a.mainWindow = a.fyneApp.NewWindow("Jira AI Generator v2")
	a.mainWindow.Resize(fyne.NewSize(1920, 1080))
	a.mainWindow.CenterOnScreen()

	v2 := a.initV2State()
	content := a.createMainContentV2(v2)
	a.mainWindow.SetContent(content)

	// DB에서 이전 분석 이력 로드
	a.loadHistoryFromDB(v2)

	a.loadPreviousAnalysis()

	// Note: DB close는 main.go의 defer app.Close()에서 처리

	a.mainWindow.ShowAndRun()
}

// loadHistoryFromDB DB에서 이전 분석 이력을 로드하여 사이드바와 AnalysisSelector에 표시
func (a *App) loadHistoryFromDB(v2 *AppV2State) {
	if a.issueStore == nil {
		logger.Debug("loadHistoryFromDB: issueStore is nil, skipping")
		return
	}

	// Phase 1 완료된 모든 이슈 로드
	issues, err := a.issueStore.ListIssuesByPhase(1)
	if err != nil {
		logger.Debug("loadHistoryFromDB: failed to load issues: %v", err)
		return
	}

	logger.Debug("loadHistoryFromDB: loaded %d issues from DB", len(issues))

	// 사이드바에 이력 추가
	for _, issue := range issues {
		v2.sidebar.AddHistoryItem(issue.IssueKey, issue.IssueKey, "완료", "")
	}

	// 모든 채널의 AnalysisSelector에 동일한 Phase 1 완료 항목 로드
	for channelIdx := 0; channelIdx < 3; channelIdx++ {
		logger.Debug("loadHistoryFromDB: loading %d issues to channel %d AnalysisSelector", len(issues), channelIdx)
		v2.analysisSelectors[channelIdx].SetPhase1Items(issues)
	}
}

// executePhase2ForV2 V2용 2차 분석 (AI 플랜 생성)
func (a *App) executePhase2ForV2(channelIndex int, records []*domain.IssueRecord, v2 *AppV2State) {
	logger.Debug("executePhase2ForV2: start, channel=%d, record_count=%d", channelIndex, len(records))

	workDir := a.config.Claude.ChannelPaths[channelIndex]
	if workDir == "" {
		logger.Debug("executePhase2ForV2: workDir is empty for channel %d", channelIndex)
		fyne.Do(func() {
			v2.appState.FailJob(channelIndex, "", fmt.Errorf("채널 %d 프로젝트 경로 미설정", channelIndex+1))
			v2.progressPanels[channelIndex].SetError("프로젝트 경로 미설정")
		})
		return
	}

	totalRecords := len(records)
	for idx, record := range records {
		logger.Debug("executePhase2ForV2: processing record id=%d, key=%s, mdPath=%s", record.ID, record.IssueKey, record.MDPath)

		// 진행률 업데이트 - 시작 (75% → 95% 범위)
		progress := 0.75 + (float64(idx) / float64(totalRecords) * 0.20)
		fyne.Do(func() {
			v2.progressPanels[channelIndex].SetProgress(progress, fmt.Sprintf("AI 플랜 생성 중: %s (%d/%d)", record.IssueKey, idx+1, totalRecords))
		})

		// MD 파일 경로 확인
		mdPath := record.MDPath
		if mdPath == "" {
			logger.Debug("executePhase2ForV2: mdPath is empty for record %s", record.IssueKey)
			continue
		}

		// AI 플랜 생성
		prompt := a.config.AI.PromptTemplate
		logger.Debug("executePhase2ForV2: calling AnalyzeAndGeneratePlan, mdPath=%s, workDir=%s", mdPath, workDir)

		result, err := a.claudeAdapter.AnalyzeAndGeneratePlan(mdPath, prompt, workDir)

		fyne.Do(func() {
			if err != nil {
				logger.Debug("executePhase2ForV2: AnalyzeAndGeneratePlan error: %v", err)
				v2.appState.FailJob(channelIndex, record.IssueKey, err)
				v2.progressPanels[channelIndex].SetError(err.Error())
				v2.appState.UpdatePhase(channelIndex, state.PhaseFailed)
				return
			}

			logger.Debug("executePhase2ForV2: AnalyzeAndGeneratePlan success")

			// 성공 시 Phase 변경 및 진행률 업데이트
			v2.appState.UpdatePhase(channelIndex, state.PhaseAIPlanReady)
			v2.progressPanels[channelIndex].SetProgress(0.95, "AI 플랜 생성 완료")

			// 결과 표시
			if result != nil {
				v2.resultPanels[channelIndex].SetAnalysis(fmt.Sprintf("AI 플랜 생성 완료\n경로: %s", result.PlanPath))
				a.channels[channelIndex].CurrentPlanPath = result.PlanPath
			}

			// DB에서 이슈 Phase 업데이트
			record.Phase = 2
			if a.issueStore != nil {
				a.issueStore.UpdateIssue(record)
			}

			// Phase2 완료 이벤트 발행
			v2.appState.EventBus.Publish(state.Event{
				Type:    state.EventPhase2Complete,
				Channel: channelIndex,
				Data:    record,
			})

			// 3차 분석 대상 목록 자동 갱신
			v2.appState.EventBus.Publish(state.Event{
				Type:    state.EventIssueListRefresh,
				Channel: channelIndex,
				Data:    map[string]interface{}{"phase": 2},
			})

			logger.Debug("executePhase2ForV2: completed for record %s", record.IssueKey)
		})
	}

	// 모든 레코드 처리 완료 후 최종 완료 표시
	fyne.Do(func() {
		v2.progressPanels[channelIndex].SetComplete()
	})
}

// executePhase3ForV2 V2용 3차 분석 (AI 실행)
func (a *App) executePhase3ForV2(channelIndex int, records []*domain.IssueRecord, v2 *AppV2State) {
	logger.Debug("executePhase3ForV2: start, channel=%d, record_count=%d", channelIndex, len(records))

	workDir := a.config.Claude.ChannelPaths[channelIndex]
	if workDir == "" {
		logger.Debug("executePhase3ForV2: workDir is empty for channel %d", channelIndex)
		fyne.Do(func() {
			v2.appState.FailJob(channelIndex, "", fmt.Errorf("채널 %d 프로젝트 경로 미설정", channelIndex+1))
			v2.progressPanels[channelIndex].SetError("프로젝트 경로 미설정")
		})
		return
	}

	// 시작 진행률 설정
	fyne.Do(func() {
		v2.progressPanels[channelIndex].SetProgress(0.75, "AI 플랜 실행 준비 중...")
	})

	totalRecords := len(records)
	for idx, record := range records {
		logger.Debug("executePhase3ForV2: processing record id=%d, key=%s", record.ID, record.IssueKey)

		// 진행률 업데이트 - 시작 (75% → 95% 범위)
		progress := 0.75 + (float64(idx) / float64(totalRecords) * 0.20)
		fyne.Do(func() {
			v2.progressPanels[channelIndex].SetProgress(progress, fmt.Sprintf("AI 플랜 실행 중: %s (%d/%d)", record.IssueKey, idx+1, totalRecords))
		})

		// 플랜 파일 경로 - 채널의 CurrentPlanPath 사용
		planPath := a.channels[channelIndex].CurrentPlanPath
		if planPath == "" {
			logger.Debug("executePhase3ForV2: planPath is empty for channel %d", channelIndex)
			continue
		}

		// AI 플랜 실행
		logger.Debug("executePhase3ForV2: calling ExecutePlan, planPath=%s, workDir=%s", planPath, workDir)
		result, err := a.claudeAdapter.ExecutePlan(planPath, workDir)

		fyne.Do(func() {
			if err != nil {
				logger.Debug("executePhase3ForV2: ExecutePlan error: %v", err)
				v2.appState.FailJob(channelIndex, record.IssueKey, err)
				v2.progressPanels[channelIndex].SetError(err.Error())
				v2.appState.UpdatePhase(channelIndex, state.PhaseFailed)
				return
			}

			logger.Debug("executePhase3ForV2: ExecutePlan success")

			// 성공 시 Phase 변경
			v2.appState.UpdatePhase(channelIndex, state.PhaseCompleted)

			// 결과 표시
			if result != nil {
				v2.resultPanels[channelIndex].SetAnalysis(fmt.Sprintf("AI 실행 완료\n출력: %s", result.OutputPath))
			}

			// Phase3 완료 이벤트 발행
			v2.appState.EventBus.Publish(state.Event{
				Type:    state.EventPhase3Complete,
				Channel: channelIndex,
				Data:    record,
			})

			logger.Debug("executePhase3ForV2: completed for record %s", record.IssueKey)
		})
	}

	// 모든 레코드 처리 완료 후 최종 완료 표시
	fyne.Do(func() {
		v2.progressPanels[channelIndex].SetComplete()
	})
}
