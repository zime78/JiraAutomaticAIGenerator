package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jira-ai-generator/internal/adapter"
)

// TestParseClaudeExitCodeFromLog는 로그에서 Claude 종료코드를 정상 파싱하는지 검증한다.
func TestParseClaudeExitCodeFromLog(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "claude.log")
	logContent := "[2026-02-11 10:00:00] Running Claude...\nClaude exited with code: 7\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	exitCode, ok := parseClaudeExitCodeFromLog(logPath)
	if !ok {
		t.Fatal("expected parse result to be valid")
	}
	if exitCode != 7 {
		t.Fatalf("expected exitCode=7, got %d", exitCode)
	}
}

// TestWaitForTaskResult_ReturnsErrorOnNonZeroClaudeExit는 결과 파일이 있어도 Claude 비정상 종료를 실패로 처리하는지 검증한다.
func TestWaitForTaskResult_ReturnsErrorOnNonZeroClaudeExit(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "TEST-101_plan.md")
	logPath := filepath.Join(tempDir, "TEST-101_plan_log.txt")

	if err := os.WriteFile(outputPath, []byte("# output"), 0644); err != nil {
		t.Fatalf("failed to write output file: %v", err)
	}
	logContent := "some text\nClaude exited with code: 2\nHook validation failed: denied\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	task := &RunningTask{
		IssueKey: "TEST-101",
		PID:      0, // PID<=0이면 실행 중이 아닌 상태로 즉시 결과 판정을 수행한다.
		LogPath:  logPath,
	}

	err := waitForTaskResult(task, outputPath)
	if err == nil {
		t.Fatal("expected waitForTaskResult to return error")
	}
	if !strings.Contains(err.Error(), "exit=2") {
		t.Fatalf("expected exit code in error message, got: %v", err)
	}
}

// TestIsHookRelatedError는 Hook 설정/런타임 오류 판별 규칙을 검증한다.
func TestIsHookRelatedError(t *testing.T) {
	if !isHookRelatedError(&adapter.HookConfigurationError{Reason: "missing hook"}) {
		t.Fatal("expected hook configuration error to be detected")
	}
	if !isHookRelatedError(errors.New("runtime HOOK denied")) {
		t.Fatal("expected runtime hook error to be detected")
	}
	if isHookRelatedError(errors.New("generic timeout error")) {
		t.Fatal("did not expect generic error to be detected as hook error")
	}
}

