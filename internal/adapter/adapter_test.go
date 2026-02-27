package adapter_test

import (
	"os"
	"path/filepath"
	"testing"

	"jira-ai-generator/internal/adapter"
)

func TestExtractIssueKeyFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard browse URL",
			url:      "https://example.atlassian.net/browse/PROJ-123",
			expected: "PROJ-123",
		},
		{
			name:     "URL with query parameters",
			url:      "https://example.atlassian.net/browse/PROJ-456?filter=123",
			expected: "PROJ-456",
		},
		{
			name:     "software project URL",
			url:      "https://example.atlassian.net/jira/software/projects/PROJ/issues/PROJ-789",
			expected: "PROJ-789",
		},
		{
			name:     "invalid URL - no issue key",
			url:      "https://example.atlassian.net/browse/",
			expected: "",
		},
		{
			name:     "issue key only",
			url:      "ITSM-5239",
			expected: "ITSM-5239",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.ExtractIssueKeyFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractIssueKeyFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestMarkdownGenerator_Generate(t *testing.T) {
	gen := adapter.NewMarkdownGenerator("테스트 프롬프트")

	// This is a simplified test - full test would use domain.JiraIssue
	if gen == nil {
		t.Fatal("generator should not be nil")
	}
}

func TestFFmpegVideoProcessor_IsAvailable(t *testing.T) {
	processor := adapter.NewFFmpegVideoProcessor()

	// Just verify it doesn't panic
	available := processor.IsAvailable()
	t.Logf("FFmpeg available: %v", available)
}

func TestBuildAnalysisPlanPrompt(t *testing.T) {
	prompt := adapter.BuildAnalysisPlanPrompt("ITSM-5090", "/output/ITSM-5090/ITSM-5090.md")

	// plan 파일 생성 금지 지시가 포함되어야 함
	if !containsStr(prompt, "직접 전체 출력") {
		t.Error("프롬프트에 '직접 전체 출력' 지시가 포함되어야 합니다")
	}

	// EnterPlanMode 사용 금지 지시가 포함되어야 함
	if !containsStr(prompt, "EnterPlanMode") {
		t.Error("프롬프트에 EnterPlanMode 사용 금지 지시가 포함되어야 합니다")
	}

	// 구조화된 출력 형식 섹션이 포함되어야 함
	requiredSections := []string{
		"ISSUE_SUMMARY",
		"ROOT_CAUSE",
		"FILES_TO_MODIFY",
		"TEST_CHECKLIST",
		"EXECUTION_CONTEXT",
	}
	for _, section := range requiredSections {
		if !containsStr(prompt, section) {
			t.Errorf("프롬프트에 '%s' 섹션이 포함되어야 합니다", section)
		}
	}

	// 이슈 키와 파일 경로가 포함되어야 함
	if !containsStr(prompt, "ITSM-5090") {
		t.Error("프롬프트에 이슈 키가 포함되어야 합니다")
	}
	if !containsStr(prompt, "/output/ITSM-5090/ITSM-5090.md") {
		t.Error("프롬프트에 마크다운 파일 경로가 포함되어야 합니다")
	}

	// "파일에 작성했습니다" 같은 문구 사용 금지 지시 확인
	if !containsStr(prompt, "파일에 작성했습니다") {
		t.Error("프롬프트에 '파일에 작성했습니다' 사용 금지 지시가 포함되어야 합니다")
	}
}

func TestBuildAnalysisPlanPrompt_DifferentFromLegacy(t *testing.T) {
	planPrompt := adapter.BuildAnalysisPlanPrompt("ITSM-1234", "/test/path.md")
	legacyPrompt := adapter.BuildAnalysisPrompt("ITSM-1234", "/test/path.md")

	// plan 프롬프트와 기존 프롬프트는 다른 내용이어야 함
	if planPrompt == legacyPrompt {
		t.Error("plan 프롬프트는 기존 프롬프트와 달라야 합니다")
	}

	// plan 프롬프트에는 EXECUTION_CONTEXT가 있어야 함 (기존에는 없음)
	if !containsStr(planPrompt, "EXECUTION_CONTEXT") {
		t.Error("plan 프롬프트에 EXECUTION_CONTEXT 섹션이 있어야 합니다")
	}
}

func TestAnalyzeAndGeneratePlan_Disabled(t *testing.T) {
	// Claude 비활성 상태에서 호출 시 에러 반환
	claude := adapter.NewClaudeCodeAdapter("claude", false, "", "/tmp/hook.sh")
	_, err := claude.AnalyzeAndGeneratePlan("/test/test.md", "test prompt", "/tmp")
	if err == nil {
		t.Error("비활성 상태에서 에러가 반환되어야 합니다")
	}
}

func TestAnalyzeAndGeneratePlan_EmptyWorkDir(t *testing.T) {
	// 빈 workDir로 호출 시 에러 반환
	claude := adapter.NewClaudeCodeAdapter("claude", true, "", "/tmp/hook.sh")
	_, err := claude.AnalyzeAndGeneratePlan("/test/test.md", "test prompt", "")
	if err == nil {
		t.Error("빈 프로젝트 경로에서 에러가 반환되어야 합니다")
	}
}

func TestAnalyzeAndGeneratePlan_HookPathRequired(t *testing.T) {
	tempDir := t.TempDir()
	mdPath := filepath.Join(tempDir, "TEST-999.md")
	if err := os.WriteFile(mdPath, []byte("# test"), 0644); err != nil {
		t.Fatalf("failed to write temp md: %v", err)
	}

	claude := adapter.NewClaudeCodeAdapter("claude", true, "", "")
	_, err := claude.AnalyzeAndGeneratePlan(mdPath, "test prompt", tempDir)
	if err == nil {
		t.Fatal("expected hook configuration error")
	}
	if !adapter.IsHookConfigurationError(err) {
		t.Fatalf("expected hook configuration error, got: %v", err)
	}
}

// containsStr은 문자열 포함 여부를 확인하는 헬퍼 함수
func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
