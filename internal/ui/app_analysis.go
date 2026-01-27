package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"

	"jira-ai-generator/internal/adapter"
)

func (a *App) onAnalyze() {
	if a.currentDoc == nil || a.currentMDPath == "" {
		dialog.ShowError(fmt.Errorf("먼저 Jira 이슈를 분석해주세요"), a.mainWindow)
		return
	}

	// Set project path for Claude
	projectPath := strings.TrimSpace(a.projectPathEntry.Text)
	if projectPath == "" {
		dialog.ShowError(fmt.Errorf("프로젝트 경로를 입력해주세요"), a.mainWindow)
		return
	}
	a.claudeAdapter.SetWorkDir(projectPath)

	a.stopAnalysisBtn.Enable()
	a.statusLabel.SetText(fmt.Sprintf("Claude에 분석 요청 중... (프로젝트: %s)", projectPath))
	a.analysisText.SetText("⏳ 분석 요청 중...")

	// Set script path for stop functionality
	a.currentScriptPath = strings.TrimSuffix(a.currentMDPath, ".md") + "_run.sh"

	// Delete old analysis file to avoid confusion
	oldAnalysisPath := strings.TrimSuffix(a.currentMDPath, ".md") + "_analysis.md"
	os.Remove(oldAnalysisPath)

	prompt := adapter.BuildAnalysisPrompt(a.currentDoc.IssueKey, a.currentMDPath)
	issueKey := a.currentDoc.IssueKey

	a.claudeAdapter.SendToClaudeAsync(a.currentMDPath, prompt, func(result *adapter.AnalysisResult, err error) {
		if err != nil {
			a.statusLabel.SetText(fmt.Sprintf("Claude 오류: %v", err))
			a.analysisText.SetText(fmt.Sprintf("분석 실패: %v", err))
			dialog.ShowError(fmt.Errorf("Claude 분석 실패: %w", err), a.mainWindow)
			a.stopAnalysisBtn.Disable()
		} else {
			// Add to running jobs
			job := &AnalysisJob{
				IssueKey:     issueKey,
				ScriptPath:   result.ScriptPath,
				AnalysisPath: result.OutputPath,
				StartTime:    fmt.Sprintf("%s", strings.Split(result.OutputPath, "/")[len(strings.Split(result.OutputPath, "/"))-1]),
				PID:          result.PID,
			}
			a.runningJobs = append(a.runningJobs, job)
			if a.jobsList != nil {
				a.jobsList.Refresh()
			}

			// Background process started - file will be created when complete
			a.currentAnalysisPath = result.OutputPath
			a.currentScriptPath = result.ScriptPath
			logPath := strings.TrimSuffix(result.OutputPath, "_analysis.md") + "_log.txt"

			a.analysisText.SetText(fmt.Sprintf("⏳ 분석 진행 중...\n\n이슈: %s\n결과 파일: %s\nPID: %d\n\n로그를 모니터링 중입니다...", issueKey, result.OutputPath, result.PID))
			a.statusLabel.SetText(fmt.Sprintf("분석 시작됨 [%s] PID: %d", issueKey, result.PID))

			// Start log monitoring goroutine
			go a.monitorAnalysisLog(issueKey, logPath, result.OutputPath, result.PID)

			dialog.ShowInformation("분석 시작됨", fmt.Sprintf("Claude 분석이 백그라운드에서 시작되었습니다.\n\n이슈: %s\nPID: %d\n\n진행 상황이 자동으로 표시됩니다.", issueKey, result.PID), a.mainWindow)
			// Switch to AI analysis tab
			a.tabs.SelectIndex(1)
		}
	})
}

func (a *App) onCopyAnalysis() {
	if a.analysisText.Text == "" {
		return
	}

	a.mainWindow.Clipboard().SetContent(a.analysisText.Text)
	dialog.ShowInformation("완료", "분석 결과가 복사되었습니다.", a.mainWindow)
}

func (a *App) onRefreshAnalysis() {
	if a.currentAnalysisPath == "" {
		dialog.ShowInformation("알림", "아직 분석 결과 파일이 없습니다.", a.mainWindow)
		return
	}

	content, err := os.ReadFile(a.currentAnalysisPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("파일 읽기 실패: %w", err), a.mainWindow)
		return
	}

	a.analysisText.SetText(string(content))
	a.copyAnalysisBtn.Enable()
	a.statusLabel.SetText(fmt.Sprintf("분석 결과 새로고침 완료: %s", a.currentAnalysisPath))
}

func (a *App) onStopAnalysis() {
	if a.currentScriptPath == "" {
		return
	}

	// Kill processes related to the script
	cmd := exec.Command("pkill", "-f", a.currentScriptPath)
	cmd.Run()

	// Remove from running jobs
	for i, job := range a.runningJobs {
		if job.ScriptPath == a.currentScriptPath {
			a.runningJobs = append(a.runningJobs[:i], a.runningJobs[i+1:]...)
			break
		}
	}
	if a.jobsList != nil {
		a.jobsList.Refresh()
	}

	a.stopAnalysisBtn.Disable()
	a.currentScriptPath = ""
	a.statusLabel.SetText("분석이 중지되었습니다.")
	a.analysisText.SetText("❌ 분석이 사용자에 의해 중지되었습니다.")
	dialog.ShowInformation("중지됨", "분석이 중지되었습니다.", a.mainWindow)
}

func (a *App) loadJobResult(job *AnalysisJob) {
	// 이슈 정보 마크다운 로드
	mdPath := job.MDPath
	if mdPath == "" {
		// MDPath가 없으면 AnalysisPath에서 파생
		mdPath = strings.TrimSuffix(job.AnalysisPath, "_analysis.md") + ".md"
	}
	if mdContent, err := os.ReadFile(mdPath); err == nil {
		a.resultText.SetText(string(mdContent))
		a.copyBtn.Enable()
		a.currentMDPath = mdPath
	}

	// AI 분석 결과 로드
	content, err := os.ReadFile(job.AnalysisPath)
	if err != nil {
		a.analysisText.SetText(fmt.Sprintf("⏳ 분석 진행 중...\n\n이슈: %s\nPID: %d\n\n아직 결과가 생성되지 않았습니다.", job.IssueKey, job.PID))
		return
	}
	a.analysisText.SetText(string(content))
	a.currentAnalysisPath = job.AnalysisPath
	a.currentScriptPath = job.ScriptPath
	a.copyAnalysisBtn.Enable()
	a.statusLabel.SetText(fmt.Sprintf("결과 로드됨: %s", job.IssueKey))
}

func (a *App) stopJob(job *AnalysisJob) {
	// Kill processes related to the script
	cmd := exec.Command("pkill", "-f", job.ScriptPath)
	cmd.Run()

	// Remove from running jobs
	for i, j := range a.runningJobs {
		if j.ScriptPath == job.ScriptPath {
			a.runningJobs = append(a.runningJobs[:i], a.runningJobs[i+1:]...)
			break
		}
	}
	a.jobsList.Refresh()
	a.statusLabel.SetText(fmt.Sprintf("중지됨: %s", job.IssueKey))
}

func (a *App) monitorAnalysisLog(issueKey, logPath, outputPath string, pid int) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var lastLogSize int64 = 0
	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			// Check if process is still running
			checkCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid))
			if err := checkCmd.Run(); err != nil {
				// Process finished - load final result
				time.Sleep(500 * time.Millisecond) // Wait for file write
				if content, err := os.ReadFile(outputPath); err == nil {
					a.analysisText.SetText(string(content))
					a.copyAnalysisBtn.Enable()
					a.statusLabel.SetText(fmt.Sprintf("✅ 분석 완료: %s", issueKey))

					// Remove from running jobs
					for i, job := range a.runningJobs {
						if job.PID == pid {
							a.runningJobs = append(a.runningJobs[:i], a.runningJobs[i+1:]...)
							break
						}
					}
					if a.jobsList != nil {
						a.jobsList.Refresh()
					}
					a.stopAnalysisBtn.Disable()
				}
				return
			}

			// Read log file for progress
			logContent, err := os.ReadFile(logPath)
			if err != nil {
				continue
			}

			// Check if log has changed
			if int64(len(logContent)) != lastLogSize {
				lastLogSize = int64(len(logContent))
				elapsed := time.Since(startTime).Round(time.Second)

				// Parse log for status
				logStr := string(logContent)
				status := "시작 중..."
				if strings.Contains(logStr, "Running Claude") {
					status = "Claude 실행 중..."
				}

				fmt.Printf("[Monitor] %s: %s (경과: %s)\n", issueKey, status, elapsed)
				a.statusLabel.SetText(fmt.Sprintf("⏳ %s: %s (경과: %s)", issueKey, status, elapsed))
			}
		}
	}
}

// loadPreviousAnalysis scans output folder for existing analysis results
func (a *App) loadPreviousAnalysis() {
	outputDir := a.config.Output.Dir
	if outputDir == "" {
		outputDir = "output"
	}

	// Check if output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return
	}

	// Scan for analysis files
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		fmt.Printf("[History] Failed to read output dir: %v\n", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		issueKey := entry.Name()
		analysisPath := filepath.Join(outputDir, issueKey, issueKey+"_analysis.md")

		// Check if analysis file exists
		info, err := os.Stat(analysisPath)
		if err != nil {
			continue
		}

		// Get modification time
		modTime := info.ModTime()
		elapsed := time.Since(modTime)
		var timeStr string
		if elapsed.Hours() < 1 {
			timeStr = fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
		} else if elapsed.Hours() < 24 {
			timeStr = fmt.Sprintf("%dh ago", int(elapsed.Hours()))
		} else {
			timeStr = modTime.Format("01/02 15:04")
		}

		mdPath := filepath.Join(outputDir, issueKey, issueKey+".md")
		job := &AnalysisJob{
			IssueKey:     issueKey,
			AnalysisPath: analysisPath,
			MDPath:       mdPath,
			StartTime:    timeStr,
		}
		a.completedJobs = append(a.completedJobs, job)
	}

	if a.historyList != nil {
		a.historyList.Refresh()
	}

	fmt.Printf("[History] Loaded %d previous analysis results\n", len(a.completedJobs))
}
