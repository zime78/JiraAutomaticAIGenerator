package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"

	"jira-ai-generator/internal/adapter"
)

// AnalysisJob represents a running analysis task
type AnalysisJob struct {
	IssueKey      string
	ScriptPath    string
	AnalysisPath  string
	PlanPath      string                // Phase 1 결과: _plan.md 경로
	ExecutionPath string                // Phase 2 결과: _execution.md 경로
	LogPath       string                // 로그 파일 경로
	MDPath        string
	StartTime     string
	PID           int
	Phase         adapter.AnalysisPhase // 현재 실행 단계
}

// AnalysisQueue represents a queue channel for sequential processing
type AnalysisQueue struct {
	Name      string
	Current   *AnalysisJob
	Pending   []*AnalysisJob
	IsRunning bool
}

// addToQueue adds the current issue to a specific queue
func (a *App) addToQueue(channelIndex int) {
	if a.currentDoc == nil || a.currentMDPath == "" {
		dialog.ShowError(fmt.Errorf("먼저 이슈를 분석해주세요"), a.mainWindow)
		return
	}

	projectPath := a.projectPathEntry.Text
	if projectPath == "" {
		dialog.ShowError(fmt.Errorf("프로젝트 경로를 입력해주세요"), a.mainWindow)
		return
	}

	issueKey := a.currentDoc.IssueKey

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
		MDPath:       a.currentMDPath,
		PlanPath:     strings.TrimSuffix(a.currentMDPath, ".md") + "_plan.md",
		AnalysisPath: strings.TrimSuffix(a.currentMDPath, ".md") + "_plan.md",
		ScriptPath:   strings.TrimSuffix(a.currentMDPath, ".md") + "_plan_run.sh",
		Phase:        adapter.PhaseAnalyze,
	}

	queue := a.queues[channelIndex]
	queue.Pending = append(queue.Pending, job)
	a.queueLists[channelIndex].Refresh()

	a.statusLabel.SetText(fmt.Sprintf("%s에 %s 추가됨 (대기: %d)", queue.Name, job.IssueKey, len(queue.Pending)))

	// If not running, start processing
	if !queue.IsRunning {
		go a.processQueue(channelIndex)
	}
}

// stopQueueCurrent stops the current running job in a queue
func (a *App) stopQueueCurrent(channelIndex int) {
	queue := a.queues[channelIndex]
	if queue.Current == nil {
		return
	}

	// Kill the process
	cmd := exec.Command("pkill", "-f", queue.Current.ScriptPath)
	cmd.Run()

	a.statusLabel.SetText(fmt.Sprintf("%s의 %s 중지됨", queue.Name, queue.Current.IssueKey))
	queue.Current = nil
	queue.IsRunning = false
	a.queueLists[channelIndex].Refresh()

	// Process next in queue
	if len(queue.Pending) > 0 {
		go a.processQueue(channelIndex)
	}
}

// processQueue processes jobs in a queue sequentially
func (a *App) processQueue(channelIndex int) {
	queue := a.queues[channelIndex]

	for len(queue.Pending) > 0 {
		if queue.IsRunning {
			return // Already processing
		}

		// Get next job
		job := queue.Pending[0]
		queue.Pending = queue.Pending[1:]
		queue.Current = job
		queue.IsRunning = true
		a.queueLists[channelIndex].Refresh()

		fmt.Printf("[Queue] %s: 시작 - %s\n", queue.Name, job.IssueKey)
		a.statusLabel.SetText(fmt.Sprintf("%s: %s 분석 시작", queue.Name, job.IssueKey))

		// 기존 plan 파일 삭제
		os.Remove(job.PlanPath)

		// Phase 1: 분석 및 계획 생성
		prompt := adapter.BuildAnalysisPlanPrompt(job.IssueKey, job.MDPath)
		result, err := a.claudeAdapter.AnalyzeAndGeneratePlan(job.MDPath, prompt)
		if err != nil {
			fmt.Printf("[Queue] %s: 오류 - %s: %v\n", queue.Name, job.IssueKey, err)
			queue.Current = nil
			queue.IsRunning = false
			continue
		}

		job.PID = result.PID
		job.PlanPath = result.PlanPath
		job.AnalysisPath = result.PlanPath
		job.ScriptPath = result.ScriptPath
		job.LogPath = result.LogPath
		job.Phase = adapter.PhaseAnalyze

		// App 상태도 업데이트 (새로고침 버튼 등에서 사용)
		a.currentAnalysisPath = result.PlanPath
		a.currentPlanPath = result.PlanPath
		a.currentScriptPath = result.ScriptPath

		a.queueLists[channelIndex].Refresh()

		// Wait for completion
		a.waitForJobCompletion(channelIndex, job)

		queue.Current = nil
		queue.IsRunning = false
		a.queueLists[channelIndex].Refresh()

		fmt.Printf("[Queue] %s: 완료 - %s\n", queue.Name, job.IssueKey)
	}
}

// waitForJobCompletion waits for a job to complete while displaying progress
func (a *App) waitForJobCompletion(channelIndex int, job *AnalysisJob) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	var lastLogSize int64

	phaseLabel := "Phase 1"
	if job.Phase == adapter.PhaseExecute {
		phaseLabel = "Phase 2"
	}

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

			// plan 파일 우선 로드
			resultPath := job.AnalysisPath
			if job.PlanPath != "" {
				resultPath = job.PlanPath
			}
			if content, err := os.ReadFile(resultPath); err == nil {
				a.analysisText.SetText(string(content))
				a.copyAnalysisBtn.Enable()
			}

			// Phase 1 완료 시 "계획 실행" 버튼 활성화
			if job.Phase == adapter.PhaseAnalyze && job.PlanPath != "" {
				a.currentPlanPath = job.PlanPath
				a.executePlanBtn.Enable()
			}

			// 완료 작업 목록에 추가
			a.completedJobs = append([]*AnalysisJob{job}, a.completedJobs...)
			if a.historyList != nil {
				a.historyList.Refresh()
			}

			fmt.Printf("[Queue] %s: %s %s 완료 (%s)\n", queueName, job.IssueKey, phaseLabel, elapsedStr)
			a.statusLabel.SetText(fmt.Sprintf("✅ %s: %s %s 완료 (%s)", queueName, job.IssueKey, phaseLabel, elapsedStr))
			return
		}

		// 로그 파일 읽기 → 진행상황 UI에 표시
		status := "분석 중..."
		if job.LogPath != "" {
			logContent, err := os.ReadFile(job.LogPath)
			if err == nil {
				if int64(len(logContent)) != lastLogSize {
					lastLogSize = int64(len(logContent))
					fmt.Printf("[Queue] %s %s: %s (경과: %s, 로그: %d bytes)\n", queueName, phaseLabel, job.IssueKey, elapsedStr, len(logContent))
				}

				logStr := string(logContent)
				if strings.Contains(logStr, "Building plan") {
					status = "Plan 파일 조립 중..."
				} else if strings.Contains(logStr, "Running Claude") {
					status = "Claude 실행 중..."
				} else if strings.Contains(logStr, "Phase 1") {
					status = "시작 중..."
				}
			}
		}
		// 매 틱마다 경과시간 갱신
		a.statusLabel.SetText(fmt.Sprintf("⏳ %s %s: %s %s (경과: %s)", queueName, phaseLabel, job.IssueKey, status, elapsedStr))
	}
}

// onStopAllQueues stops all running and pending jobs in all queues
func (a *App) onStopAllQueues() {
	stoppedCount := 0
	for i := 0; i < 3; i++ {
		queue := a.queues[i]

		// Stop current job
		if queue.Current != nil {
			cmd := exec.Command("pkill", "-f", queue.Current.ScriptPath)
			cmd.Run()
			queue.Current = nil
			stoppedCount++
		}

		// Clear pending jobs
		stoppedCount += len(queue.Pending)
		queue.Pending = []*AnalysisJob{}
		queue.IsRunning = false
		a.queueLists[i].Refresh()
	}

	a.statusLabel.SetText(fmt.Sprintf("전체 중지됨 (작업 %d개)", stoppedCount))
	dialog.ShowInformation("전체 중지", fmt.Sprintf("%d개 작업이 중지되었습니다.", stoppedCount), a.mainWindow)
}
