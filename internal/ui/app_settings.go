package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/config"
)

// showSettingsDialog 설정 다이얼로그 표시
func (a *App) showSettingsDialog() {
	// Jira 설정
	jiraURLEntry := widget.NewEntry()
	jiraURLEntry.SetText(a.config.Jira.URL)
	jiraURLEntry.SetMinRowsVisible(1)

	jiraEmailEntry := widget.NewEntry()
	jiraEmailEntry.SetText(a.config.Jira.Email)

	jiraAPIKeyEntry := widget.NewPasswordEntry()
	jiraAPIKeyEntry.SetText(a.config.Jira.APIKey)

	// Claude 설정
	claudeEnabledCheck := widget.NewCheck("Claude Code 활성화", nil)
	claudeEnabledCheck.SetChecked(a.config.Claude.Enabled)

	claudePathEntry := widget.NewEntry()
	claudePathEntry.SetText(a.config.Claude.CLIPath)

	hookScriptEntry := widget.NewEntry()
	hookScriptEntry.SetText(a.config.Claude.HookScriptPath)

	// 모델 선택 드롭다운
	modelSelect := widget.NewSelect(config.AvailableModels, nil)
	if a.config.Claude.Model != "" {
		modelSelect.SetSelected(a.config.Claude.Model)
	} else {
		modelSelect.SetSelected(config.AvailableModels[0])
	}

	// 출력 디렉토리
	outputDirEntry := widget.NewEntry()
	outputDirEntry.SetText(a.config.Output.Dir)

	// 프로젝트 경로 (단일)
	projectPathEntry := widget.NewEntry()
	projectPathEntry.SetText(a.config.Claude.ProjectPath)

	// 설정 파일 경로
	configPath := config.GetConfigPath()

	// 폼 구성
	form := widget.NewForm(
		widget.NewFormItem("Jira URL", jiraURLEntry),
		widget.NewFormItem("Jira Email", jiraEmailEntry),
		widget.NewFormItem("Jira API Key", jiraAPIKeyEntry),
		widget.NewFormItem("", widget.NewSeparator()),
		widget.NewFormItem("", claudeEnabledCheck),
		widget.NewFormItem("Claude CLI 경로", claudePathEntry),
		widget.NewFormItem("Claude Hook 스크립트", hookScriptEntry),
		widget.NewFormItem("Claude 모델", modelSelect),
		widget.NewFormItem("", widget.NewSeparator()),
		widget.NewFormItem("출력 디렉토리", outputDirEntry),
		widget.NewFormItem("", widget.NewSeparator()),
		widget.NewFormItem("프로젝트 경로", projectPathEntry),
	)

	// 버튼
	var settingsDialog dialog.Dialog

	saveBtn := widget.NewButton("저장", func() {
		// 설정 업데이트
		a.config.Jira.URL = jiraURLEntry.Text
		a.config.Jira.Email = jiraEmailEntry.Text
		a.config.Jira.APIKey = jiraAPIKeyEntry.Text
		a.config.Claude.Enabled = claudeEnabledCheck.Checked
		a.config.Claude.CLIPath = claudePathEntry.Text
		a.config.Claude.Model = modelSelect.Selected
		a.config.Claude.HookScriptPath = hookScriptEntry.Text
		a.config.Output.Dir = outputDirEntry.Text

		// Claude Adapter 모델 업데이트
		if a.claudeAdapter != nil {
			a.claudeAdapter.SetModel(modelSelect.Selected)
			a.claudeAdapter.SetHookScriptPath(hookScriptEntry.Text)
		}
		a.config.Claude.ProjectPath = projectPathEntry.Text

		// 채널 UI 업데이트
		if a.channel != nil && a.channel.ProjectPathEntry != nil {
			a.channel.ProjectPathEntry.SetText(a.config.Claude.ProjectPath)
		}

		// 파일에 저장
		if err := a.config.SaveDefault(); err != nil {
			dialog.ShowError(fmt.Errorf("설정 저장 실패: %w", err), a.mainWindow)
			return
		}

		settingsDialog.Hide()
		dialog.ShowInformation("설정 저장", fmt.Sprintf("설정이 저장되었습니다.\n\n저장 위치: %s", configPath), a.mainWindow)
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("취소", func() {
		settingsDialog.Hide()
	})

	buttons := container.NewHBox(
		cancelBtn,
		saveBtn,
	)

	// 상단 헤더: 타이틀 (파일명 포함) + 버튼
	titleLabel := widget.NewLabelWithStyle(fmt.Sprintf("⚙️ 설정 (%s)", configPath), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewBorder(nil, nil, titleLabel, buttons)

	// 전체 컨텐츠 구성
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		form,
	)

	// 스크롤 가능한 컨테이너 (창이 작아지면 스크롤)
	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(650, 600))

	// 커스텀 다이얼로그 생성 (타이틀은 내부 헤더에 포함)
	settingsDialog = dialog.NewCustom("", "닫기", scroll, a.mainWindow)
	settingsDialog.Resize(fyne.NewSize(750, 680))
	settingsDialog.Show()
}
