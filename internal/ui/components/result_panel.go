package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ResultPanel 결과 표시 패널 (단일 이슈 정보 뷰)
type ResultPanel struct {
	widget.BaseWidget

	container *fyne.Container

	// 이슈 정보 뷰어 (MarkdownViewer 사용)
	issueViewer *MarkdownViewer

	// 검색 버튼
	searchIssueBtn *widget.Button

	// 액션 버튼
	copyIssueBtn *widget.Button

	// 콜백
	onCopyIssue func()
}

// NewResultPanel 새 ResultPanel 생성
func NewResultPanel() *ResultPanel {
	r := &ResultPanel{}

	// 이슈 정보 MarkdownViewer
	r.issueViewer = NewMarkdownViewer()

	// 검색 버튼
	r.searchIssueBtn = widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		r.issueViewer.ShowSearch()
	})

	// 액션 버튼
	r.copyIssueBtn = widget.NewButton("📋 복사", func() {
		if r.onCopyIssue != nil {
			r.onCopyIssue()
		}
	})
	r.copyIssueBtn.Disable()

	// 이슈 정보 컨텐츠
	issueActions := container.NewHBox(r.searchIssueBtn, r.copyIssueBtn)
	issueContent := container.NewBorder(
		nil,
		issueActions,
		nil,
		nil,
		r.issueViewer,
	)

	r.container = container.NewBorder(
		widget.NewLabelWithStyle("📝 결과", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		nil,
		nil,
		issueContent,
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

// SetOnCopyIssue 이슈 복사 콜백 설정
func (r *ResultPanel) SetOnCopyIssue(callback func()) {
	r.onCopyIssue = callback
}

// GetIssueInfo 이슈 정보 조회
func (r *ResultPanel) GetIssueInfo() string {
	return r.issueViewer.GetContent()
}

// Reset 상태 초기화
func (r *ResultPanel) Reset() {
	r.issueViewer.Reset()
	fyne.Do(func() {
		r.copyIssueBtn.Disable()
	})
}

// ShowIssueSearch 이슈 정보 검색 표시
func (r *ResultPanel) ShowIssueSearch() {
	r.issueViewer.ShowSearch()
}
