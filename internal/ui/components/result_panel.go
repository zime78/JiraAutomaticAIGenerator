package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ResultPanel 결과 표시 패널
type ResultPanel struct {
	widget.BaseWidget

	container *fyne.Container

	// 탭
	tabs *container.AppTabs

	// 이슈 정보 탭
	issueInfoText *widget.Entry

	// AI 분석 결과 탭
	analysisText *widget.Entry

	// 액션 버튼
	copyIssueBtn    *widget.Button
	copyAnalysisBtn *widget.Button
	exportBtn       *widget.Button
	executePlanBtn  *widget.Button

	// 콜백
	onCopyIssue    func()
	onCopyAnalysis func()
	onExport       func()
	onExecutePlan  func()
}

// NewResultPanel 새 ResultPanel 생성
func NewResultPanel() *ResultPanel {
	r := &ResultPanel{}

	// 이슈 정보 텍스트 영역
	r.issueInfoText = widget.NewMultiLineEntry()
	r.issueInfoText.Wrapping = fyne.TextWrapWord
	r.issueInfoText.SetPlaceHolder("이슈 정보가 여기에 표시됩니다...")

	// AI 분석 결과 텍스트 영역
	r.analysisText = widget.NewMultiLineEntry()
	r.analysisText.Wrapping = fyne.TextWrapWord
	r.analysisText.SetPlaceHolder("AI 분석 결과가 여기에 표시됩니다...")

	// 액션 버튼들
	r.copyIssueBtn = widget.NewButton("📋 이슈 복사", func() {
		if r.onCopyIssue != nil {
			r.onCopyIssue()
		}
	})
	r.copyIssueBtn.Disable()

	r.copyAnalysisBtn = widget.NewButton("📋 분석 복사", func() {
		if r.onCopyAnalysis != nil {
			r.onCopyAnalysis()
		}
	})
	r.copyAnalysisBtn.Disable()

	r.exportBtn = widget.NewButton("💾 내보내기", func() {
		if r.onExport != nil {
			r.onExport()
		}
	})
	r.exportBtn.Disable()

	r.executePlanBtn = widget.NewButton("▶️ 계획 실행", func() {
		if r.onExecutePlan != nil {
			r.onExecutePlan()
		}
	})
	r.executePlanBtn.Disable()

	// 이슈 정보 탭 컨텐츠
	issueContent := container.NewBorder(
		nil,
		container.NewHBox(r.copyIssueBtn),
		nil,
		nil,
		container.NewScroll(r.issueInfoText),
	)

	// AI 분석 탭 컨텐츠
	analysisActions := container.NewHBox(
		r.copyAnalysisBtn,
		r.executePlanBtn,
		r.exportBtn,
	)
	analysisContent := container.NewBorder(
		nil,
		analysisActions,
		nil,
		nil,
		container.NewScroll(r.analysisText),
	)

	// 탭 구성
	r.tabs = container.NewAppTabs(
		container.NewTabItem("📄 이슈 정보", issueContent),
		container.NewTabItem("🤖 AI 분석", analysisContent),
	)

	r.container = container.NewBorder(
		widget.NewLabelWithStyle("📝 결과", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		nil,
		nil,
		r.tabs,
	)

	r.ExtendBaseWidget(r)
	return r
}

// CreateRenderer ResultPanel 렌더러
func (r *ResultPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.container)
}

// SetIssueInfo 이슈 정보 설정
func (r *ResultPanel) SetIssueInfo(content string) {
	r.issueInfoText.SetText(content)
	if content != "" {
		r.copyIssueBtn.Enable()
	} else {
		r.copyIssueBtn.Disable()
	}
}

// SetAnalysis AI 분석 결과 설정
func (r *ResultPanel) SetAnalysis(content string) {
	r.analysisText.SetText(content)
	if content != "" {
		r.copyAnalysisBtn.Enable()
		r.exportBtn.Enable()
	} else {
		r.copyAnalysisBtn.Disable()
		r.exportBtn.Disable()
	}
}

// EnableExecutePlan 계획 실행 버튼 활성화
func (r *ResultPanel) EnableExecutePlan() {
	r.executePlanBtn.Enable()
}

// DisableExecutePlan 계획 실행 버튼 비활성화
func (r *ResultPanel) DisableExecutePlan() {
	r.executePlanBtn.Disable()
}

// SetOnCopyIssue 이슈 복사 콜백 설정
func (r *ResultPanel) SetOnCopyIssue(callback func()) {
	r.onCopyIssue = callback
}

// SetOnCopyAnalysis 분석 복사 콜백 설정
func (r *ResultPanel) SetOnCopyAnalysis(callback func()) {
	r.onCopyAnalysis = callback
}

// SetOnExport 내보내기 콜백 설정
func (r *ResultPanel) SetOnExport(callback func()) {
	r.onExport = callback
}

// SetOnExecutePlan 계획 실행 콜백 설정
func (r *ResultPanel) SetOnExecutePlan(callback func()) {
	r.onExecutePlan = callback
}

// SelectIssueTab 이슈 정보 탭 선택
func (r *ResultPanel) SelectIssueTab() {
	r.tabs.SelectIndex(0)
}

// SelectAnalysisTab AI 분석 탭 선택
func (r *ResultPanel) SelectAnalysisTab() {
	r.tabs.SelectIndex(1)
}

// GetIssueInfo 이슈 정보 조회
func (r *ResultPanel) GetIssueInfo() string {
	return r.issueInfoText.Text
}

// GetAnalysis AI 분석 결과 조회
func (r *ResultPanel) GetAnalysis() string {
	return r.analysisText.Text
}

// Reset 상태 초기화
func (r *ResultPanel) Reset() {
	r.issueInfoText.SetText("")
	r.analysisText.SetText("")
	r.copyIssueBtn.Disable()
	r.copyAnalysisBtn.Disable()
	r.exportBtn.Disable()
	r.executePlanBtn.Disable()
	r.tabs.SelectIndex(0)
}
