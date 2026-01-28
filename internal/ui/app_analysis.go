package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"

	"jira-ai-generator/internal/adapter"
)

// onExecuteChannelPlan은 해당 채널의 큐에 Phase 2 작업을 추가한다.
func (a *App) onExecuteChannelPlan(channelIndex int) {
	ch := a.channels[channelIndex]
	planPath := ch.CurrentPlanPath

	if planPath == "" {
		dialog.ShowError(fmt.Errorf("먼저 Phase 1 분석을 실행해주세요"), a.mainWindow)
		return
	}

	// plan 파일 존재 확인
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf("plan 파일이 존재하지 않습니다: %s", planPath), a.mainWindow)
		return
	}

	issueKey := extractIssueKeyFromPath(planPath)

	job := &AnalysisJob{
		IssueKey:     issueKey,
		PlanPath:     planPath,
		AnalysisPath: planPath,
		MDPath:       strings.TrimSuffix(planPath, "_plan.md") + ".md",
		Phase:        adapter.PhaseExecute,
		ChannelIndex: channelIndex,
	}

	queue := a.queues[channelIndex]
	queue.Pending = append(queue.Pending, job)
	ch.QueueList.Refresh()
	ch.ExecutePlanBtn.Disable()

	ch.StatusLabel.SetText(fmt.Sprintf("Phase 2 대기열에 추가됨: %s", issueKey))

	if !queue.IsRunning {
		go a.processQueue(channelIndex)
	}
}

// onCopyChannelAnalysis는 해당 채널의 분석 텍스트를 클립보드에 복사한다.
func (a *App) onCopyChannelAnalysis(channelIndex int) {
	ch := a.channels[channelIndex]
	if ch.AnalysisText.Text == "" {
		return
	}

	a.mainWindow.Clipboard().SetContent(ch.AnalysisText.Text)
	dialog.ShowInformation("완료", "분석 결과가 복사되었습니다.", a.mainWindow)
}

// onRefreshChannelAnalysis는 해당 채널의 분석 결과 파일을 다시 읽는다.
func (a *App) onRefreshChannelAnalysis(channelIndex int) {
	ch := a.channels[channelIndex]
	path := ch.CurrentAnalysisPath

	if path == "" {
		dialog.ShowInformation("알림", "아직 분석 결과 파일이 없습니다.", a.mainWindow)
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		dialog.ShowError(fmt.Errorf("파일 읽기 실패: %w", err), a.mainWindow)
		return
	}

	ch.AnalysisText.SetText(string(content))
	ch.CopyAnalysisBtn.Enable()
	ch.StatusLabel.SetText(fmt.Sprintf("분석 결과 새로고침 완료: %s", path))
}

// loadJobResultToChannel은 완료된 작업의 결과를 해당 채널에 로드한다.
func (a *App) loadJobResultToChannel(job *AnalysisJob) {
	channelIndex := job.ChannelIndex
	if channelIndex < 0 || channelIndex >= 3 {
		channelIndex = 0
	}
	ch := a.channels[channelIndex]

	// 해당 채널 탭으로 전환
	if a.tabs != nil {
		a.tabs.SelectIndex(channelIndex)
	}

	// 이슈 마크다운 로드
	mdPath := job.MDPath
	if mdPath == "" {
		mdPath = strings.TrimSuffix(job.AnalysisPath, "_plan.md") + ".md"
		if _, err := os.Stat(mdPath); os.IsNotExist(err) {
			mdPath = strings.TrimSuffix(job.AnalysisPath, "_execution.md") + ".md"
			if _, err := os.Stat(mdPath); os.IsNotExist(err) {
				mdPath = strings.TrimSuffix(job.AnalysisPath, "_analysis.md") + ".md"
			}
		}
	}
	if mdContent, err := os.ReadFile(mdPath); err == nil {
		ch.ResultText.SetText(string(mdContent))
		ch.CopyResultBtn.Enable()
		ch.CurrentMDPath = mdPath
	}

	// AI 분석 결과 로드 (plan 파일 우선)
	analysisPath := job.AnalysisPath
	if job.PlanPath != "" {
		analysisPath = job.PlanPath
	}

	content, err := os.ReadFile(analysisPath)
	if err != nil {
		ch.AnalysisText.SetText(fmt.Sprintf("분석 진행 중...\n\n이슈: %s\nPID: %d\n\n아직 결과가 생성되지 않았습니다.", job.IssueKey, job.PID))
		return
	}

	ch.AnalysisText.SetText(string(content))
	ch.CurrentAnalysisPath = analysisPath
	ch.CurrentScriptPath = job.ScriptPath
	if job.PlanPath != "" {
		ch.CurrentPlanPath = job.PlanPath
	}
	ch.CopyAnalysisBtn.Enable()

	// plan 파일이 있으면 "계획 실행" 버튼 활성화
	if job.PlanPath != "" {
		ch.ExecutePlanBtn.Enable()
	}

	// AI 분석 탭으로 전환
	if ch.InnerTabs != nil {
		ch.InnerTabs.SelectIndex(1)
	}

	a.statusLabel.SetText(fmt.Sprintf("결과 로드됨: %s → %s", job.IssueKey, ch.Name))
}

// loadPreviousAnalysis는 output 폴더에서 기존 분석 결과를 스캔한다.
func (a *App) loadPreviousAnalysis() {
	outputDir := a.config.Output.Dir
	if outputDir == "" {
		outputDir = "output"
	}

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

		if _, err := os.Stat(planPath); err == nil {
			resultPath = planPath
			resultPlanPath = planPath
		} else if _, err := os.Stat(analysisPath); err == nil {
			resultPath = analysisPath
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
			ChannelIndex: 0, // 기본: 채널 1
		}
		a.completedJobs = append(a.completedJobs, job)
	}

	if a.historyList != nil {
		a.historyList.Refresh()
	}

	fmt.Printf("[History] Loaded %d previous analysis results\n", len(a.completedJobs))
}
