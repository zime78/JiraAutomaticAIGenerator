package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNormalizeCopiedAnalysisForAnyAI_RewritesClaudeTerms는 Claude 전용 문구가 AI 공통 문구로 변환되는지 검증한다.
func TestNormalizeCopiedAnalysisForAnyAI_RewritesClaudeTerms(t *testing.T) {
	input := strings.Join([]string{
		"# Claude Code 실행 계획",
		"",
		"> 이 파일은 Claude Code에 직접 전달하여 자동 수정을 실행할 수 있는 구조화된 계획입니다.",
		"",
		"### EXECUTION_CONTEXT",
		"(이 수정을 실행할 때 Claude Code가 알아야 할 추가 컨텍스트)",
		"",
		"## 실행 지시사항",
		"",
		"- 파일 수정",
	}, "\n")

	output := normalizeCopiedAnalysisForAnyAI(input)

	if !strings.Contains(output, "# AI 실행 계획") {
		t.Fatalf("Claude 헤더가 AI 공통 헤더로 변환되지 않았습니다: %s", output)
	}
	if strings.Contains(output, "Claude Code") {
		t.Fatalf("출력에 Claude Code 문구가 남아있습니다: %s", output)
	}
	if !strings.Contains(output, "## AI 사용 가이드") {
		t.Fatalf("AI 공통 가이드가 추가되지 않았습니다: %s", output)
	}
}

// TestNormalizeCopiedAnalysisForAnyAI_NormalizesRelativeImagePath는 상대 이미지 경로가 절대경로로 변환되는지 검증한다.
func TestNormalizeCopiedAnalysisForAnyAI_NormalizesRelativeImagePath(t *testing.T) {
	relative := "output/ITSM-6046/frames/frame_0001.png"
	input := "![프레임 1](" + relative + ")"

	output := normalizeCopiedAnalysisForAnyAI(input)

	absPath, err := filepath.Abs(relative)
	if err != nil {
		t.Fatalf("절대경로 계산 실패: %v", err)
	}
	expected := "![프레임 1](" + absPath + ")"
	if output != expected {
		t.Fatalf("상대 경로 절대경로 변환 실패\nexpected: %s\nactual:   %s", expected, output)
	}
}

