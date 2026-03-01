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

// TestAnalyzeAndGeneratePlan_HookPathEmpty는 hookScriptPath가 비어있을 때
// 에러 없이 Hook을 스킵하고 Claude 분석이 진행되는지 검증한다.
// (실제 Claude CLI가 없으므로 스크립트 실행 단계에서 실패하지만, Hook 에러는 발생하지 않아야 한다)
func TestAnalyzeAndGeneratePlan_HookPathEmpty(t *testing.T) {
	tempDir := t.TempDir()
	mdPath := filepath.Join(tempDir, "TEST-999.md")
	if err := os.WriteFile(mdPath, []byte("# test"), 0644); err != nil {
		t.Fatalf("failed to write temp md: %v", err)
	}

	claude := adapter.NewClaudeCodeAdapter("claude", true, "", "")
	_, err := claude.AnalyzeAndGeneratePlan(mdPath, "test prompt", tempDir)
	// hookScriptPath가 비어있어도 HookConfigurationError가 발생하면 안 된다
	if err != nil && adapter.IsHookConfigurationError(err) {
		t.Fatalf("empty hook_script_path should not cause HookConfigurationError, got: %v", err)
	}
}

// TestAnalyzeAndGeneratePlan_HookPathInvalid는 hookScriptPath에 디렉터리를 지정했을 때
// HookConfigurationError가 발생하는지 검증한다.
func TestAnalyzeAndGeneratePlan_HookPathInvalid(t *testing.T) {
	tempDir := t.TempDir()
	mdPath := filepath.Join(tempDir, "TEST-999.md")
	if err := os.WriteFile(mdPath, []byte("# test"), 0644); err != nil {
		t.Fatalf("failed to write temp md: %v", err)
	}

	// 디렉터리를 hook script 경로로 지정 → 에러 발생해야 함
	claude := adapter.NewClaudeCodeAdapter("claude", true, "", tempDir)
	_, err := claude.AnalyzeAndGeneratePlan(mdPath, "test prompt", tempDir)
	if err == nil {
		t.Fatal("expected hook configuration error for directory path")
	}
	if !adapter.IsHookConfigurationError(err) {
		t.Fatalf("expected hook configuration error, got: %v", err)
	}
}

// TestExtractAISections_Normal은 정상적인 plan.md에서 AI 분석 결과 섹션을 추출하는지 검증한다.
func TestExtractAISections_Normal(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "TEST-001_plan.md")

	content := `# Claude Code 실행 계획

## Jira 이슈 컨텍스트
이슈 내용...

---

## AI 분석 결과
생성 시간: 2026-02-27 21:47:17
프로젝트: /Users/test/project

### ISSUE_SUMMARY
테스트 이슈 요약

### ROOT_CAUSE
파일: /src/main.kt (라인 42)

### FILES_TO_MODIFY
#### 파일: /src/main.kt
- 수정 이유: 버그 수정

### TEST_CHECKLIST
- [ ] 테스트 항목 1

### EXECUTION_CONTEXT
추가 컨텍스트 정보

---

## 실행 지시사항
위 분석 결과를 바탕으로 수행하세요.
`
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	result := adapter.ExtractAISections(planPath)

	if result == "" {
		t.Fatal("AI 분석 결과 섹션이 추출되어야 합니다")
	}
	if !containsStr(result, "ISSUE_SUMMARY") {
		t.Error("추출 결과에 ISSUE_SUMMARY가 포함되어야 합니다")
	}
	if !containsStr(result, "ROOT_CAUSE") {
		t.Error("추출 결과에 ROOT_CAUSE가 포함되어야 합니다")
	}
	if !containsStr(result, "FILES_TO_MODIFY") {
		t.Error("추출 결과에 FILES_TO_MODIFY가 포함되어야 합니다")
	}
	// 실행 지시사항은 포함되면 안 됨
	if containsStr(result, "실행 지시사항") {
		t.Error("추출 결과에 '실행 지시사항'이 포함되면 안 됩니다")
	}
	// Jira 이슈 컨텍스트는 포함되면 안 됨
	if containsStr(result, "Jira 이슈 컨텍스트") {
		t.Error("추출 결과에 'Jira 이슈 컨텍스트'가 포함되면 안 됩니다")
	}
}

// TestExtractAISections_EmptyPath는 빈 경로에서 빈 문자열을 반환하는지 검증한다.
func TestExtractAISections_EmptyPath(t *testing.T) {
	result := adapter.ExtractAISections("")
	if result != "" {
		t.Error("빈 경로에서 빈 문자열을 반환해야 합니다")
	}
}

// TestExtractAISections_FileNotFound는 존재하지 않는 파일에서 빈 문자열을 반환하는지 검증한다.
func TestExtractAISections_FileNotFound(t *testing.T) {
	result := adapter.ExtractAISections("/nonexistent/path_plan.md")
	if result != "" {
		t.Error("파일 미존재 시 빈 문자열을 반환해야 합니다")
	}
}

// TestExtractAISections_NoAISection은 AI 분석 결과 섹션이 없는 파일에서 빈 문자열을 반환하는지 검증한다.
func TestExtractAISections_NoAISection(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "TEST-002_plan.md")

	content := `# 일반 마크다운 파일
## 섹션 1
내용...
## 섹션 2
내용...
`
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	result := adapter.ExtractAISections(planPath)
	if result != "" {
		t.Error("AI 분석 결과 섹션이 없는 파일에서 빈 문자열을 반환해야 합니다")
	}
}

// TestBuildPhase2PromptWithPlanContext_WithContent는 1차 결과 포함 시
// Phase 2 전용 프롬프트가 생성되는지 검증한다.
func TestBuildPhase2PromptWithPlanContext_WithContent(t *testing.T) {
	phase1Content := `## AI 분석 결과

### ROOT_CAUSE
파일: /src/main.kt (라인 42)
날짜 변환 로직 오류

### FILES_TO_MODIFY
#### 파일: /src/main.kt
- 수정 이유: 시간대 변환 버그`

	prompt := adapter.BuildPhase2PromptWithPlanContext("ITSM-1234", "/test/path.md", phase1Content)

	// Phase 2 전용 내용이 포함되어야 함
	if !containsStr(prompt, "1차 분석 결과") {
		t.Error("프롬프트에 '1차 분석 결과' 참조가 포함되어야 합니다")
	}
	if !containsStr(prompt, "수정 가이드") {
		t.Error("프롬프트에 '수정 가이드' 섹션이 포함되어야 합니다")
	}
	// 1차 분석 내용이 포함되어야 함
	if !containsStr(prompt, "ROOT_CAUSE") {
		t.Error("프롬프트에 1차 분석의 ROOT_CAUSE가 포함되어야 합니다")
	}
	if !containsStr(prompt, "/src/main.kt") {
		t.Error("프롬프트에 1차 분석의 파일 경로가 포함되어야 합니다")
	}
	// 이슈 키와 대상 파일 경로 포함
	if !containsStr(prompt, "ITSM-1234") {
		t.Error("프롬프트에 이슈 키가 포함되어야 합니다")
	}
	// 출력 형식 섹션 포함
	requiredSections := []string{"ISSUE_SUMMARY", "FILES_TO_MODIFY", "TEST_CHECKLIST", "EXECUTION_CONTEXT"}
	for _, section := range requiredSections {
		if !containsStr(prompt, section) {
			t.Errorf("프롬프트에 '%s' 섹션이 포함되어야 합니다", section)
		}
	}
}

// TestBuildPhase2PromptWithPlanContext_EmptyContent는 빈 1차 결과에서
// 기존 BuildAnalysisPlanPrompt로 fallback되는지 검증한다.
func TestBuildPhase2PromptWithPlanContext_EmptyContent(t *testing.T) {
	prompt := adapter.BuildPhase2PromptWithPlanContext("ITSM-1234", "/test/path.md", "")
	legacyPrompt := adapter.BuildAnalysisPlanPrompt("ITSM-1234", "/test/path.md")

	if prompt != legacyPrompt {
		t.Error("빈 1차 결과에서 기존 BuildAnalysisPlanPrompt로 fallback해야 합니다")
	}
}

// TestBuildPhase2PromptWithPlanContext_WhitespaceContent는 공백만 있는 1차 결과에서
// fallback되는지 검증한다.
func TestBuildPhase2PromptWithPlanContext_WhitespaceContent(t *testing.T) {
	prompt := adapter.BuildPhase2PromptWithPlanContext("ITSM-1234", "/test/path.md", "   \n  ")
	legacyPrompt := adapter.BuildAnalysisPlanPrompt("ITSM-1234", "/test/path.md")

	if prompt != legacyPrompt {
		t.Error("공백만 있는 1차 결과에서 기존 BuildAnalysisPlanPrompt로 fallback해야 합니다")
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
