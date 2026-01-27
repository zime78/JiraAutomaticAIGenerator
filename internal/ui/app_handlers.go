package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2/dialog"

	"jira-ai-generator/internal/adapter"
)

func (a *App) onProcess() {
	url := strings.TrimSpace(a.urlEntry.Text)
	if url == "" {
		dialog.ShowError(fmt.Errorf("Jira URL을 입력해주세요"), a.mainWindow)
		return
	}

	issueKey := adapter.ExtractIssueKeyFromURL(url)
	if issueKey == "" {
		dialog.ShowError(fmt.Errorf("유효하지 않은 Jira URL입니다"), a.mainWindow)
		return
	}

	a.processBtn.Disable()
	a.copyBtn.Disable()
	a.progressBar.Show()
	a.progressBar.SetValue(0)
	a.statusLabel.SetText("이슈 조회 중...")

	go a.processIssue(issueKey)
}

func (a *App) processIssue(issueKey string) {
	defer func() {
		a.processBtn.Enable()
		a.progressBar.Hide()
	}()

	// Use UseCase to process the issue
	result, err := a.processIssueUC.Execute(issueKey, func(progress float64, status string) {
		a.progressBar.SetValue(progress)
		a.statusLabel.SetText(status)
	})

	if err != nil {
		a.statusLabel.SetText(fmt.Sprintf("오류: %v", err))
		dialog.ShowError(err, a.mainWindow)
		return
	}

	a.currentDoc = result.Document
	a.currentMDPath = result.MDPath
	a.resultText.SetText(result.Document.Content)
	a.copyBtn.Enable()
}

func (a *App) onCopy() {
	if a.currentDoc == nil {
		return
	}

	clipboardContent := a.docGenerator.GenerateClipboardContent(a.currentDoc)
	a.mainWindow.Clipboard().SetContent(clipboardContent)

	dialog.ShowInformation("완료", "클립보드에 복사되었습니다.", a.mainWindow)
}
