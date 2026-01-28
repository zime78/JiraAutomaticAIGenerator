package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2/dialog"

	"jira-ai-generator/internal/adapter"
)

// onChannelProcess는 해당 채널에서 이슈 분석을 시작한다.
func (a *App) onChannelProcess(channelIndex int) {
	ch := a.channels[channelIndex]
	url := strings.TrimSpace(ch.UrlEntry.Text)
	if url == "" {
		dialog.ShowError(fmt.Errorf("Jira URL을 입력해주세요"), a.mainWindow)
		return
	}

	issueKey := adapter.ExtractIssueKeyFromURL(url)
	if issueKey == "" {
		dialog.ShowError(fmt.Errorf("유효하지 않은 Jira URL입니다"), a.mainWindow)
		return
	}

	ch.ProcessBtn.Disable()
	ch.CopyResultBtn.Disable()
	ch.CopyAnalysisBtn.Disable()
	ch.ExecutePlanBtn.Disable()
	ch.ProgressBar.Show()
	ch.ProgressBar.SetValue(0)
	ch.StatusLabel.SetText("이슈 조회 중...")

	// 이전 결과 초기화
	ch.ResultText.SetText("")
	ch.AnalysisText.SetText("")
	ch.CurrentDoc = nil
	ch.CurrentMDPath = ""
	ch.CurrentAnalysisPath = ""
	ch.CurrentPlanPath = ""
	ch.CurrentScriptPath = ""

	go a.processIssue(issueKey, channelIndex)
}

// processIssue는 채널별로 이슈를 처리한다.
func (a *App) processIssue(issueKey string, channelIndex int) {
	ch := a.channels[channelIndex]
	defer func() {
		ch.ProcessBtn.Enable()
		ch.ProgressBar.Hide()
	}()

	// Use UseCase to process the issue
	result, err := a.processIssueUC.Execute(issueKey, func(progress float64, status string) {
		ch.ProgressBar.SetValue(progress)
		ch.StatusLabel.SetText(status)
	})

	if err != nil {
		ch.StatusLabel.SetText(fmt.Sprintf("오류: %v", err))
		dialog.ShowError(err, a.mainWindow)
		return
	}

	ch.CurrentDoc = result.Document
	ch.CurrentMDPath = result.MDPath
	ch.ResultText.SetText(result.Document.Content)
	ch.CopyResultBtn.Enable()
	ch.InnerTabs.SelectIndex(0) // 이슈 정보 탭으로 전환
}

// onCopyChannelResult는 해당 채널의 이슈 결과를 클립보드에 복사한다.
func (a *App) onCopyChannelResult(channelIndex int) {
	ch := a.channels[channelIndex]
	if ch.CurrentDoc == nil {
		return
	}

	clipboardContent := a.docGenerator.GenerateClipboardContent(ch.CurrentDoc)
	a.mainWindow.Clipboard().SetContent(clipboardContent)

	dialog.ShowInformation("완료", "클립보드에 복사되었습니다.", a.mainWindow)
}
