package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"jira-ai-generator/internal/domain"
)

// MarkdownGenerator implements port.DocumentGenerator
type MarkdownGenerator struct {
	promptTemplate string
}

// NewMarkdownGenerator creates a new markdown generator
func NewMarkdownGenerator(promptTemplate string) *MarkdownGenerator {
	if promptTemplate == "" {
		promptTemplate = "다음 Jira 이슈를 분석하고 수정 코드를 작성해주세요:"
	}
	return &MarkdownGenerator{
		promptTemplate: promptTemplate,
	}
}

// normalizeMediaPathsToAbsolute는 마크다운에 출력할 미디어 경로를 절대경로로 정규화한다.
func normalizeMediaPathsToAbsolute(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, mediaPath := range paths {
		absPath, err := filepath.Abs(mediaPath)
		if err != nil {
			normalized = append(normalized, mediaPath)
			continue
		}
		normalized = append(normalized, absPath)
	}
	return normalized
}

// Generate creates a document from a Jira issue
func (g *MarkdownGenerator) Generate(issue *domain.JiraIssue, imagePaths, framePaths []string, outputDir string) (*domain.GeneratedDocument, error) {
	var content strings.Builder

	// 출력 마크다운 경로 일관성을 위해 이미지/프레임 경로를 모두 절대경로로 통일한다.
	imagePaths = normalizeMediaPathsToAbsolute(imagePaths)
	framePaths = normalizeMediaPathsToAbsolute(framePaths)

	content.WriteString(fmt.Sprintf("# [%s] %s\n\n", issue.Key, issue.Summary))

	content.WriteString("## 이슈 정보\n\n")
	content.WriteString(fmt.Sprintf("- **Jira 링크**: [%s](%s)\n", issue.Key, issue.Link))
	content.WriteString(fmt.Sprintf("- **생성일**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	content.WriteString("\n---\n\n")

	// Create a map of filename to path for quick lookup
	imageMap := make(map[string]string)
	for _, imgPath := range imagePaths {
		filename := filepath.Base(imgPath)
		imageMap[filename] = imgPath
	}

	// Create a map of video filename to its extracted frames
	videoFrameMap := make(map[string][]string)
	for _, framePath := range framePaths {
		// Frame filename format: videoname_frame_0001.png
		frameBase := filepath.Base(framePath)
		parts := strings.Split(frameBase, "_frame_")
		if len(parts) >= 1 {
			videoName := parts[0]
			// Try common video extensions
			for _, ext := range []string{".mp4", ".mov", ".avi", ".webm", ".mkv"} {
				videoKey := videoName + ext
				videoFrameMap[videoKey] = append(videoFrameMap[videoKey], framePath)
			}
		}
	}

	// Format description and replace media markers with actual images/frames
	formattedDesc := reformatDescription(issue.Description)
	usedImages := make(map[string]bool)
	usedFrames := make(map[string]bool)

	// Replace {{MEDIA:filename}} markers with actual image markdown or video frames
	mediaPattern := regexp.MustCompile(`\{\{MEDIA:([^}]+)\}\}`)
	formattedDesc = mediaPattern.ReplaceAllStringFunc(formattedDesc, func(match string) string {
		filename := mediaPattern.FindStringSubmatch(match)[1]

		// Check if it's an image
		if imgPath, ok := imageMap[filename]; ok {
			usedImages[filename] = true
			return fmt.Sprintf("![%s](%s)", filename, imgPath)
		}

		// Check if it's a video with extracted frames
		if frames, ok := videoFrameMap[filename]; ok && len(frames) > 0 {
			var frameMarkdown strings.Builder
			frameMarkdown.WriteString(fmt.Sprintf("\n**[동영상: %s - 프레임 캡처]**\n", filename))
			for i, framePath := range frames {
				usedFrames[framePath] = true
				frameMarkdown.WriteString(fmt.Sprintf("![프레임 %d](%s)\n", i+1, framePath))
			}
			return frameMarkdown.String()
		}

		return match
	})

	// Also handle MEDIA_ID markers (fallback)
	mediaIDPattern := regexp.MustCompile(`\{\{MEDIA_ID:[^}]+\}\}`)
	formattedDesc = mediaIDPattern.ReplaceAllString(formattedDesc, "")

	content.WriteString("## 문제 설명\n\n")
	content.WriteString(formattedDesc)
	content.WriteString("\n\n---\n\n")

	// Collect unused images and unused frames for appendix
	var unusedImages []string
	for _, imgPath := range imagePaths {
		filename := filepath.Base(imgPath)
		if !usedImages[filename] {
			unusedImages = append(unusedImages, imgPath)
		}
	}

	var unusedFramePaths []string
	for _, framePath := range framePaths {
		if !usedFrames[framePath] {
			unusedFramePaths = append(unusedFramePaths, framePath)
		}
	}

	if len(unusedImages) > 0 || len(unusedFramePaths) > 0 {
		content.WriteString("## 추가 첨부 자료\n\n")

		if len(unusedImages) > 0 {
			content.WriteString("### 기타 이미지\n\n")
			for i, imgPath := range unusedImages {
				content.WriteString(fmt.Sprintf("%d. ![이미지 %d](%s)\n", i+1, i+1, imgPath))
			}
			content.WriteString("\n")
		}

		if len(unusedFramePaths) > 0 {
			content.WriteString("### 동영상 프레임 캡처\n\n")
			for i, framePath := range unusedFramePaths {
				content.WriteString(fmt.Sprintf("%d. ![프레임 %d](%s)\n", i+1, i+1, framePath))
			}
			content.WriteString("\n")
		}

		content.WriteString("---\n\n")
	}

	content.WriteString("## AI 분석 요청\n\n")
	content.WriteString(g.promptTemplate)
	content.WriteString("\n\n")
	content.WriteString("### 요청 사항\n\n")
	content.WriteString("1. **문제 분석**: 위 이슈 내용과 첨부 이미지/동영상을 분석하여 문제 상황을 파악해주세요.\n")
	content.WriteString("2. **원인 추정**: 가능한 원인을 추정하고 확인해야 할 사항을 나열해주세요.\n")
	content.WriteString("3. **수정 계획 작성**: 직접 코드를 수정하지 말고, 수정이 필요한 부분과 방법을 계획으로 작성해주세요.\n")
	content.WriteString("4. **체크리스트 제공**: 개발자가 확인해야 할 체크리스트를 제공해주세요.\n")
	content.WriteString("\n> ⚠️ 참고: 이 요청은 다른 프로젝트 컨텍스트에서 실행되므로 직접 코드 수정이 불가합니다.\n")

	doc := &domain.GeneratedDocument{
		IssueKey:   issue.Key,
		Title:      fmt.Sprintf("[%s] %s", issue.Key, issue.Summary),
		Content:    content.String(),
		OutputDir:  outputDir,
		ImagePaths: imagePaths,
		FramePaths: framePaths,
	}

	return doc, nil
}

// SaveToFile saves the document to a file
func (g *MarkdownGenerator) SaveToFile(doc *domain.GeneratedDocument) (string, error) {
	// Convert to absolute path
	absOutputDir, err := filepath.Abs(doc.OutputDir)
	if err != nil {
		absOutputDir = doc.OutputDir
	}

	issueDir := filepath.Join(absOutputDir, doc.IssueKey)
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	mdPath := filepath.Join(issueDir, fmt.Sprintf("%s.md", doc.IssueKey))
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write markdown file: %w", err)
	}

	return mdPath, nil
}

// GenerateClipboardContent creates content for clipboard
func (g *MarkdownGenerator) GenerateClipboardContent(doc *domain.GeneratedDocument) string {
	var content strings.Builder

	content.WriteString(g.promptTemplate)
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("## Jira 이슈: %s\n", doc.Title))
	content.WriteString(fmt.Sprintf("링크: (Jira 이슈 %s 참조)\n\n", doc.IssueKey))

	idx := strings.Index(doc.Content, "## 문제 설명")
	if idx >= 0 {
		endIdx := strings.Index(doc.Content[idx:], "---")
		if endIdx > 0 {
			content.WriteString(doc.Content[idx : idx+endIdx])
		}
	}

	if len(doc.ImagePaths) > 0 || len(doc.FramePaths) > 0 {
		content.WriteString("\n\n> 참고: 첨부 이미지/프레임은 로컬 폴더를 확인해주세요.\n")
		content.WriteString(fmt.Sprintf("> 위치: %s/%s/\n", doc.OutputDir, doc.IssueKey))
	}

	return content.String()
}

// Formatting utilities
var sectionPattern = regexp.MustCompile(`\[(재현 ?스텝|현 ?결과|오류 ?내용|기대 ?결과|수정 ?요청|추가 ?정보)\]`)
var multipleNewlines = regexp.MustCompile(`\n{2,}`)

func reformatDescription(description string) string {
	// Normalize line endings
	description = strings.ReplaceAll(description, "\r\n", "\n")

	// Trim leading/trailing whitespace from each line
	lines := strings.Split(description, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip completely empty lines, but keep non-empty ones
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	// Join with single newline and add line break after section headers
	result := strings.Join(cleanLines, "\n")

	// Add blank line before section headers for readability
	result = sectionPattern.ReplaceAllString(result, "\n$0")

	// Remove any double+ newlines that might have been created
	result = multipleNewlines.ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}
