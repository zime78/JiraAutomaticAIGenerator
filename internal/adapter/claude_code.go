package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AnalysisResult contains the result of starting an analysis
type AnalysisResult struct {
	OutputPath string
	ScriptPath string
	PID        int
}

// ClaudeCodeAdapter implements Claude Code CLI integration
type ClaudeCodeAdapter struct {
	cliPath string
	workDir string
	enabled bool
}

// NewClaudeCodeAdapter creates a new Claude Code adapter
func NewClaudeCodeAdapter(cliPath, workDir string, enabled bool) *ClaudeCodeAdapter {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		absWorkDir = workDir
	}
	return &ClaudeCodeAdapter{
		cliPath: cliPath,
		workDir: absWorkDir,
		enabled: enabled,
	}
}

// IsEnabled returns whether Claude integration is enabled
func (c *ClaudeCodeAdapter) IsEnabled() bool {
	return c.enabled
}

// SetWorkDir sets the working directory for Claude
func (c *ClaudeCodeAdapter) SetWorkDir(workDir string) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		absWorkDir = workDir
	}
	c.workDir = absWorkDir
}

// AnalyzeIssue launches Claude as a detached background process
func (c *ClaudeCodeAdapter) AnalyzeIssue(mdFilePath, prompt string) (*AnalysisResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("Claude integration is not enabled")
	}

	fmt.Printf("[Claude] Starting analysis...\n")
	fmt.Printf("[Claude] CLI Path: %s\n", c.cliPath)
	fmt.Printf("[Claude] Work Dir: %s\n", c.workDir)
	fmt.Printf("[Claude] MD File: %s\n", mdFilePath)

	// Read the markdown file content
	mdContent, err := os.ReadFile(mdFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MD file: %w", err)
	}

	// Output path for analysis result
	outputPath := strings.TrimSuffix(mdFilePath, ".md") + "_analysis.md"

	// Create a temporary prompt file (to avoid shell escaping issues)
	promptFile := strings.TrimSuffix(mdFilePath, ".md") + "_prompt.txt"
	fullPrompt := fmt.Sprintf("%s\n\n---\n%s", prompt, string(mdContent))
	if err := os.WriteFile(promptFile, []byte(fullPrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to write prompt file: %w", err)
	}

	// Create a wrapper script for background execution
	// Create wrapper script - use file content directly as argument
	scriptPath := strings.TrimSuffix(mdFilePath, ".md") + "_run.sh"
	logFile := strings.TrimSuffix(mdFilePath, ".md") + "_log.txt"
	scriptContent := fmt.Sprintf(`#!/bin/bash
exec > "%s" 2>&1
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Starting Claude analysis..."
echo "Working directory: %s"
cd "%s"
echo "Prompt file: %s"
echo "Output file: %s"
echo ""
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Running Claude..."
%s --print "$(cat '%s')" --output-format text > /tmp/claude_output_$$.txt 2>&1
CLAUDE_EXIT=$?
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Claude exited with code: $CLAUDE_EXIT"
echo "Output size: $(wc -c < /tmp/claude_output_$$.txt) bytes"
echo ""
echo "=== Claude Output ==="
cat /tmp/claude_output_$$.txt
echo "=== End Output ==="
echo ""
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Writing final output..."

echo "# Claude 분석 결과" > "%s"
echo "" >> "%s"
echo "📅 생성 시간: $(date '+%%Y-%%m-%%d %%H:%%M:%%S')" >> "%s"
echo "📁 프로젝트: %s" >> "%s"
echo "" >> "%s"
echo "---" >> "%s"
echo "" >> "%s"
if [ $CLAUDE_EXIT -ne 0 ]; then
    echo "❌ Claude 오류 발생 (exit code: $CLAUDE_EXIT)" >> "%s"
    echo "" >> "%s"
fi
cat /tmp/claude_output_$$.txt >> "%s"
echo "" >> "%s"
echo "---" >> "%s"
echo "" >> "%s"
echo "✅ 분석 완료: $(date '+%%Y-%%m-%%d %%H:%%M:%%S')" >> "%s"

rm -f /tmp/claude_output_$$.txt "%s" "%s"
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Done!"
`, logFile, c.workDir, c.workDir, promptFile, outputPath, c.cliPath, promptFile, outputPath, outputPath, outputPath, c.workDir, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, promptFile, scriptPath)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// Launch script as a completely detached background process
	cmd := exec.Command("nohup", "bash", scriptPath)
	cmd.Dir = c.workDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start background process: %w", err)
	}

	// Don't wait for the process - let it run independently
	go func() {
		cmd.Wait() // Clean up process but don't block
	}()

	fmt.Printf("[Claude] Background process started (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("[Claude] Results will be saved to: %s\n", outputPath)
	fmt.Printf("[Claude] Log file: %s\n", logFile)
	fmt.Printf("[Claude] You can close this app - analysis will continue in background.\n")

	// Return the result with PID and paths
	return &AnalysisResult{
		OutputPath: outputPath,
		ScriptPath: scriptPath,
		PID:        cmd.Process.Pid,
	}, nil
}

// SendToClaudeAsync sends the analysis request asynchronously
func (c *ClaudeCodeAdapter) SendToClaudeAsync(mdFilePath, prompt string, onComplete func(*AnalysisResult, error)) {
	go func() {
		result, err := c.AnalyzeIssue(mdFilePath, prompt)
		if onComplete != nil {
			onComplete(result, err)
		}
	}()
}

// CheckCLIAvailable checks if Claude CLI is available
func (c *ClaudeCodeAdapter) CheckCLIAvailable() bool {
	cmd := exec.Command(c.cliPath, "--version")
	if err := cmd.Run(); err != nil {
		// Try with --help as fallback
		cmd = exec.Command(c.cliPath, "--help")
		return cmd.Run() == nil
	}
	return true
}

// GetCLIPath returns the configured CLI path
func (c *ClaudeCodeAdapter) GetCLIPath() string {
	return c.cliPath
}

// GetWorkDir returns the configured work directory
func (c *ClaudeCodeAdapter) GetWorkDir() string {
	return c.workDir
}

// BuildAnalysisPrompt builds the analysis prompt from a document
func BuildAnalysisPrompt(issueKey, mdPath string) string {
	return fmt.Sprintf(`Jira 이슈 %s를 분석해주세요.

분석 대상 파일: %s

## 요청 사항

1. **문제 분석**: 위 이슈 내용과 첨부 이미지를 분석하여 문제 상황을 파악해주세요.
2. **원인 분석**: 코드베이스를 검색하여 관련 파일을 찾고 근본 원인을 파악해주세요.
3. **수정 코드 제시**: 수정이 필요한 부분의 **구체적인 코드 변경 예시**를 diff 형식으로 제시해주세요.
4. **체크리스트**: 개발자가 확인해야 할 테스트 항목을 제공해주세요.

## 출력 형식 (반드시 이 형식을 따라주세요)

### 1. 문제 요약
(간단히 1-2줄로 요약)

### 2. 원인 분석
(관련 파일의 **전체 경로**와 문제가 되는 코드 라인 번호 명시)

### 3. 수정 코드
(각 파일별 수정 전/후 코드를 아래 형식으로 표시)

#### 파일: [전체 파일 경로]
'''kotlin (또는 해당 언어)
// 수정 전 코드
'''

'''kotlin
// 수정 후 코드
'''

### 4. 테스트 체크리스트
- [ ] 체크 항목 1
- [ ] 체크 항목 2

## 중요 규칙
- **별도의 파일을 생성하지 마세요**. 모든 분석 결과를 이 응답에 직접 출력하세요.
- 요약만 하지 말고, **복사해서 바로 적용할 수 있는 구체적인 수정 코드**를 반드시 포함하세요.
- "계획 파일에 작성했습니다" 같은 문구 없이 모든 내용을 여기에 출력하세요.`,
		issueKey, mdPath)
}

// ExtractAnalysisFromMD extracts the key content from generated markdown
func ExtractAnalysisFromMD(mdContent string) string {
	// Find the "## 문제 설명" section
	if idx := strings.Index(mdContent, "## 문제 설명"); idx >= 0 {
		endIdx := strings.Index(mdContent[idx:], "---")
		if endIdx > 0 {
			return strings.TrimSpace(mdContent[idx : idx+endIdx])
		}
		return strings.TrimSpace(mdContent[idx:])
	}
	return mdContent
}
