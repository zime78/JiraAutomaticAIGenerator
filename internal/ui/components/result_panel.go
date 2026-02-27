package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ResultPanel 결과 표시 패널
type ResultPanel struct {
	widget.BaseWidget

	container *fyne.Container

	// 탭
	tabs *container.AppTabs

	// 이슈 정보 탭 (MarkdownViewer 사용)
	issueViewer *MarkdownViewer

	// AI 분석 결과 탭 (MarkdownViewer 사용)
	analysisViewer *MarkdownViewer

	// 검색 버튼
	searchIssueBtn    *widget.Button
	searchAnalysisBtn *widget.Button

	// 액션 버튼
	copyIssueBtn    *widget.Button
	copyAnalysisBtn *widget.Button
	exportBtn       *widget.Button

	// 콜백
	onCopyIssue    func()
	onCopyAnalysis func()
	onExport       func()
}

// NewResultPanel 새 ResultPanel 생성
func NewResultPanel() *ResultPanel {
	r := &ResultPanel{}

	// 이슈 정보 MarkdownViewer
	r.issueViewer = NewMarkdownViewer()

	// AI 분석 결과 MarkdownViewer
	r.analysisViewer = NewMarkdownViewer()

	// 검색 버튼들
	r.searchIssueBtn = widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		r.issueViewer.ShowSearch()
	})

	r.searchAnalysisBtn = widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		r.analysisViewer.ShowSearch()
	})

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

	// 이슈 정보 탭 컨텐츠
	issueActions := container.NewHBox(r.searchIssueBtn, r.copyIssueBtn)
	issueContent := container.NewBorder(
		nil,
		issueActions,
		nil,
		nil,
		r.issueViewer,
	)

	// AI 분석 탭 컨텐츠
	analysisActions := container.NewHBox(
		r.searchAnalysisBtn,
		r.copyAnalysisBtn,
		r.exportBtn,
	)
	analysisContent := container.NewBorder(
		nil,
		analysisActions,
		nil,
		nil,
		r.analysisViewer,
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
	r.issueViewer.SetContent(content)
	if content != "" {
		r.copyIssueBtn.Enable()
	} else {
		r.copyIssueBtn.Disable()
	}
}

// SetAnalysis AI 분석 결과 설정
func (r *ResultPanel) SetAnalysis(content string) {
	r.analysisViewer.SetContent(content)
	if content != "" {
		r.copyAnalysisBtn.Enable()
		r.exportBtn.Enable()
	} else {
		r.copyAnalysisBtn.Disable()
		r.exportBtn.Disable()
	}
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
	return r.issueViewer.GetContent()
}

// GetAnalysis AI 분석 결과 조회
func (r *ResultPanel) GetAnalysis() string {
	return r.analysisViewer.GetContent()
}

// Reset 상태 초기화
func (r *ResultPanel) Reset() {
	r.issueViewer.Reset()
	r.analysisViewer.Reset()
	fyne.Do(func() {
		r.copyIssueBtn.Disable()
		r.copyAnalysisBtn.Disable()
		r.exportBtn.Disable()
		r.tabs.SelectIndex(0)
	})
}

// ShowIssueSearch 이슈 정보 검색 표시
func (r *ResultPanel) ShowIssueSearch() {
	r.issueViewer.ShowSearch()
}

// ShowAnalysisSearch 분석 결과 검색 표시
func (r *ResultPanel) ShowAnalysisSearch() {
	r.analysisViewer.ShowSearch()
}
