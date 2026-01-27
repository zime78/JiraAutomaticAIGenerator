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
	IssueKey     string
	ScriptPath   string
	AnalysisPath string
	MDPath       string
	StartTime    string
	PID          int
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

	job := &AnalysisJob{
		IssueKey:     a.currentDoc.IssueKey,
		MDPath:       a.currentMDPath,
		AnalysisPath: strings.TrimSuffix(a.currentMDPath, ".md") + "_analysis.md",
		ScriptPath:   strings.TrimSuffix(a.currentMDPath, ".md") + "_run.sh",
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

		// Delete old analysis file
		os.Remove(job.AnalysisPath)

		// Start analysis
		prompt := adapter.BuildAnalysisPrompt(job.IssueKey, job.MDPath)
		result, err := a.claudeAdapter.AnalyzeIssue(job.MDPath, prompt)
		if err != nil {
			fmt.Printf("[Queue] %s: 오류 - %s: %v\n", queue.Name, job.IssueKey, err)
			queue.Current = nil
			queue.IsRunning = false
			continue
		}

		job.PID = result.PID
		job.AnalysisPath = result.OutputPath
		job.ScriptPath = result.ScriptPath
		a.queueLists[channelIndex].Refresh()

		// Wait for completion
		a.waitForJobCompletion(channelIndex, job)

		queue.Current = nil
		queue.IsRunning = false
		a.queueLists[channelIndex].Refresh()

		fmt.Printf("[Queue] %s: 완료 - %s\n", queue.Name, job.IssueKey)
	}
}

// waitForJobCompletion waits for a job to complete
func (a *App) waitForJobCompletion(channelIndex int, job *AnalysisJob) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for range ticker.C {
		// Check if process is still running
		checkCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", job.PID))
		if err := checkCmd.Run(); err != nil {
			// Process finished
			time.Sleep(500 * time.Millisecond)

			elapsed := time.Since(startTime).Round(time.Second)
			job.StartTime = fmt.Sprintf("%dm %ds", int(elapsed.Minutes()), int(elapsed.Seconds())%60)

			if content, err := os.ReadFile(job.AnalysisPath); err == nil {
				a.analysisText.SetText(string(content))
				a.copyAnalysisBtn.Enable()
			}

			// Add to completed jobs
			a.completedJobs = append([]*AnalysisJob{job}, a.completedJobs...)
			if a.historyList != nil {
				a.historyList.Refresh()
			}

			a.statusLabel.SetText(fmt.Sprintf("✅ %s: %s 완료 (%s)", a.queues[channelIndex].Name, job.IssueKey, job.StartTime))
			return
		}
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
