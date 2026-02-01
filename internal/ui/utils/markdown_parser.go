package utils

import (
	"regexp"
	"strings"
)

// MarkdownSection 마크다운 섹션
type MarkdownSection struct {
	Level   int
	Title   string
	Content string
	Start   int
	End     int
}

// MarkdownCodeBlock 코드 블록
type MarkdownCodeBlock struct {
	Language string
	Code     string
	Start    int
	End      int
}

// MarkdownParser 마크다운 파서
type MarkdownParser struct {
	content string
}

// NewMarkdownParser 새 마크다운 파서 생성
func NewMarkdownParser(content string) *MarkdownParser {
	return &MarkdownParser{content: content}
}

// ParseSections 섹션 파싱
func (p *MarkdownParser) ParseSections() []MarkdownSection {
	var sections []MarkdownSection
	lines := strings.Split(p.content, "\n")

	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	var currentSection *MarkdownSection
	var contentLines []string
	lineStart := 0

	for _, line := range lines {
		match := headerRegex.FindStringSubmatch(line)
		if match != nil {
			// 이전 섹션 저장
			if currentSection != nil {
				currentSection.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
				currentSection.End = lineStart - 1
				sections = append(sections, *currentSection)
			}

			// 새 섹션 시작
			currentSection = &MarkdownSection{
				Level: len(match[1]),
				Title: match[2],
				Start: lineStart,
			}
			contentLines = nil
		} else if currentSection != nil {
			contentLines = append(contentLines, line)
		}
		lineStart += len(line) + 1
	}

	// 마지막 섹션 저장
	if currentSection != nil {
		currentSection.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
		currentSection.End = len(p.content)
		sections = append(sections, *currentSection)
	}

	return sections
}

// ParseCodeBlocks 코드 블록 파싱
func (p *MarkdownParser) ParseCodeBlocks() []MarkdownCodeBlock {
	var blocks []MarkdownCodeBlock
	codeBlockRegex := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")

	matches := codeBlockRegex.FindAllStringSubmatchIndex(p.content, -1)
	for _, match := range matches {
		if len(match) >= 6 {
			lang := p.content[match[2]:match[3]]
			code := p.content[match[4]:match[5]]
			blocks = append(blocks, MarkdownCodeBlock{
				Language: lang,
				Code:     strings.TrimSpace(code),
				Start:    match[0],
				End:      match[1],
			})
		}
	}

	return blocks
}

// ExtractLinks 링크 추출
func (p *MarkdownParser) ExtractLinks() [][2]string {
	var links [][2]string
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	matches := linkRegex.FindAllStringSubmatch(p.content, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			links = append(links, [2]string{match[1], match[2]})
		}
	}

	return links
}

// StripMarkdown 마크다운 문법 제거하여 순수 텍스트 반환
func (p *MarkdownParser) StripMarkdown() string {
	content := p.content

	// 코드 블록 제거
	codeBlockRegex := regexp.MustCompile("(?s)```.*?```")
	content = codeBlockRegex.ReplaceAllString(content, "")

	// 인라인 코드 제거
	inlineCodeRegex := regexp.MustCompile("`[^`]+`")
	content = inlineCodeRegex.ReplaceAllString(content, "")

	// 링크를 텍스트로 변환
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	content = linkRegex.ReplaceAllString(content, "$1")

	// 이미지 제거
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	content = imageRegex.ReplaceAllString(content, "")

	// 헤더 마커 제거
	headerRegex := regexp.MustCompile(`(?m)^#{1,6}\s+`)
	content = headerRegex.ReplaceAllString(content, "")

	// 강조 마커 제거
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	content = boldRegex.ReplaceAllString(content, "$1")

	italicRegex := regexp.MustCompile(`\*([^*]+)\*`)
	content = italicRegex.ReplaceAllString(content, "$1")

	// 리스트 마커 제거
	listRegex := regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`)
	content = listRegex.ReplaceAllString(content, "")

	// 번호 리스트 마커 제거
	numListRegex := regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`)
	content = numListRegex.ReplaceAllString(content, "")

	return strings.TrimSpace(content)
}

// Search 마크다운 내 검색
func (p *MarkdownParser) Search(query string) []SearchResult {
	var results []SearchResult
	query = strings.ToLower(query)
	lines := strings.Split(p.content, "\n")

	for i, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, query) {
			results = append(results, SearchResult{
				Line:    i + 1,
				Content: line,
				Index:   strings.Index(lowerLine, query),
			})
		}
	}

	return results
}

// SearchResult 검색 결과
type SearchResult struct {
	Line    int
	Content string
	Index   int
}

// GetTableOfContents 목차 생성
func (p *MarkdownParser) GetTableOfContents() []TOCEntry {
	var toc []TOCEntry
	sections := p.ParseSections()

	for _, section := range sections {
		toc = append(toc, TOCEntry{
			Level: section.Level,
			Title: section.Title,
			Line:  p.getLineNumber(section.Start),
		})
	}

	return toc
}

// TOCEntry 목차 항목
type TOCEntry struct {
	Level int
	Title string
	Line  int
}

// getLineNumber 위치에서 라인 번호 계산
func (p *MarkdownParser) getLineNumber(pos int) int {
	return strings.Count(p.content[:pos], "\n") + 1
}
