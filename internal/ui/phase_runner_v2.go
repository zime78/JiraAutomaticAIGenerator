package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/adapter"
	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/logger"
	"jira-ai-generator/internal/ui/state"
)

const (
	// phaseTaskTimeout는 2차 개별 항목의 최대 대기 시간이다.
	phaseTaskTimeout = 30 * time.Minute
	// maxHookRetries는 Hook 오류 발생 시 최대 재시도 횟수이다.
	maxHookRetries = 3
)

var (
	// errTaskCancelled는 사용자가 중지를 요청해 작업이 취소된 경우를 나타낸다.
	errTaskCancelled = errors.New("task cancelled")
)

// phaseRunOutcome는 2차 개별 항목 실행 결과를 전달한다.
type phaseRunOutcome struct {
	record     *domain.IssueRecord
	phaseLabel string
	planPath   string
	err        error
}

// buildHistoryID는 사이드바 이력 식별자를 channel:issueID 형식으로 생성한다.
func buildHistoryID(channelIndex int, issueID int64) string {
	return fmt.Sprintf("%d:%d", channelIndex, issueID)
}

// parseHistoryID는 channel:issueID 형식 문자열을 파싱한다.
func parseHistoryID(historyID string) (int, int64, error) {
	parts := strings.Split(historyID, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid history id format")
	}
	channelIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid history channel: %w", err)
	}
	issueID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid history issue id: %w", err)
	}
	return channelIndex, issueID, nil
}

// listIssuesByMinPhase는 채널의 이슈 중 지정 단계 이상인 항목만 반환한다.
func (a *App) listIssuesByMinPhase(channelIndex, minPhase int) ([]*domain.IssueRecord, error) {
	if a.issueStore == nil {
		return nil, nil
	}
	issues, err := a.issueStore.ListIssuesByChannel(channelIndex)
	if err != nil {
		return nil, err
	}
	filtered := make([]*domain.IssueRecord, 0, len(issues))
	for _, issue := range issues {
		if issue.Phase >= minPhase {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

// nextIssueListLoadToken은 채널별 목록 로딩 요청의 최신 토큰을 증가시킨다.
func (a *App) nextIssueListLoadToken(channelIndex int) uint64 {
	a.issueListLoadMu.Lock()
	defer a.issueListLoadMu.Unlock()
	if channelIndex < 0 || channelIndex >= 3 {
		return 0
	}
	a.issueListLoadSeq[channelIndex]++
	return a.issueListLoadSeq[channelIndex]
}

// isLatestIssueListLoadToken은 주어진 토큰이 현재 채널의 최신 요청인지 확인한다.
func (a *App) isLatestIssueListLoadToken(channelIndex int, token uint64) bool {
	a.issueListLoadMu.Lock()
	defer a.issueListLoadMu.Unlock()
	if channelIndex < 0 || channelIndex >= 3 {
		return false
	}
	return a.issueListLoadSeq[channelIndex] == token
}

// refreshIssueListsForSingleChannel는 특정 채널의 목록을 비동기 로딩하고 최신 요청만 UI에 반영한다.
func (a *App) refreshIssueListsForSingleChannel(channelIndex, phase int, v2 *AppV2State) {
	if channelIndex < 0 || channelIndex >= 3 || v2 == nil {
		return
	}
	selector := v2.analysisSelectors[channelIndex]
	if selector == nil {
		return
	}

	selector.SetPhase1ListLoading(true)

	token := a.nextIssueListLoadToken(channelIndex)
	go func(targetChannel int, loadToken uint64) {
		phase1Issues, errPhase1 := a.listIssuesByMinPhase(targetChannel, 1)

		fyne.Do(func() {
			if !a.isLatestIssueListLoadToken(targetChannel, loadToken) {
				return
			}

			targetSelector := v2.analysisSelectors[targetChannel]
			if targetSelector == nil {
				return
			}

			if errPhase1 != nil {
				logger.Debug("refreshIssueListsForSingleChannel: phase1 load failed (channel=%d): %v", targetChannel, errPhase1)
			} else {
				targetSelector.SetPhase1Items(phase1Issues)
			}
			targetSelector.SetPhase1ListLoading(false)
		})
	}(channelIndex, token)
}

// refreshIssueListsForChannel은 특정 채널의 1차/2차 완료 목록을 갱신한다.
func (a *App) refreshIssueListsForChannel(channelIndex, phase int, v2 *AppV2State) {
	if a.issueStore == nil || v2 == nil {
		return
	}

	if channelIndex >= 0 && channelIndex < 3 {
		a.refreshIssueListsForSingleChannel(channelIndex, phase, v2)
		return
	}
	for i := 0; i < 3; i++ {
		a.refreshIssueListsForSingleChannel(i, phase, v2)
	}
}

// loadHistoryRecordToChannelV2는 사이드바 이력 선택 시 DB 기준으로 결과를 복원한다.
func (a *App) loadHistoryRecordToChannelV2(historyID string, v2 *AppV2State) {
	if a.issueStore == nil || v2 == nil {
		return
	}

	channelIndex, issueID, err := parseHistoryID(historyID)
	if err != nil {
		// 구버전 이력 ID(이슈 키 문자열) 호환 처리
		legacyIssue, legacyErr := a.issueStore.GetIssue(historyID)
		if legacyErr != nil {
			logger.Debug("loadHistoryRecordToChannelV2: parse error: %v", err)
			return
		}
		a.loadIssueRecordToChannelV2(legacyIssue, v2)
		return
	}
	if channelIndex < 0 || channelIndex >= 3 {
		return
	}

	issues, err := a.issueStore.ListIssuesByChannel(channelIndex)
	if err != nil {
		logger.Debug("loadHistoryRecordToChannelV2: list issues failed: %v", err)
		return
	}

	var selected *domain.IssueRecord
	for _, issue := range issues {
		if issue.ID == issueID {
			selected = issue
			break
		}
	}
	if selected == nil {
		return
	}

	a.loadIssueRecordToChannelV2(selected, v2)
}

// loadIssueRecordToChannelV2는 특정 이슈 레코드를 해당 채널 UI에 로드한다.
func (a *App) loadIssueRecordToChannelV2(issue *domain.IssueRecord, v2 *AppV2State) {
	if issue == nil || v2 == nil {
		return
	}
	channelIndex := issue.ChannelIndex
	if channelIndex < 0 || channelIndex >= 3 {
		return
	}
	ch := a.channels[channelIndex]

	if a.tabs != nil {
		a.tabs.SelectIndex(channelIndex)
	}

	issueContent := issue.Description
	if issue.MDPath != "" {
		if raw, err := os.ReadFile(issue.MDPath); err == nil {
			issueContent = string(raw)
		}
	}
	v2.resultPanels[channelIndex].SetIssueInfo(issueContent)
	ch.ResultText.SetText(issueContent)
	ch.CurrentDoc = &domain.GeneratedDocument{
		IssueKey: issue.IssueKey,
		Title:    issue.Summary,
		Content:  issueContent,
	}
	ch.CurrentMDPath = issue.MDPath

	analysisPath := ""
	planPath := ""
	if a.analysisStore != nil {
		if results, err := a.analysisStore.ListAnalysisResultsByIssue(issue.ID); err == nil {
			for _, result := range results {
				if result.PlanPath != "" {
					planPath = result.PlanPath
				}
				if result.ResultPath != "" {
					analysisPath = result.ResultPath
				}
			}
		}
	}
	if analysisPath == "" {
		analysisPath = planPath
	}

	ch.CurrentPlanPath = planPath
	ch.CurrentAnalysisPath = analysisPath

	if analysisPath != "" {
		if raw, err := os.ReadFile(analysisPath); err == nil {
			analysis := string(raw)
			v2.resultPanels[channelIndex].SetAnalysis(analysis)
			ch.AnalysisText.SetText(analysis)
		}
	}

	a.statusLabel.SetText(fmt.Sprintf("이력 로드됨: %s (채널 %d)", issue.IssueKey, channelIndex+1))
}

// registerRunningTask는 채널별 실행 작업을 등록한다.
func (a *App) registerRunningTask(task *RunningTask) {
	if task == nil || task.ChannelIndex < 0 || task.ChannelIndex >= 3 {
		return
	}
	a.runningTasksMu.Lock()
	defer a.runningTasksMu.Unlock()
	a.runningTasks[task.ChannelIndex][task.TaskID] = task
}

// unregisterRunningTask는 채널별 실행 작업을 해제한다.
func (a *App) unregisterRunningTask(channelIndex int, taskID string) {
	a.runningTasksMu.Lock()
	defer a.runningTasksMu.Unlock()
	if channelIndex < 0 || channelIndex >= 3 {
		return
	}
	delete(a.runningTasks[channelIndex], taskID)
}

// markCancelRunningTasks는 채널의 실행 중 작업에 취소 플래그를 설정한다.
func (a *App) markCancelRunningTasks(channelIndex int) []*RunningTask {
	a.runningTasksMu.Lock()
	defer a.runningTasksMu.Unlock()

	if channelIndex < 0 || channelIndex >= 3 {
		return nil
	}

	tasks := make([]*RunningTask, 0, len(a.runningTasks[channelIndex]))
	for _, task := range a.runningTasks[channelIndex] {
		task.CancelRequested = true
		tasks = append(tasks, task)
	}
	return tasks
}

// markRunningTaskCancelledInDB는 사용자 중지 요청을 DB 상태로 기록한다.
func (a *App) markRunningTaskCancelledInDB(task *RunningTask) {
	if task == nil || a.issueStore == nil {
		return
	}

	issues, err := a.issueStore.ListIssuesByChannel(task.ChannelIndex)
	if err == nil {
		for _, issue := range issues {
			if issue.ID == task.IssueID || issue.IssueKey == task.IssueKey {
				issue.Status = "cancelled"
				if updateErr := a.issueStore.UpdateIssue(issue); updateErr != nil {
					logger.Debug("markRunningTaskCancelledInDB: UpdateIssue failed: %v", updateErr)
				}
				break
			}
		}
	}

	if a.analysisStore != nil && task.IssueID > 0 {
		now := time.Now()
		if createErr := a.analysisStore.CreateAnalysisResult(&domain.AnalysisResult{
			IssueID:       task.IssueID,
			AnalysisPhase: 0,
			Status:        "cancelled",
			ErrorMessage:  "cancelled by user",
			CompletedAt:   &now,
		}); createErr != nil {
			logger.Debug("markRunningTaskCancelledInDB: CreateAnalysisResult failed: %v", createErr)
		}
	}
}

// killRunningTask는 등록된 실행 작업의 프로세스를 종료한다.
// PID 기반 종료를 우선하고, 실패 시에만 pkill -f fallback을 사용한다.
func killRunningTask(task *RunningTask) {
	if task == nil {
		return
	}
	if task.PID > 0 {
		exec.Command("kill", "-TERM", strconv.Itoa(task.PID)).Run()
		if !isProcessRunning(task.PID) {
			return
		}
	}
	if task.ScriptPath != "" {
		cleaned := filepath.Clean(task.ScriptPath)
		if filepath.IsAbs(cleaned) {
			exec.Command("pkill", "-f", cleaned).Run()
		}
	}
}

// isProcessRunning은 주어진 PID 프로세스가 실행 중인지 확인한다.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid))
	return cmd.Run() == nil
}

// parseClaudeExitCodeFromLog는 Claude 로그 파일에서 종료 코드를 추출한다.
func parseClaudeExitCodeFromLog(logPath string) (int, bool) {
	if strings.TrimSpace(logPath) == "" {
		return 0, false
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		return 0, false
	}

	marker := "Claude exited with code:"
	idx := strings.LastIndex(string(raw), marker)
	if idx < 0 {
		return 0, false
	}

	rest := strings.TrimSpace(string(raw)[idx+len(marker):])
	if rest == "" {
		return 0, false
	}
	line := rest
	if nl := strings.Index(line, "\n"); nl >= 0 {
		line = line[:nl]
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0, false
	}

	exitCode, convErr := strconv.Atoi(fields[0])
	if convErr != nil {
		return 0, false
	}

	return exitCode, true
}

// extractClaudeFailureReason은 로그 파일에서 실패 원인 후보를 추출한다.
func extractClaudeFailureReason(logPath string) string {
	if strings.TrimSpace(logPath) == "" {
		return ""
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}

	text := string(raw)
	lines := strings.Split(text, "\n")
	candidates := make([]string, 0, 6)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "hook") || strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
			candidates = append(candidates, trimmed)
			if len(candidates) >= 6 {
				break
			}
		}
	}

	if len(candidates) == 0 {
		start := len(lines) - 5
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				candidates = append(candidates, trimmed)
			}
		}
	}

	reason := strings.Join(candidates, " | ")
	if len(reason) > 400 {
		reason = reason[:400] + "..."
	}
	return reason
}

// waitForTaskResult는 프로세스 종료와 결과 파일 생성을 동시에 확인한다.
func waitForTaskResult(task *RunningTask, outputPath string) error {
	deadline := time.Now().Add(phaseTaskTimeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if task.CancelRequested {
			killRunningTask(task)
			return errTaskCancelled
		}

		if !isProcessRunning(task.PID) {
			time.Sleep(500 * time.Millisecond)
			break
		}

		if time.Now().After(deadline) {
			killRunningTask(task)
			return fmt.Errorf("timeout: %s 결과 대기 시간이 30분을 초과했습니다", task.IssueKey)
		}
	}

	if task.CancelRequested {
		return errTaskCancelled
	}

	if _, err := os.Stat(outputPath); err != nil {
		if reason := extractClaudeFailureReason(task.LogPath); reason != "" {
			return fmt.Errorf("결과 파일이 생성되지 않았습니다: %s", reason)
		}
		return fmt.Errorf("결과 파일이 생성되지 않았습니다: %w", err)
	}
	if _, err := os.ReadFile(outputPath); err != nil {
		if reason := extractClaudeFailureReason(task.LogPath); reason != "" {
			return fmt.Errorf("결과 파일 읽기 실패: %s", reason)
		}
		return fmt.Errorf("결과 파일 읽기 실패: %w", err)
	}
	if exitCode, ok := parseClaudeExitCodeFromLog(task.LogPath); ok && exitCode != 0 {
		if reason := extractClaudeFailureReason(task.LogPath); reason != "" {
			return fmt.Errorf("Claude 실행 실패(exit=%d): %s", exitCode, reason)
		}
		return fmt.Errorf("Claude 실행 실패(exit=%d)", exitCode)
	}

	return nil
}

// askRetryForHookFailure는 Hook 설정 오류가 발생했을 때 재시도 여부를 사용자에게 묻는다.
// 5분 타임아웃 후 자동으로 false(건너뛰기)를 반환한다.
func (a *App) askRetryForHookFailure(issueKey, phaseLabel string, hookErr error) bool {
	resultCh := make(chan bool, 1)
	message := fmt.Sprintf("%s %s 중 Hook 오류가 발생했습니다.\n\n사유: %v\n\n재시도하시겠습니까?", issueKey, phaseLabel, hookErr)
	fyne.Do(func() {
		dialog.ShowCustomConfirm(
			"Claude Hook 실행 실패",
			"재시도",
			"건너뛰기",
			widget.NewLabel(message),
			func(retry bool) {
				resultCh <- retry
			},
			a.mainWindow,
		)
	})
	select {
	case result := <-resultCh:
		return result
	case <-time.After(5 * time.Minute):
		return false
	}
}

// isHookRelatedError는 Hook 오류(설정/런타임) 여부를 판별한다.
func isHookRelatedError(err error) bool {
	if err == nil {
		return false
	}
	if adapter.IsHookConfigurationError(err) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "hook")
}

// runPhase2BatchV2는 선택된 1차 완료 항목들을 병렬로 2차 실행한다.
func (a *App) runPhase2BatchV2(channelIndex int, records []*domain.IssueRecord, v2 *AppV2State) {
	a.runPhaseBatchV2(channelIndex, records, "2차", v2)
}

// runPhaseBatchV2는 2차 병렬 실행 흐름을 처리한다.
func (a *App) runPhaseBatchV2(channelIndex int, records []*domain.IssueRecord, phaseLabel string, v2 *AppV2State) {
	if len(records) == 0 {
		return
	}

	workDir := strings.TrimSpace(a.config.Claude.ChannelPaths[channelIndex])
	if workDir == "" {
		fyne.Do(func() {
			v2.appState.FailJob(channelIndex, "", fmt.Errorf("채널 %d 프로젝트 경로 미설정", channelIndex+1))
			v2.progressPanels[channelIndex].SetError("프로젝트 경로 미설정")
		})
		return
	}

	v2.appState.UpdatePhase(channelIndex, state.PhaseAIPlanGeneration)
	fyne.Do(func() {
		v2.progressPanels[channelIndex].SetProgress(0.75, fmt.Sprintf("%s 작업 시작...", phaseLabel))
	})

	resultsCh := make(chan phaseRunOutcome, len(records))
	var wg sync.WaitGroup

	for _, rec := range records {
		record := rec
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultsCh <- a.runPhase2RecordV2(channelIndex, record, workDir, v2)
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	total := len(records)
	doneCount := 0
	successCount := 0
	failedCount := 0

	for outcome := range resultsCh {
		doneCount++
		if outcome.err != nil {
			failedCount++
			logger.Debug("runPhaseBatchV2: %s %s failed: %v", phaseLabel, outcome.record.IssueKey, outcome.err)
		} else {
			successCount++
		}

		progress := 0.75 + (float64(doneCount)/float64(total))*0.25
		message := fmt.Sprintf("%s 진행 중 (%d/%d, 성공 %d, 실패 %d)", phaseLabel, doneCount, total, successCount, failedCount)
		fyne.Do(func() {
			v2.progressPanels[channelIndex].SetProgress(progress, message)
		})
	}

	fyne.Do(func() {
		if successCount == 0 {
			v2.progressPanels[channelIndex].SetError(fmt.Sprintf("%s 모든 항목 실패", phaseLabel))
			v2.appState.UpdatePhase(channelIndex, state.PhaseFailed)
			return
		}

		v2.appState.UpdatePhase(channelIndex, state.PhaseCompleted)
		v2.progressPanels[channelIndex].SetComplete()
	})

	v2.appState.EventBus.Publish(state.Event{
		Type:    state.EventIssueListRefresh,
		Channel: channelIndex,
		Data:    map[string]interface{}{"phase": 1},
	})
}

// runPhase2RecordV2는 단일 이슈의 2차 실행을 처리한다.
func (a *App) runPhase2RecordV2(channelIndex int, record *domain.IssueRecord, workDir string, v2 *AppV2State) phaseRunOutcome {
	outcome := phaseRunOutcome{record: record, phaseLabel: "2차"}
	if record == nil || record.MDPath == "" {
		outcome.err = fmt.Errorf("md path is empty")
		return outcome
	}

	var result *adapter.PlanResult
	var err error
	hookRetryCount := 0
	for {
		result, err = a.claudeAdapter.AnalyzeAndGeneratePlan(record.MDPath, a.config.AI.PromptTemplate, workDir)
		if err == nil {
			task := &RunningTask{
				TaskID:       fmt.Sprintf("phase2:%d:%d", channelIndex, record.ID),
				IssueID:      record.ID,
				IssueKey:     record.IssueKey,
				ChannelIndex: channelIndex,
				PhaseLabel:   "2차",
				PID:          result.PID,
				ScriptPath:   result.ScriptPath,
				LogPath:      result.LogPath,
			}
			a.registerRunningTask(task)
			waitErr := waitForTaskResult(task, result.PlanPath)
			a.unregisterRunningTask(channelIndex, task.TaskID)
			if waitErr == nil {
				break
			}
			if isHookRelatedError(waitErr) && hookRetryCount < maxHookRetries && a.askRetryForHookFailure(record.IssueKey, "2차", waitErr) {
				hookRetryCount++
				continue
			}
			outcome.err = waitErr
			return outcome
		}
		if isHookRelatedError(err) && hookRetryCount < maxHookRetries && a.askRetryForHookFailure(record.IssueKey, "2차", err) {
			hookRetryCount++
			continue
		}
		outcome.err = err
		return outcome
	}

	record.Phase = 2
	if updateErr := a.issueStore.UpdateIssue(record); updateErr != nil {
		outcome.err = updateErr
		return outcome
	}

	if a.analysisStore != nil {
		now := time.Now()
		if createErr := a.analysisStore.CreateAnalysisResult(&domain.AnalysisResult{
			IssueID:       record.ID,
			AnalysisPhase: 1,
			ResultPath:    result.PlanPath,
			PlanPath:      result.PlanPath,
			Status:        "completed",
			CompletedAt:   &now,
		}); createErr != nil {
			logger.Debug("runPhase2RecordV2: CreateAnalysisResult failed: %v", createErr)
		}
	}

	outcome.planPath = result.PlanPath
	analysisContent := fmt.Sprintf("AI 플랜 생성 완료\n이슈: %s\n경로: %s", record.IssueKey, result.PlanPath)
	if raw, readErr := os.ReadFile(result.PlanPath); readErr == nil {
		analysisContent = string(raw)
	}
	fyne.Do(func() {
		v2.resultPanels[channelIndex].SetAnalysis(analysisContent)
		a.channels[channelIndex].AnalysisText.SetText(v2.resultPanels[channelIndex].GetAnalysis())
		a.channels[channelIndex].CurrentPlanPath = result.PlanPath
		a.channels[channelIndex].CurrentAnalysisPath = result.PlanPath
	})

	v2.appState.EventBus.Publish(state.Event{
		Type:    state.EventPhase2Complete,
		Channel: channelIndex,
		Data:    record,
	})
	return outcome
}

