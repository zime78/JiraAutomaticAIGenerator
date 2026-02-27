package adapter

import (
	"path/filepath"
	"strings"
	"testing"

	"jira-ai-generator/internal/domain"
)

// TestMarkdownGenerator_Generate_ConvertsRelativeMediaPathsToAbsolute는 상대경로 미디어 경로가 절대경로로 변환되는지 검증한다.
func TestMarkdownGenerator_Generate_ConvertsRelativeMediaPathsToAbsolute(t *testing.T) {
	generator := NewMarkdownGenerator("테스트 프롬프트")

	issue := &domain.JiraIssue{
		Key:         "TEST-999",
		Summary:     "상대경로 미디어 테스트",
		Link:        "https://example.atlassian.net/browse/TEST-999",
		Description: "{{MEDIA:image.png}}\n{{MEDIA:video.mp4}}",
	}

	relativeImagePath := "output/TEST-999/image.png"
	relativeFramePath := "output/TEST-999/frames/video_frame_0001.png"

	doc, err := generator.Generate(issue, []string{relativeImagePath}, []string{relativeFramePath}, "./output")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expectedImageAbsPath, err := filepath.Abs(relativeImagePath)
	if err != nil {
		t.Fatalf("failed to build expected image abs path: %v", err)
	}
	expectedFrameAbsPath, err := filepath.Abs(relativeFramePath)
	if err != nil {
		t.Fatalf("failed to build expected frame abs path: %v", err)
	}

	if !strings.Contains(doc.Content, "]("+expectedImageAbsPath+")") {
		t.Fatalf("expected absolute image path in markdown, path=%s", expectedImageAbsPath)
	}
	if !strings.Contains(doc.Content, "]("+expectedFrameAbsPath+")") {
		t.Fatalf("expected absolute frame path in markdown, path=%s", expectedFrameAbsPath)
	}

	if strings.Contains(doc.Content, "]("+relativeImagePath+")") {
		t.Fatalf("unexpected relative image path in markdown: %s", relativeImagePath)
	}
	if strings.Contains(doc.Content, "]("+relativeFramePath+")") {
		t.Fatalf("unexpected relative frame path in markdown: %s", relativeFramePath)
	}
}

