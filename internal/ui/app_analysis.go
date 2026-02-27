package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"
)

var markdownImageLinkPattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// onCopyChannelAnalysis는 분석 텍스트를 클립보드에 복사한다.
func (a *App) onCopyChannelAnalysis() {
	analysisContent := ""
	if a.v2State != nil {
		analysisContent = a.v2State.resultPanel.GetIssueInfo()
	}
	if analysisContent == "" {
		return
	}

	normalized := normalizeCopiedAnalysisForAnyAI(analysisContent)
	a.mainWindow.Clipboard().SetContent(normalized)
	dialog.ShowInformation("완료", "분석 결과가 복사되었습니다.", a.mainWindow)
}

// onRefreshChannelAnalysis는 분석 결과 파일을 다시 읽는다.
func (a *App) onRefreshChannelAnalysis() {
	ch := a.channel
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

	if a.v2State != nil {
		a.v2State.resultPanel.SetIssueInfo(string(content))
	}
	ch.StatusLabel.SetText(fmt.Sprintf("분석 결과 새로고침 완료: %s", path))
}

// loadJobResultToChannel은 완료된 작업의 결과를 로드한다.
func (a *App) loadJobResultToChannel(job *AnalysisJob) {
	ch := a.channel

	// AI 분석 결과 로드 (plan 파일 우선)
	analysisPath := job.AnalysisPath
	if job.PlanPath != "" {
		analysisPath = job.PlanPath
	}

	content, err := os.ReadFile(analysisPath)
	if err != nil {
		// 이슈 마크다운 로드 (fallback)
		mdPath := job.MDPath
		if mdPath == "" {
			mdPath = strings.TrimSuffix(job.AnalysisPath, "_plan.md") + ".md"
			if _, err := os.Stat(mdPath); os.IsNotExist(err) {
				mdPath = strings.TrimSuffix(job.AnalysisPath, "_analysis.md") + ".md"
			}
		}
		if mdContent, readErr := os.ReadFile(mdPath); readErr == nil {
			if a.v2State != nil {
				a.v2State.resultPanel.SetIssueInfo(string(mdContent))
			}
			ch.CurrentMDPath = mdPath
		}
		return
	}

	if a.v2State != nil {
		a.v2State.resultPanel.SetIssueInfo(string(content))
	}
	ch.CurrentAnalysisPath = analysisPath
	ch.CurrentScriptPath = job.ScriptPath
	if job.PlanPath != "" {
		ch.CurrentPlanPath = job.PlanPath
	}

	a.statusLabel.SetText(fmt.Sprintf("결과 로드됨: %s", job.IssueKey))
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
		}
		a.completedJobs = append(a.completedJobs, job)
	}

	if a.historyList != nil {
		a.historyList.Refresh()
	}

	fmt.Printf("[History] Loaded %d previous analysis results\n", len(a.completedJobs))
}

// normalizeCopiedAnalysisForAnyAI는 복사 시 Claude 전용 문구를 AI 공통 지시로 변환한다.
// 원본 파일 내용은 유지하고 클립보드 텍스트만 중립화하여 다른 AI에서도 바로 활용 가능하게 한다.
func normalizeCopiedAnalysisForAnyAI(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}

	replacer := strings.NewReplacer(
		"# Claude Code 실행 계획", "# AI 실행 계획",
		"Claude Code에 직접 전달", "AI 코딩 도구에 직접 전달",
		"Claude Code가 알아야 할", "AI가 알아야 할",
		"Claude 분석 중 오류 발생", "AI 분석 중 오류 발생",
		"Claude 오류 발생", "AI 실행 오류 발생",
	)
	normalized := replacer.Replace(content)
	normalized = normalizeMarkdownImageLinksToAbsolute(normalized)

	// 계획 문서를 복사할 때는 어떤 AI에도 공통 적용 가능한 실행 가이드를 상단에 추가한다.
	if strings.Contains(normalized, "## 실행 지시사항") && !strings.Contains(normalized, "## AI 사용 가이드") {
		guide := strings.Join([]string{
			"## AI 사용 가이드",
			"",
			"- 아래 계획은 Claude/GPT/Gemini 등 어떤 AI 코딩 도구에도 동일하게 사용할 수 있습니다.",
			"- \"FILES_TO_MODIFY\" 범위 내에서만 수정하고, 불필요한 리팩토링은 하지 마세요.",
			"- 수정 후 빌드/테스트 결과와 변경 요약을 반드시 함께 작성하세요.",
			"",
		}, "\n")
		normalized = guide + normalized
	}

	return normalized
}

// normalizeMarkdownImageLinksToAbsolute는 마크다운 이미지 링크의 상대경로를 절대경로로 변환한다.
// 이미 절대경로/URL인 링크는 원문을 유지하여 기존 문맥을 깨지 않도록 한다.
func normalizeMarkdownImageLinksToAbsolute(content string) string {
	return markdownImageLinkPattern.ReplaceAllStringFunc(content, func(match string) string {
		subMatches := markdownImageLinkPattern.FindStringSubmatch(match)
		if len(subMatches) != 3 {
			return match
		}

		altText := subMatches[1]
		rawPath := strings.TrimSpace(subMatches[2])
		if rawPath == "" {
			return match
		}

		if isAbsoluteOrRemotePath(rawPath) {
			return match
		}

		absPath, err := filepath.Abs(rawPath)
		if err != nil {
			return match
		}

		return fmt.Sprintf("![%s](%s)", altText, absPath)
	})
}

// isAbsoluteOrRemotePath는 경로가 절대경로 또는 원격 URL인지 판별한다.
func isAbsoluteOrRemotePath(path string) bool {
	lowerPath := strings.ToLower(path)
	if strings.HasPrefix(lowerPath, "http://") ||
		strings.HasPrefix(lowerPath, "https://") ||
		strings.HasPrefix(lowerPath, "file://") ||
		strings.HasPrefix(lowerPath, "data:") {
		return true
	}

	if filepath.IsAbs(path) {
		return true
	}

	// Windows 드라이브 문자 경로(C:\...) 및 UNC 경로(\\server\share)도 절대경로로 간주한다.
	if len(path) >= 2 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' {
		return true
	}
	if strings.HasPrefix(path, `\\`) {
		return true
	}

	return false
}
