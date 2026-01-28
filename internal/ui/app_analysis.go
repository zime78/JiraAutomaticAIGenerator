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

// onAnalyze는 Phase 1: 읽기 전용 분석을 실행하여 _plan.md를 생성한다.
func (a *App) onAnalyze() {
	if a.currentDoc == nil || a.currentMDPath == "" {
		dialog.ShowError(fmt.Errorf("먼저 Jira 이슈를 분석해주세요"), a.mainWindow)
		return
	}

	// 프로젝트 경로 설정
	projectPath := strings.TrimSpace(a.projectPathEntry.Text)
	if projectPath == "" {
		dialog.ShowError(fmt.Errorf("프로젝트 경로를 입력해주세요"), a.mainWindow)
		return
	}
	a.claudeAdapter.SetWorkDir(projectPath)

	a.stopAnalysisBtn.Enable()
	a.executePlanBtn.Disable()
	a.statusLabel.SetText(fmt.Sprintf("Phase 1: Claude에 분석 요청 중... (프로젝트: %s)", projectPath))
	a.analysisText.SetText("⏳ Phase 1: 분석 및 계획 생성 요청 중...")

	// 중지 기능용 스크립트 경로 설정
	a.currentScriptPath = strings.TrimSuffix(a.currentMDPath, ".md") + "_plan_run.sh"

	// 기존 plan 파일 삭제
	oldPlanPath := strings.TrimSuffix(a.currentMDPath, ".md") + "_plan.md"
	os.Remove(oldPlanPath)

	// Phase 1: 구조화된 plan 프롬프트 사용
	prompt := adapter.BuildAnalysisPlanPrompt(a.currentDoc.IssueKey, a.currentMDPath)
	issueKey := a.currentDoc.IssueKey

	a.claudeAdapter.SendPlanToClaudeAsync(a.currentMDPath, prompt, func(result *adapter.PlanResult, err error) {
		if err != nil {
			a.statusLabel.SetText(fmt.Sprintf("Claude 오류: %v", err))
			a.analysisText.SetText(fmt.Sprintf("분석 실패: %v", err))
			dialog.ShowError(fmt.Errorf("Claude 분석 실패: %w", err), a.mainWindow)
			a.stopAnalysisBtn.Disable()
		} else {
			// 실행 중 작업 목록에 추가
			job := &AnalysisJob{
				IssueKey:     issueKey,
				ScriptPath:   result.ScriptPath,
				PlanPath:     result.PlanPath,
				AnalysisPath: result.PlanPath,
				StartTime:    fmt.Sprintf("%s", filepath.Base(result.PlanPath)),
				PID:          result.PID,
				Phase:        adapter.PhaseAnalyze,
			}
			a.runningJobs = append(a.runningJobs, job)
			if a.jobsList != nil {
				a.jobsList.Refresh()
			}

			// 상태 업데이트
			a.currentAnalysisPath = result.PlanPath
			a.currentPlanPath = result.PlanPath
			a.currentScriptPath = result.ScriptPath

			a.analysisText.SetText(fmt.Sprintf("⏳ Phase 1: 분석 진행 중...\n\n이슈: %s\nPlan 파일: %s\nPID: %d\n\n로그를 모니터링 중입니다...", issueKey, result.PlanPath, result.PID))
			a.statusLabel.SetText(fmt.Sprintf("Phase 1 시작됨 [%s] PID: %d", issueKey, result.PID))

			// 로그 모니터링 시작 (Phase 1)
			go a.monitorAnalysisLog(issueKey, result.LogPath, result.PlanPath, result.PID, adapter.PhaseAnalyze)

			dialog.ShowInformation("Phase 1 시작됨", fmt.Sprintf("Claude 분석이 백그라운드에서 시작되었습니다.\n\n이슈: %s\nPID: %d\n\n완료 후 '계획 실행' 버튼으로 Phase 2를 실행할 수 있습니다.", issueKey, result.PID), a.mainWindow)
			// AI 분석 탭으로 전환
			a.tabs.SelectIndex(1)
		}
	})
}

// onExecutePlan은 Phase 2: plan 파일을 Claude Code에 전달하여 실제 코드 수정을 실행한다.
func (a *App) onExecutePlan() {
	if a.currentPlanPath == "" {
		dialog.ShowError(fmt.Errorf("먼저 Phase 1 분석을 실행해주세요"), a.mainWindow)
		return
	}

	// plan 파일 존재 확인
	if _, err := os.Stat(a.currentPlanPath); os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf("plan 파일이 존재하지 않습니다: %s", a.currentPlanPath), a.mainWindow)
		return
	}

	projectPath := strings.TrimSpace(a.projectPathEntry.Text)
	if projectPath == "" {
		dialog.ShowError(fmt.Errorf("프로젝트 경로를 입력해주세요"), a.mainWindow)
		return
	}
	a.claudeAdapter.SetWorkDir(projectPath)

	a.executePlanBtn.Disable()
	a.stopAnalysisBtn.Enable()
	a.statusLabel.SetText(fmt.Sprintf("Phase 2: 계획 실행 중... (프로젝트: %s)", projectPath))
	a.analysisText.SetText("⏳ Phase 2: 계획 실행 중...")

	issueKey := ""
	if a.currentDoc != nil {
		issueKey = a.currentDoc.IssueKey
	}

	go func() {
		result, err := a.claudeAdapter.ExecutePlan(a.currentPlanPath)
		if err != nil {
			a.statusLabel.SetText(fmt.Sprintf("Phase 2 오류: %v", err))
			a.analysisText.SetText(fmt.Sprintf("계획 실행 실패: %v", err))
			dialog.ShowError(fmt.Errorf("계획 실행 실패: %w", err), a.mainWindow)
			a.stopAnalysisBtn.Disable()
			return
		}

		// 상태 업데이트
		a.currentAnalysisPath = result.OutputPath
		a.currentScriptPath = result.ScriptPath
		logPath := strings.TrimSuffix(result.OutputPath, "_execution.md") + "_exec_log.txt"

		a.analysisText.SetText(fmt.Sprintf("⏳ Phase 2: 계획 실행 중...\n\n이슈: %s\n실행 결과: %s\nPID: %d\n\n로그를 모니터링 중입니다...", issueKey, result.OutputPath, result.PID))
		a.statusLabel.SetText(fmt.Sprintf("Phase 2 시작됨 [%s] PID: %d", issueKey, result.PID))

		// 로그 모니터링 시작 (Phase 2)
		go a.monitorAnalysisLog(issueKey, logPath, result.OutputPath, result.PID, adapter.PhaseExecute)
	}()
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

	// 스크립트 관련 프로세스 종료
	cmd := exec.Command("pkill", "-f", a.currentScriptPath)
	cmd.Run()

	// 실행 중 작업에서 제거
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
		mdPath = strings.TrimSuffix(job.AnalysisPath, "_plan.md") + ".md"
		if _, err := os.Stat(mdPath); os.IsNotExist(err) {
			mdPath = strings.TrimSuffix(job.AnalysisPath, "_analysis.md") + ".md"
		}
	}
	if mdContent, err := os.ReadFile(mdPath); err == nil {
		a.resultText.SetText(string(mdContent))
		a.copyBtn.Enable()
		a.currentMDPath = mdPath
	}

	// AI 분석 결과 로드 (plan 파일 우선)
	analysisPath := job.AnalysisPath
	if job.PlanPath != "" {
		analysisPath = job.PlanPath
	}

	content, err := os.ReadFile(analysisPath)
	if err != nil {
		a.analysisText.SetText(fmt.Sprintf("⏳ 분석 진행 중...\n\n이슈: %s\nPID: %d\n\n아직 결과가 생성되지 않았습니다.", job.IssueKey, job.PID))
		return
	}
	a.analysisText.SetText(string(content))
	a.currentAnalysisPath = analysisPath
	a.currentScriptPath = job.ScriptPath
	a.copyAnalysisBtn.Enable()

	// plan 파일이 있으면 "계획 실행" 버튼 활성화
	if job.PlanPath != "" {
		a.currentPlanPath = job.PlanPath
		a.executePlanBtn.Enable()
	}

	a.statusLabel.SetText(fmt.Sprintf("결과 로드됨: %s", job.IssueKey))
}

func (a *App) stopJob(job *AnalysisJob) {
	// 스크립트 관련 프로세스 종료
	cmd := exec.Command("pkill", "-f", job.ScriptPath)
	cmd.Run()

	// 실행 중 작업에서 제거
	for i, j := range a.runningJobs {
		if j.ScriptPath == job.ScriptPath {
			a.runningJobs = append(a.runningJobs[:i], a.runningJobs[i+1:]...)
			break
		}
	}
	a.jobsList.Refresh()
	a.statusLabel.SetText(fmt.Sprintf("중지됨: %s", job.IssueKey))
}

// monitorAnalysisLog는 백그라운드 프로세스의 로그를 모니터링한다.
// phase 파라미터에 따라 Phase 1 완료 시 "계획 실행" 버튼을 활성화하고,
// Phase 2 완료 시 실행 결과를 표시한다.
func (a *App) monitorAnalysisLog(issueKey, logPath, outputPath string, pid int, phase adapter.AnalysisPhase) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var lastLogSize int64 = 0
	startTime := time.Now()

	phaseLabel := "Phase 1"
	if phase == adapter.PhaseExecute {
		phaseLabel = "Phase 2"
	}

	for {
		select {
		case <-ticker.C:
			// 프로세스 실행 여부 확인
			checkCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid))
			if err := checkCmd.Run(); err != nil {
				// 프로세스 종료 → 최종 결과 로드
				time.Sleep(500 * time.Millisecond)
				if content, err := os.ReadFile(outputPath); err == nil {
					a.analysisText.SetText(string(content))
					a.copyAnalysisBtn.Enable()

					if phase == adapter.PhaseAnalyze {
						// Phase 1 완료: "계획 실행" 버튼 활성화
						a.currentPlanPath = outputPath
						a.executePlanBtn.Enable()
						a.statusLabel.SetText(fmt.Sprintf("✅ Phase 1 완료: %s — '계획 실행' 버튼으로 Phase 2를 실행하세요", issueKey))
					} else {
						// Phase 2 완료
						a.statusLabel.SetText(fmt.Sprintf("✅ Phase 2 완료: %s — 코드 수정이 적용되었습니다", issueKey))
					}

					// 실행 중 작업에서 제거
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

			// 로그 파일 읽기
			logContent, err := os.ReadFile(logPath)
			if err != nil {
				continue
			}

			// 로그 변경 감지
			if int64(len(logContent)) != lastLogSize {
				lastLogSize = int64(len(logContent))
				elapsed := time.Since(startTime).Round(time.Second)

				logStr := string(logContent)
				status := "시작 중..."
				if strings.Contains(logStr, "Running Claude") {
					status = "Claude 실행 중..."
				}

				fmt.Printf("[Monitor] %s %s: %s (경과: %s)\n", phaseLabel, issueKey, status, elapsed)
				a.statusLabel.SetText(fmt.Sprintf("⏳ %s %s: %s (경과: %s)", phaseLabel, issueKey, status, elapsed))
			}
		}
	}
}

// loadPreviousAnalysis는 output 폴더에서 기존 분석 결과를 스캔한다.
// _plan.md 파일을 우선 사용하고, 없으면 _analysis.md를 사용한다.
func (a *App) loadPreviousAnalysis() {
	outputDir := a.config.Output.Dir
	if outputDir == "" {
		outputDir = "output"
	}

	// output 디렉토리 존재 확인
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return
	}

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

		// plan 파일 우선 검색, 없으면 analysis 파일 사용
		planPath := filepath.Join(outputDir, issueKey, issueKey+"_plan.md")
		analysisPath := filepath.Join(outputDir, issueKey, issueKey+"_analysis.md")

		var resultPath string
		var resultPlanPath string

		if info, err := os.Stat(planPath); err == nil {
			resultPath = planPath
			resultPlanPath = planPath
			_ = info
		} else if info, err := os.Stat(analysisPath); err == nil {
			resultPath = analysisPath
			_ = info
		} else {
			continue
		}

		// 수정 시간 가져오기
		info, err := os.Stat(resultPath)
		if err != nil {
			continue
		}

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
			AnalysisPath: resultPath,
			PlanPath:     resultPlanPath,
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
