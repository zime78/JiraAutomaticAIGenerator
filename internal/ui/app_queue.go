package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/adapter"
	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/ui/state"
)

// ChannelState는 각 채널의 독립적인 UI 및 상태를 관리한다.
type ChannelState struct {
	Index int
	Name  string

	// 채널별 입력 위젯
	UrlEntry         *widget.Entry
	ProjectPathEntry *widget.Entry
	ProcessBtn       *widget.Button
	ProgressBar      *widget.ProgressBar

	// 채널별 결과 위젯
	ResultText      *widget.Entry
	AnalysisText    *widget.Entry
	StatusLabel     *widget.Label
	CopyResultBtn   *widget.Button
	CopyAnalysisBtn *widget.Button
	QueueList *widget.List
	InnerTabs       *container.AppTabs

	// 채널별 상태
	CurrentDoc          *domain.GeneratedDocument
	CurrentMDPath       string
	CurrentAnalysisPath string
	CurrentPlanPath     string
	CurrentScriptPath   string
}

// AnalysisJob represents a running analysis task
type AnalysisJob struct {
	IssueKey        string
	ScriptPath      string
	AnalysisPath    string
	PlanPath string // Phase 1 결과: _plan.md 경로
	LogPath  string // 로그 파일 경로
	MDPath          string
	StartTime       string
	PID             int
	Phase           adapter.AnalysisPhase // 현재 실행 단계
	ChannelIndex    int                   // 실행된 채널 인덱스
	CancelRequested bool                  // 사용자 중단 요청 여부
	Outcome         QueueJobOutcome       // 마지막 실행 결과
}

// QueueJobOutcome은 큐 작업의 최종 실행 결과를 나타낸다.
type QueueJobOutcome int

const (
	// QueueJobOutcomeCompleted는 작업이 정상 완료되었음을 의미한다.
	QueueJobOutcomeCompleted QueueJobOutcome = iota
	// QueueJobOutcomeFailed는 작업이 오류로 종료되었음을 의미한다.
	QueueJobOutcomeFailed
	// QueueJobOutcomeCancelled는 작업이 사용자 요청으로 중단되었음을 의미한다.
	QueueJobOutcomeCancelled
)

// AnalysisQueue represents a queue channel for sequential processing
type AnalysisQueue struct {
	Name      string
	Current   *AnalysisJob
	Pending   []*AnalysisJob
	Completed []*AnalysisJob // 완료된 작업 목록
	Failed    []*AnalysisJob // 실패한 작업 목록
	Cancelled []*AnalysisJob // 중단된 작업 목록
	IsRunning bool
}

// recordQueueJobByOutcome은 작업 결과에 따라 큐의 상태별 목록에 작업을 기록한다.
func recordQueueJobByOutcome(queue *AnalysisQueue, job *AnalysisJob, outcome QueueJobOutcome) {
	if queue == nil || job == nil {
		return
	}

	job.Outcome = outcome

	switch outcome {
	case QueueJobOutcomeCompleted:
		queue.Completed = append([]*AnalysisJob{job}, queue.Completed...)
	case QueueJobOutcomeCancelled:
		queue.Cancelled = append([]*AnalysisJob{job}, queue.Cancelled...)
	default:
		queue.Failed = append([]*AnalysisJob{job}, queue.Failed...)
	}
}

// addToQueue adds the current issue to a specific queue
func (a *App) addToQueue(channelIndex int) {
	ch := a.channels[channelIndex]

	if ch.CurrentDoc == nil || ch.CurrentMDPath == "" {
		dialog.ShowError(fmt.Errorf("먼저 이슈를 분석해주세요"), a.mainWindow)
		return
	}

	projectPath := ch.ProjectPathEntry.Text
	if projectPath == "" {
		dialog.ShowError(fmt.Errorf("프로젝트 경로를 입력해주세요"), a.mainWindow)
		return
	}

	issueKey := ch.CurrentDoc.IssueKey

	// 전체 큐에서 동일 이슈 중복 체크
	for i := 0; i < 3; i++ {
		q := a.queues[i]
		if q.Current != nil && q.Current.IssueKey == issueKey {
			dialog.ShowInformation("알림", fmt.Sprintf("%s이(가) %s에서 이미 실행 중입니다.", issueKey, q.Name), a.mainWindow)
			return
		}
		for _, p := range q.Pending {
			if p.IssueKey == issueKey {
				dialog.ShowInformation("알림", fmt.Sprintf("%s이(가) %s에 이미 대기 중입니다.", issueKey, q.Name), a.mainWindow)
				return
			}
		}
	}

	job := &AnalysisJob{
		IssueKey:     issueKey,
		MDPath:       ch.CurrentMDPath,
		PlanPath:     strings.TrimSuffix(ch.CurrentMDPath, ".md") + "_plan.md",
		AnalysisPath: strings.TrimSuffix(ch.CurrentMDPath, ".md") + "_plan.md",
		ScriptPath:   strings.TrimSuffix(ch.CurrentMDPath, ".md") + "_plan_run.sh",
		Phase:        adapter.PhaseAnalyze,
		ChannelIndex: channelIndex,
	}

	queue := a.queues[channelIndex]
	queue.Pending = append(queue.Pending, job)
	ch.QueueList.Refresh()

	ch.StatusLabel.SetText(fmt.Sprintf("%s에 %s 추가됨 (대기: %d)", queue.Name, job.IssueKey, len(queue.Pending)))

	// If not running, start processing
	if !queue.IsRunning {
		go a.processQueue(channelIndex)
	}
}

// stopQueueCurrent stops the current running job in a queue
func (a *App) stopQueueCurrent(channelIndex int) {
	queue := a.queues[channelIndex]
	ch := a.channels[channelIndex]
	stopped := 0

	// V2 실행 중인 작업 취소
	for _, task := range a.markCancelRunningTasks(channelIndex) {
		killRunningTask(task)
		a.markRunningTaskCancelledInDB(task)
		stopped++
	}

	if queue.Current == nil {
		if stopped > 0 {
			ch.StatusLabel.SetText(fmt.Sprintf("채널 %d 중지 요청됨 (%d개 실행)", channelIndex+1, stopped))
		}
		return
	}

	// 현재 작업에 중단 요청 상태를 기록한다.
	queue.Current.CancelRequested = true

	// PID 기반 종료를 우선하고, 실패 시에만 pkill -f fallback 사용
	if queue.Current.PID > 0 {
		exec.Command("kill", "-TERM", strconv.Itoa(queue.Current.PID)).Run()
	} else if queue.Current.ScriptPath != "" {
		cleaned := filepath.Clean(queue.Current.ScriptPath)
		if filepath.IsAbs(cleaned) {
			exec.Command("pkill", "-f", cleaned).Run()
		}
	}

	stopped++
	ch.StatusLabel.SetText(fmt.Sprintf("%s의 %s 중지 요청됨 (총 %d개)", queue.Name, queue.Current.IssueKey, stopped))
	ch.QueueList.Refresh()
}

// processQueue processes jobs in a queue sequentially
func (a *App) processQueue(channelIndex int) {
	queue := a.queues[channelIndex]
	ch := a.channels[channelIndex]

	for len(queue.Pending) > 0 {
		if queue.IsRunning {
			return // Already processing
		}

		// Get next job
		job := queue.Pending[0]
		queue.Pending = queue.Pending[1:]
		queue.Current = job
		queue.IsRunning = true
		ch.QueueList.Refresh()

		fmt.Printf("[Queue] %s: Phase 1 시작 - %s\n", queue.Name, job.IssueKey)
		ch.StatusLabel.SetText(fmt.Sprintf("%s: %s Phase 1 시작", queue.Name, job.IssueKey))

		outcome := a.executePhase1(channelIndex, job)

		// 실행 결과에 따라 상태별 목록에 기록한다.
		recordQueueJobByOutcome(queue, job, outcome)
		queue.Current = nil
		queue.IsRunning = false
		ch.QueueList.Refresh()

		switch outcome {
		case QueueJobOutcomeCompleted:
			fmt.Printf("[Queue] %s: Phase 1 완료 - %s\n", queue.Name, job.IssueKey)
		case QueueJobOutcomeCancelled:
			fmt.Printf("[Queue] %s: Phase 1 중단 - %s\n", queue.Name, job.IssueKey)
		default:
			fmt.Printf("[Queue] %s: Phase 1 실패 - %s\n", queue.Name, job.IssueKey)
		}
	}
}

// executePhase1은 Phase 1: 읽기 전용 분석을 실행한다.
func (a *App) executePhase1(channelIndex int, job *AnalysisJob) QueueJobOutcome {
	ch := a.channels[channelIndex]

	// 기존 plan 파일 삭제
	os.Remove(job.PlanPath)

	prompt := adapter.BuildAnalysisPlanPrompt(job.IssueKey, job.MDPath)
	projectPath := strings.TrimSpace(ch.ProjectPathEntry.Text)
	result, err := a.claudeAdapter.AnalyzeAndGeneratePlan(job.MDPath, prompt, projectPath)
	if err != nil {
		fmt.Printf("[Queue] %s: 오류 - %s: %v\n", a.queues[channelIndex].Name, job.IssueKey, err)
		ch.StatusLabel.SetText(fmt.Sprintf("오류: %s - %v", job.IssueKey, err))
		return QueueJobOutcomeFailed
	}

	job.PID = result.PID
	job.PlanPath = result.PlanPath
	job.AnalysisPath = result.PlanPath
	job.ScriptPath = result.ScriptPath
	job.LogPath = result.LogPath
	job.Phase = adapter.PhaseAnalyze

	// 채널별 상태 업데이트
	ch.CurrentAnalysisPath = result.PlanPath
	ch.CurrentPlanPath = result.PlanPath
	ch.CurrentScriptPath = result.ScriptPath

	ch.QueueList.Refresh()

	// Wait for completion
	return a.waitForJobCompletion(channelIndex, job)
}

// waitForJobCompletion waits for a job to complete while displaying progress
func (a *App) waitForJobCompletion(channelIndex int, job *AnalysisJob) QueueJobOutcome {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	var lastLogSize int64

	ch := a.channels[channelIndex]
	queueName := a.queues[channelIndex].Name

	for range ticker.C {
		elapsed := time.Since(startTime).Round(time.Second)
		elapsedStr := fmt.Sprintf("%dm %ds", int(elapsed.Minutes()), int(elapsed.Seconds())%60)

		// Check if process is still running
		checkCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", job.PID))
		if err := checkCmd.Run(); err != nil {
			// Process finished
			time.Sleep(500 * time.Millisecond)

			job.StartTime = elapsedStr

			// 사용자 중단 요청된 작업은 중단 상태로 처리한다.
			if job.CancelRequested {
				fmt.Printf("[Queue] %s: %s 중단됨 (%s)\n", queueName, job.IssueKey, elapsedStr)
				ch.StatusLabel.SetText(fmt.Sprintf("⏹ %s 중단됨 (%s)", job.IssueKey, elapsedStr))
				return QueueJobOutcomeCancelled
			}

			// 결과 파일 로드
			resultPath := job.AnalysisPath
			if job.PlanPath != "" {
				resultPath = job.PlanPath
			}
			content, readErr := os.ReadFile(resultPath)
			if readErr != nil {
				fmt.Printf("[Queue] %s: %s 실패 - 결과 파일 읽기 오류: %v\n", queueName, job.IssueKey, readErr)
				ch.StatusLabel.SetText(fmt.Sprintf("❌ %s 실패 (결과 파일 읽기 오류)", job.IssueKey))
				return QueueJobOutcomeFailed
			}
			ch.AnalysisText.SetText(string(content))
			ch.CopyAnalysisBtn.Enable()

			if job.PlanPath != "" {
				ch.CurrentPlanPath = job.PlanPath
			}

			// 완료 작업 목록에 추가
			a.mu.Lock()
			a.completedJobs = append([]*AnalysisJob{job}, a.completedJobs...)
			a.mu.Unlock()
			if a.historyList != nil {
				a.historyList.Refresh()
			}

			fmt.Printf("[Queue] %s: %s 완료 (%s)\n", queueName, job.IssueKey, elapsedStr)
			ch.StatusLabel.SetText(fmt.Sprintf("✅ %s 완료 (%s)", job.IssueKey, elapsedStr))
			return QueueJobOutcomeCompleted
		}

		// 로그 파일 읽기 → 진행상황 UI에 표시
		status := "분석 중..."
		if job.LogPath != "" {
			logContent, err := os.ReadFile(job.LogPath)
			if err == nil {
				if int64(len(logContent)) != lastLogSize {
					lastLogSize = int64(len(logContent))
					fmt.Printf("[Queue] %s: %s (경과: %s, 로그: %d bytes)\n", queueName, job.IssueKey, elapsedStr, len(logContent))
				}

				logStr := string(logContent)
				if strings.Contains(logStr, "Building plan") {
					status = "Plan 파일 조립 중..."
				} else if strings.Contains(logStr, "Running Claude") {
					status = "Claude 실행 중..."
				} else if strings.Contains(logStr, "Phase 1") {
					status = "시작 중..."
				} else if strings.Contains(logStr, "Phase 2") {
					status = "실행 중..."
				}
			}
		}
		// 매 틱마다 경과시간 갱신
		if job.CancelRequested {
			status = "중지 요청됨..."
		}
		ch.StatusLabel.SetText(fmt.Sprintf("⏳ %s: %s (경과: %s)", job.IssueKey, status, elapsedStr))
	}

	return QueueJobOutcomeFailed
}

// onStopAllQueues stops all running and pending jobs in all queues
func (a *App) onStopAllQueues() {
	stoppedCount := 0
	for i := 0; i < 3; i++ {
		queue := a.queues[i]
		ch := a.channels[i]

		// V2 병렬 실행 작업 중지
		for _, task := range a.markCancelRunningTasks(i) {
			killRunningTask(task)
			a.markRunningTaskCancelledInDB(task)
			stoppedCount++
		}

		// Stop current job
		if queue.Current != nil {
			// 현재 실행 중 작업은 중단 요청 상태로 표시한다.
			queue.Current.CancelRequested = true
			cmd := exec.Command("pkill", "-f", queue.Current.ScriptPath)
			cmd.Run()
			stoppedCount++
		}

		// Clear pending jobs
		stoppedCount += len(queue.Pending)
		queue.Pending = []*AnalysisJob{}
		queue.IsRunning = false
		if ch.QueueList != nil {
			ch.QueueList.Refresh()
		}
		if ch.StatusLabel != nil {
			ch.StatusLabel.SetText("중지됨")
		}
	}

	a.statusLabel.SetText(fmt.Sprintf("전체 중지됨 (작업 %d개)", stoppedCount))
	dialog.ShowInformation("전체 중지", fmt.Sprintf("%d개 작업이 중지되었습니다.", stoppedCount), a.mainWindow)
}

// extractIssueKeyFromPath는 파일 경로에서 이슈 키를 추출한다.
func extractIssueKeyFromPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, "_plan.md")
	base = strings.TrimSuffix(base, "_analysis.md")
	base = strings.TrimSuffix(base, ".md")
	return base
}

// mapAdapterPhaseToStatePhase converts adapter.AnalysisPhase to state.ProcessPhase
func mapAdapterPhaseToStatePhase(adapterPhase adapter.AnalysisPhase) state.ProcessPhase {
	switch adapterPhase {
	case adapter.PhaseAnalyze:
		return state.PhaseAnalyzing
	default:
		return state.PhaseIdle
	}
}
