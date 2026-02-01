package components

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/ui/utils"
)

// MarkdownViewer 마크다운 렌더링 뷰어
type MarkdownViewer struct {
	widget.BaseWidget

	container *fyne.Container
	content   string
	parser    *utils.MarkdownParser

	// UI 요소
	scrollContainer *container.Scroll
	richText        *widget.RichText
	searchEntry     *widget.Entry
	searchResults   []utils.SearchResult
	currentResult   int

	// 섹션 접기 상태
	collapsedSections map[int]bool

	// 검색 UI
	searchBar       *fyne.Container
	searchLabel     *widget.Label
	prevBtn         *widget.Button
	nextBtn         *widget.Button
	closeSearchBtn  *widget.Button

	// 콜백
	onLinkClick func(url string)
}

// NewMarkdownViewer 새 MarkdownViewer 생성
func NewMarkdownViewer() *MarkdownViewer {
	mv := &MarkdownViewer{
		collapsedSections: make(map[int]bool),
		searchResults:     make([]utils.SearchResult, 0),
	}

	// 리치 텍스트 위젯
	mv.richText = widget.NewRichText()
	mv.richText.Wrapping = fyne.TextWrapWord

	// 스크롤 컨테이너
	mv.scrollContainer = container.NewScroll(mv.richText)

	// 검색 UI 구성
	mv.searchEntry = widget.NewEntry()
	mv.searchEntry.SetPlaceHolder("검색...")
	mv.searchEntry.OnChanged = func(query string) {
		mv.search(query)
	}

	mv.searchLabel = widget.NewLabel("0/0")

	mv.prevBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		mv.prevSearchResult()
	})
	mv.prevBtn.Disable()

	mv.nextBtn = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		mv.nextSearchResult()
	})
	mv.nextBtn.Disable()

	mv.closeSearchBtn = widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		mv.hideSearch()
	})

	mv.searchBar = container.NewBorder(
		nil, nil,
		widget.NewIcon(theme.SearchIcon()),
		container.NewHBox(mv.searchLabel, mv.prevBtn, mv.nextBtn, mv.closeSearchBtn),
		mv.searchEntry,
	)
	mv.searchBar.Hide()

	// 메인 컨테이너
	mv.container = container.NewBorder(
		mv.searchBar,
		nil, nil, nil,
		mv.scrollContainer,
	)

	mv.ExtendBaseWidget(mv)
	return mv
}

// CreateRenderer MarkdownViewer 렌더러
func (mv *MarkdownViewer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(mv.container)
}

// SetContent 마크다운 내용 설정
func (mv *MarkdownViewer) SetContent(content string) {
	mv.content = content
	mv.parser = utils.NewMarkdownParser(content)
	mv.collapsedSections = make(map[int]bool)
	mv.renderContent()
}

// GetContent 마크다운 내용 반환
func (mv *MarkdownViewer) GetContent() string {
	return mv.content
}

// renderContent 내용 렌더링
func (mv *MarkdownViewer) renderContent() {
	if mv.content == "" {
		mv.richText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{
				Text: "내용이 없습니다.",
				Style: widget.RichTextStyle{
					ColorName: theme.ColorNameDisabled,
				},
			},
		}
		mv.richText.Refresh()
		return
	}

	segments := mv.parseToSegments(mv.content)
	mv.richText.Segments = segments
	mv.richText.Refresh()
}

// parseToSegments 마크다운을 RichText 세그먼트로 변환
func (mv *MarkdownViewer) parseToSegments(content string) []widget.RichTextSegment {
	var segments []widget.RichTextSegment
	lines := strings.Split(content, "\n")

	inCodeBlock := false
	var codeBlockContent strings.Builder
	codeBlockLang := ""

	for _, line := range lines {
		// 코드 블록 처리
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// 코드 블록 종료
				segments = append(segments, mv.createCodeBlockSegment(codeBlockLang, codeBlockContent.String()))
				codeBlockContent.Reset()
				inCodeBlock = false
				codeBlockLang = ""
			} else {
				// 코드 블록 시작
				inCodeBlock = true
				codeBlockLang = strings.TrimPrefix(line, "```")
			}
			continue
		}

		if inCodeBlock {
			if codeBlockContent.Len() > 0 {
				codeBlockContent.WriteString("\n")
			}
			codeBlockContent.WriteString(line)
			continue
		}

		// 헤더 처리
		if strings.HasPrefix(line, "# ") {
			segments = append(segments, mv.createHeaderSegment(1, strings.TrimPrefix(line, "# ")))
		} else if strings.HasPrefix(line, "## ") {
			segments = append(segments, mv.createHeaderSegment(2, strings.TrimPrefix(line, "## ")))
		} else if strings.HasPrefix(line, "### ") {
			segments = append(segments, mv.createHeaderSegment(3, strings.TrimPrefix(line, "### ")))
		} else if strings.HasPrefix(line, "#### ") {
			segments = append(segments, mv.createHeaderSegment(4, strings.TrimPrefix(line, "#### ")))
		} else if strings.HasPrefix(line, "##### ") {
			segments = append(segments, mv.createHeaderSegment(5, strings.TrimPrefix(line, "##### ")))
		} else if strings.HasPrefix(line, "###### ") {
			segments = append(segments, mv.createHeaderSegment(6, strings.TrimPrefix(line, "###### ")))
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			// 리스트 아이템
			segments = append(segments, mv.createListItemSegment(strings.TrimLeft(line, "-* ")))
		} else if len(line) > 0 && line[0] >= '0' && line[0] <= '9' && strings.Contains(line, ". ") {
			// 번호 리스트
			idx := strings.Index(line, ". ")
			segments = append(segments, mv.createNumberedListSegment(line[:idx+1], line[idx+2:]))
		} else if strings.HasPrefix(line, "> ") {
			// 인용문
			segments = append(segments, mv.createQuoteSegment(strings.TrimPrefix(line, "> ")))
		} else if strings.TrimSpace(line) == "" {
			// 빈 줄
			segments = append(segments, &widget.TextSegment{Text: "\n"})
		} else {
			// 일반 텍스트 (인라인 스타일 처리)
			segments = append(segments, mv.parseInlineStyles(line)...)
			segments = append(segments, &widget.TextSegment{Text: "\n"})
		}
	}

	return segments
}

// createHeaderSegment 헤더 세그먼트 생성
func (mv *MarkdownViewer) createHeaderSegment(level int, text string) widget.RichTextSegment {
	// 헤더 레벨에 따른 마커 추가
	var prefix string
	switch level {
	case 1:
		prefix = "▶ "
	case 2:
		prefix = "► "
	case 3:
		prefix = "▸ "
	default:
		prefix = "• "
	}

	return &widget.TextSegment{
		Text: "\n" + prefix + text + "\n",
		Style: widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		},
	}
}

// createCodeBlockSegment 코드 블록 세그먼트 생성
func (mv *MarkdownViewer) createCodeBlockSegment(lang, code string) widget.RichTextSegment {
	return &widget.TextSegment{
		Text: "\n" + code + "\n",
		Style: widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Monospace: true},
			ColorName: theme.ColorNamePrimary,
		},
	}
}

// createListItemSegment 리스트 아이템 세그먼트 생성
func (mv *MarkdownViewer) createListItemSegment(text string) widget.RichTextSegment {
	return &widget.TextSegment{
		Text: "  • " + text + "\n",
	}
}

// createNumberedListSegment 번호 리스트 세그먼트 생성
func (mv *MarkdownViewer) createNumberedListSegment(num, text string) widget.RichTextSegment {
	return &widget.TextSegment{
		Text: "  " + num + " " + text + "\n",
	}
}

// createQuoteSegment 인용문 세그먼트 생성
func (mv *MarkdownViewer) createQuoteSegment(text string) widget.RichTextSegment {
	return &widget.TextSegment{
		Text: "│ " + text + "\n",
		Style: widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Italic: true},
			ColorName: theme.ColorNameDisabled,
		},
	}
}

// parseInlineStyles 인라인 스타일 파싱
func (mv *MarkdownViewer) parseInlineStyles(text string) []widget.RichTextSegment {
	var segments []widget.RichTextSegment

	// 간단한 구현: 볼드, 이탤릭, 인라인 코드
	// 실제로는 더 복잡한 파싱이 필요하지만 기본 구현
	segments = append(segments, &widget.TextSegment{
		Text: text,
	})

	return segments
}

// ShowSearch 검색 바 표시
func (mv *MarkdownViewer) ShowSearch() {
	mv.searchBar.Show()
	mv.searchEntry.FocusGained()
	mv.Refresh()
}

// hideSearch 검색 바 숨기기
func (mv *MarkdownViewer) hideSearch() {
	mv.searchBar.Hide()
	mv.searchEntry.SetText("")
	mv.searchResults = nil
	mv.currentResult = 0
	mv.updateSearchLabel()
	mv.Refresh()
}

// search 검색 수행
func (mv *MarkdownViewer) search(query string) {
	if query == "" {
		mv.searchResults = nil
		mv.currentResult = 0
		mv.updateSearchLabel()
		mv.prevBtn.Disable()
		mv.nextBtn.Disable()
		return
	}

	if mv.parser != nil {
		mv.searchResults = mv.parser.Search(query)
		mv.currentResult = 0
		mv.updateSearchLabel()

		if len(mv.searchResults) > 0 {
			mv.prevBtn.Enable()
			mv.nextBtn.Enable()
			mv.highlightCurrentResult()
		} else {
			mv.prevBtn.Disable()
			mv.nextBtn.Disable()
		}
	}
}

// prevSearchResult 이전 검색 결과
func (mv *MarkdownViewer) prevSearchResult() {
	if len(mv.searchResults) == 0 {
		return
	}
	mv.currentResult--
	if mv.currentResult < 0 {
		mv.currentResult = len(mv.searchResults) - 1
	}
	mv.updateSearchLabel()
	mv.highlightCurrentResult()
}

// nextSearchResult 다음 검색 결과
func (mv *MarkdownViewer) nextSearchResult() {
	if len(mv.searchResults) == 0 {
		return
	}
	mv.currentResult++
	if mv.currentResult >= len(mv.searchResults) {
		mv.currentResult = 0
	}
	mv.updateSearchLabel()
	mv.highlightCurrentResult()
}

// updateSearchLabel 검색 라벨 업데이트
func (mv *MarkdownViewer) updateSearchLabel() {
	if len(mv.searchResults) == 0 {
		mv.searchLabel.SetText("0/0")
	} else {
		mv.searchLabel.SetText(fmt.Sprintf("%d/%d", mv.currentResult+1, len(mv.searchResults)))
	}
}

// highlightCurrentResult 현재 검색 결과 하이라이트
func (mv *MarkdownViewer) highlightCurrentResult() {
	// 스크롤 위치 조정 - 실제 구현에서는 라인 위치를 계산해야 함
	// 현재는 간단한 구현
}

// SetOnLinkClick 링크 클릭 콜백 설정
func (mv *MarkdownViewer) SetOnLinkClick(callback func(url string)) {
	mv.onLinkClick = callback
}

// ScrollToTop 맨 위로 스크롤
func (mv *MarkdownViewer) ScrollToTop() {
	mv.scrollContainer.ScrollToTop()
}

// ScrollToBottom 맨 아래로 스크롤
func (mv *MarkdownViewer) ScrollToBottom() {
	mv.scrollContainer.ScrollToBottom()
}

// Reset 초기화
func (mv *MarkdownViewer) Reset() {
	mv.content = ""
	mv.parser = nil
	mv.searchResults = nil
	mv.currentResult = 0
	mv.collapsedSections = make(map[int]bool)
	mv.hideSearch()
	mv.richText.Segments = nil
	mv.richText.Refresh()
}

// MarkdownPreview 간단한 마크다운 미리보기 (canvas 기반)
type MarkdownPreview struct {
	widget.BaseWidget
	container *fyne.Container
	content   string
}

// NewMarkdownPreview 새 미리보기 생성
func NewMarkdownPreview() *MarkdownPreview {
	mp := &MarkdownPreview{}
	mp.container = container.NewVBox()
	mp.ExtendBaseWidget(mp)
	return mp
}

// CreateRenderer 렌더러 생성
func (mp *MarkdownPreview) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(mp.container)
}

// SetContent 내용 설정
func (mp *MarkdownPreview) SetContent(content string) {
	mp.content = content
	mp.render()
}

// render 렌더링
func (mp *MarkdownPreview) render() {
	mp.container.RemoveAll()

	lines := strings.Split(mp.content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			text := canvas.NewText(strings.TrimPrefix(line, "# "), theme.ForegroundColor())
			text.TextSize = 24
			text.TextStyle = fyne.TextStyle{Bold: true}
			mp.container.Add(text)
		} else if strings.HasPrefix(line, "## ") {
			text := canvas.NewText(strings.TrimPrefix(line, "## "), theme.ForegroundColor())
			text.TextSize = 20
			text.TextStyle = fyne.TextStyle{Bold: true}
			mp.container.Add(text)
		} else if strings.HasPrefix(line, "### ") {
			text := canvas.NewText(strings.TrimPrefix(line, "### "), theme.ForegroundColor())
			text.TextSize = 18
			text.TextStyle = fyne.TextStyle{Bold: true}
			mp.container.Add(text)
		} else if strings.TrimSpace(line) != "" {
			text := canvas.NewText(line, theme.ForegroundColor())
			text.TextSize = 14
			mp.container.Add(text)
		}
	}

	mp.container.Refresh()
}

// 컬러 상수
var (
	CodeBlockBg = color.RGBA{R: 40, G: 44, B: 52, A: 255}
	QuoteBorder = color.RGBA{R: 100, G: 100, B: 100, A: 255}
)
