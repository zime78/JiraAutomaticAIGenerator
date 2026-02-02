package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"jira-ai-generator/internal/logger"
)

// AnalysisPhase는 분석 실행 단계를 나타냄
type AnalysisPhase int

const (
	PhaseAnalyze AnalysisPhase = iota // Phase 1: 읽기 전용 분석 → _plan.md 생성
	PhaseExecute                      // Phase 2: 계획 실행 → _execution.md 생성
)

// AnalysisResult contains the result of starting an analysis
type AnalysisResult struct {
	OutputPath string
	ScriptPath string
	PID        int
}

// PlanResult는 Phase 1 (분석 및 계획 생성) 결과를 담는 구조체
type PlanResult struct {
	PlanPath   string // _plan.md 파일 경로
	ScriptPath string // 실행 스크립트 경로
	LogPath    string // 로그 파일 경로
	PID        int    // 백그라운드 프로세스 ID
}

// ClaudeCodeAdapter implements Claude Code CLI integration
type ClaudeCodeAdapter struct {
	cliPath string
	enabled bool
	model   string
}

// NewClaudeCodeAdapter creates a new Claude Code adapter
func NewClaudeCodeAdapter(cliPath string, enabled bool, model string) *ClaudeCodeAdapter {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &ClaudeCodeAdapter{
		cliPath: cliPath,
		enabled: enabled,
		model:   model,
	}
}

// GetModel returns the configured model
func (c *ClaudeCodeAdapter) GetModel() string {
	return c.model
}

// SetModel updates the model
func (c *ClaudeCodeAdapter) SetModel(model string) {
	c.model = model
}

// IsEnabled returns whether Claude integration is enabled
func (c *ClaudeCodeAdapter) IsEnabled() bool {
	return c.enabled
}

// resolveWorkDir은 workDir을 절대 경로로 변환한다. 비어있으면 에러를 반환한다.
func resolveWorkDir(workDir string) (string, error) {
	if workDir == "" {
		return "", fmt.Errorf("프로젝트 경로가 설정되지 않았습니다. 채널별 프로젝트 경로를 입력해주세요")
	}
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return workDir, nil
	}
	return absDir, nil
}

// AnalyzeIssue launches Claude as a detached background process
func (c *ClaudeCodeAdapter) AnalyzeIssue(mdFilePath, prompt, workDir string) (*AnalysisResult, error) {
	defer logger.DebugFunc("AnalyzeIssue")()
	logger.Debug("AnalyzeIssue: mdPath=%s, workDir=%s", mdFilePath, workDir)

	if !c.enabled {
		logger.Debug("AnalyzeIssue: Claude integration is not enabled")
		return nil, fmt.Errorf("Claude integration is not enabled")
	}

	effectiveDir, err := resolveWorkDir(workDir)
	if err != nil {
		logger.Debug("AnalyzeIssue: resolveWorkDir failed: %v", err)
		return nil, err
	}
	logger.Debug("AnalyzeIssue: effectiveDir=%s", effectiveDir)

	fmt.Printf("[Claude] Starting analysis...\n")
	fmt.Printf("[Claude] CLI Path: %s\n", c.cliPath)
	fmt.Printf("[Claude] Work Dir: %s\n", effectiveDir)
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
%s --model %s --print "$(cat '%s')" --output-format text > /tmp/claude_output_$$.txt 2>&1
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
`, logFile, effectiveDir, effectiveDir, promptFile, outputPath, c.cliPath, c.model, promptFile, outputPath, outputPath, outputPath, effectiveDir, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, outputPath, promptFile, scriptPath)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// Launch script as a completely detached background process
	cmd := exec.Command("nohup", "bash", scriptPath)
	cmd.Dir = effectiveDir
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

	logger.Debug("AnalyzeIssue: completed successfully, PID=%d, output=%s", cmd.Process.Pid, outputPath)

	// Return the result with PID and paths
	return &AnalysisResult{
		OutputPath: outputPath,
		ScriptPath: scriptPath,
		PID:        cmd.Process.Pid,
	}, nil
}

// SendToClaudeAsync sends the analysis request asynchronously
func (c *ClaudeCodeAdapter) SendToClaudeAsync(mdFilePath, prompt, workDir string, onComplete func(*AnalysisResult, error)) {
	go func() {
		result, err := c.AnalyzeIssue(mdFilePath, prompt, workDir)
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

// BuildAnalysisPlanPrompt는 Phase 1용 프롬프트를 생성한다.
// 기존 BuildAnalysisPrompt와 달리 모든 분석 결과를 인라인으로 출력하도록 강제하고,
// Phase 2에서 바로 실행 가능한 구조화된 형식으로 출력을 요구한다.
func BuildAnalysisPlanPrompt(issueKey, mdPath string) string {
	return fmt.Sprintf(`Jira 이슈 %s를 분석하고 수정 계획을 작성해주세요.

분석 대상 파일: %s

## 절대 규칙
- 모든 분석 결과를 이 응답에 **직접 전체 출력**하세요.
- 별도의 플랜 파일이나 외부 파일을 절대 생성하지 마세요.
- "파일에 작성했습니다", "계획을 만들었습니다" 같은 문구를 사용하지 마세요.
- EnterPlanMode 도구를 사용하지 마세요.
- TodoWrite 도구를 사용하지 마세요.
- 요약이 아닌 **전체 상세 분석**을 출력하세요.

## 분석 절차
1. 분석 대상 파일을 읽어 이슈 내용과 첨부 이미지를 파악하세요.
2. 코드베이스를 검색하여 관련 파일을 찾으세요.
3. 근본 원인을 파악하세요.
4. 구체적인 수정 코드를 제시하세요.

## 출력 형식 (반드시 이 구조를 정확히 따르세요)

### ISSUE_SUMMARY
(이슈 요약 1-2줄)

### ROOT_CAUSE
(관련 파일의 **절대 경로**와 문제가 되는 코드 라인 번호를 명시하여 원인 분석)

### FILES_TO_MODIFY
(수정이 필요한 각 파일에 대해 아래 형식으로 작성)

#### 파일: [절대 파일 경로]
- 수정 이유: [왜 수정이 필요한지]

수정 전:
` + "```" + `kotlin (또는 해당 언어)
// 기존 코드
` + "```" + `

수정 후:
` + "```" + `kotlin
// 변경된 코드
` + "```" + `

### TEST_CHECKLIST
- [ ] 체크 항목 1
- [ ] 체크 항목 2

### EXECUTION_CONTEXT
(이 수정을 실행할 때 Claude Code가 알아야 할 추가 컨텍스트: 관련 클래스 관계, 의존성, 주의사항 등)

## 중요 규칙
- **별도의 파일을 생성하지 마세요**. 모든 내용을 이 응답에 직접 출력하세요.
- 복사해서 바로 적용할 수 있는 **구체적인 수정 코드**를 반드시 포함하세요.
- "계획 파일에 작성했습니다" 같은 문구 없이 모든 내용을 여기에 출력하세요.`,
		issueKey, mdPath)
}

// AnalyzeAndGeneratePlan은 Phase 1: 읽기 전용 분석을 실행하고 _plan.md를 생성한다.
// 기존 AnalyzeIssue와 유사하지만, 결과를 Jira 컨텍스트 + 분석 결과 + 실행 지시사항으로
// 구조화된 plan 파일로 조립한다.
func (c *ClaudeCodeAdapter) AnalyzeAndGeneratePlan(mdFilePath, prompt, workDir string) (*PlanResult, error) {
	defer logger.DebugFunc("AnalyzeAndGeneratePlan")()
	logger.Debug("AnalyzeAndGeneratePlan: mdPath=%s, workDir=%s", mdFilePath, workDir)

	if !c.enabled {
		logger.Debug("AnalyzeAndGeneratePlan: Claude integration is not enabled")
		return nil, fmt.Errorf("Claude integration is not enabled")
	}

	effectiveDir, err := resolveWorkDir(workDir)
	if err != nil {
		logger.Debug("AnalyzeAndGeneratePlan: resolveWorkDir failed: %v", err)
		return nil, err
	}
	logger.Debug("AnalyzeAndGeneratePlan: effectiveDir=%s", effectiveDir)

	fmt.Printf("[Claude] Phase 1: 분석 및 계획 생성 시작...\n")
	fmt.Printf("[Claude] CLI Path: %s\n", c.cliPath)
	fmt.Printf("[Claude] Work Dir: %s\n", effectiveDir)
	fmt.Printf("[Claude] MD File: %s\n", mdFilePath)

	// 마크다운 파일 읽기
	mdContent, err := os.ReadFile(mdFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MD file: %w", err)
	}

	// 파일 경로 설정
	basePath := strings.TrimSuffix(mdFilePath, ".md")
	planPath := basePath + "_plan.md"
	promptFile := basePath + "_plan_prompt.txt"
	scriptPath := basePath + "_plan_run.sh"
	logFile := basePath + "_plan_log.txt"

	// 프롬프트 파일 작성
	fullPrompt := fmt.Sprintf("%s\n\n---\n%s", prompt, string(mdContent))
	if err := os.WriteFile(promptFile, []byte(fullPrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to write prompt file: %w", err)
	}

	// 래퍼 스크립트 생성: Claude 실행 → 결과를 plan 파일로 조립
	scriptContent := fmt.Sprintf(`#!/bin/bash
exec > "%s" 2>&1
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Phase 1: 분석 및 계획 생성 시작..."
echo "Working directory: %s"
cd "%s"
echo "Prompt file: %s"
echo "Plan file: %s"
echo ""
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Running Claude (Phase 1 - 분석)..."
%s --model %s --print "$(cat '%s')" --output-format text > /tmp/claude_plan_$$.txt 2>&1
CLAUDE_EXIT=$?
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Claude exited with code: $CLAUDE_EXIT"
echo "Output size: $(wc -c < /tmp/claude_plan_$$.txt) bytes"
echo ""
echo "=== Claude Output ==="
cat /tmp/claude_plan_$$.txt
echo "=== End Output ==="
echo ""
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Building plan file..."

# plan 파일 헤더 작성
cat > "%s" << 'PLAN_HEADER'
# Claude Code 실행 계획

> 이 파일은 Claude Code에 직접 전달하여 자동 수정을 실행할 수 있는 구조화된 계획입니다.
> 아래 "실행 지시사항" 섹션의 지침에 따라 코드를 수정하세요.

PLAN_HEADER

# Jira 이슈 컨텍스트 추가
echo "## Jira 이슈 컨텍스트" >> "%s"
echo "" >> "%s"
cat "%s" >> "%s"
echo "" >> "%s"
echo "---" >> "%s"
echo "" >> "%s"

# AI 분석 결과 추가
echo "## AI 분석 결과" >> "%s"
echo "" >> "%s"
echo "생성 시간: $(date '+%%Y-%%m-%%d %%H:%%M:%%S')" >> "%s"
echo "프로젝트: %s" >> "%s"
echo "" >> "%s"
if [ $CLAUDE_EXIT -ne 0 ]; then
    echo "⚠️ Claude 분석 중 오류 발생 (exit code: $CLAUDE_EXIT)" >> "%s"
    echo "" >> "%s"
fi
# bkit Feature Usage 섹션 제거 (─────로 시작하는 블록)
sed '/^─\{5,\}/,/^─\{5,\}$/d' /tmp/claude_plan_$$.txt >> "%s"
echo "" >> "%s"
echo "---" >> "%s"
echo "" >> "%s"

# 실행 지시사항 추가
cat >> "%s" << 'EXEC_SECTION'

## 실행 지시사항

위 분석 결과를 바탕으로 다음을 수행하세요:

1. **파일 수정**: 위 "FILES_TO_MODIFY" 섹션에서 식별된 파일을 열고, 제시된 수정 코드를 적용하세요.
2. **빌드 확인**: 수정 후 빌드가 성공하는지 확인하세요.
3. **테스트 실행**: 관련 테스트가 있다면 실행하세요.
4. **변경 요약**: 수행한 모든 변경사항을 요약하세요.

### 중요 규칙
- 분석 결과에서 명시한 파일과 코드만 수정하세요.
- 불필요한 리팩토링은 하지 마세요.
- 수정할 수 없는 항목은 이유를 설명하세요.

EXEC_SECTION

echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Plan file created: %s"
rm -f /tmp/claude_plan_$$.txt "%s"
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Phase 1 완료!"
`,
		logFile, effectiveDir, effectiveDir, promptFile, planPath,
		c.cliPath, c.model, promptFile,
		planPath,
		planPath, planPath, mdFilePath, planPath, planPath, planPath, planPath,
		planPath, planPath, planPath, effectiveDir, planPath, planPath,
		planPath, planPath, planPath, planPath, planPath,
		planPath,
		planPath, planPath,
		promptFile)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// 백그라운드 프로세스로 실행
	cmd := exec.Command("nohup", "bash", scriptPath)
	cmd.Dir = effectiveDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start background process: %w", err)
	}

	go func() {
		cmd.Wait()
	}()

	fmt.Printf("[Claude] Phase 1 시작됨 (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("[Claude] Plan 파일: %s\n", planPath)
	fmt.Printf("[Claude] 로그 파일: %s\n", logFile)

	logger.Debug("AnalyzeAndGeneratePlan: completed successfully, PID=%d, planPath=%s", cmd.Process.Pid, planPath)

	return &PlanResult{
		PlanPath:   planPath,
		ScriptPath: scriptPath,
		LogPath:    logFile,
		PID:        cmd.Process.Pid,
	}, nil
}

// ExecutePlan은 Phase 2: plan 파일을 Claude Code에 전달하여 실제 코드 수정을 실행한다.
func (c *ClaudeCodeAdapter) ExecutePlan(planPath, workDir string) (*AnalysisResult, error) {
	defer logger.DebugFunc("ExecutePlan")()
	logger.Debug("ExecutePlan: planPath=%s, workDir=%s", planPath, workDir)

	if !c.enabled {
		logger.Debug("ExecutePlan: Claude integration is not enabled")
		return nil, fmt.Errorf("Claude integration is not enabled")
	}

	effectiveDir, err := resolveWorkDir(workDir)
	if err != nil {
		logger.Debug("ExecutePlan: resolveWorkDir failed: %v", err)
		return nil, err
	}
	logger.Debug("ExecutePlan: effectiveDir=%s", effectiveDir)

	fmt.Printf("[Claude] Phase 2: 계획 실행 시작...\n")
	fmt.Printf("[Claude] Plan File: %s\n", planPath)

	// plan 파일 읽기
	planContent, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	// 파일 경로 설정
	basePath := strings.TrimSuffix(planPath, "_plan.md")
	executionPath := basePath + "_execution.md"
	promptFile := basePath + "_exec_prompt.txt"
	scriptPath := basePath + "_exec_run.sh"
	logFile := basePath + "_exec_log.txt"

	// 프롬프트 파일 작성
	if err := os.WriteFile(promptFile, planContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to write prompt file: %w", err)
	}

	// 래퍼 스크립트 생성
	scriptContent := fmt.Sprintf(`#!/bin/bash
exec > "%s" 2>&1
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Phase 2: 계획 실행 시작..."
echo "Working directory: %s"
cd "%s"
echo "Prompt file: %s"
echo "Output file: %s"
echo ""
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Running Claude (Phase 2 - 실행)..."
%s --model %s --print "$(cat '%s')" --output-format text > /tmp/claude_exec_$$.txt 2>&1
CLAUDE_EXIT=$?
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Claude exited with code: $CLAUDE_EXIT"
echo "Output size: $(wc -c < /tmp/claude_exec_$$.txt) bytes"
echo ""
echo "=== Claude Output ==="
cat /tmp/claude_exec_$$.txt
echo "=== End Output ==="
echo ""
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Writing execution result..."

echo "# 실행 결과" > "%s"
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
cat /tmp/claude_exec_$$.txt >> "%s"
echo "" >> "%s"
echo "---" >> "%s"
echo "" >> "%s"
echo "✅ 실행 완료: $(date '+%%Y-%%m-%%d %%H:%%M:%%S')" >> "%s"

rm -f /tmp/claude_exec_$$.txt "%s" "%s"
echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] Phase 2 완료!"
`,
		logFile, effectiveDir, effectiveDir, promptFile, executionPath,
		c.cliPath, c.model, promptFile,
		executionPath, executionPath, executionPath, effectiveDir, executionPath,
		executionPath, executionPath, executionPath,
		executionPath, executionPath,
		executionPath, executionPath, executionPath, executionPath, executionPath,
		promptFile, scriptPath)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// 백그라운드 프로세스로 실행
	cmd := exec.Command("nohup", "bash", scriptPath)
	cmd.Dir = effectiveDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start background process: %w", err)
	}

	go func() {
		cmd.Wait()
	}()

	fmt.Printf("[Claude] Phase 2 시작됨 (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("[Claude] 실행 결과: %s\n", executionPath)

	logger.Debug("ExecutePlan: completed successfully, PID=%d, executionPath=%s", cmd.Process.Pid, executionPath)

	return &AnalysisResult{
		OutputPath: executionPath,
		ScriptPath: scriptPath,
		PID:        cmd.Process.Pid,
	}, nil
}

// SendPlanToClaudeAsync는 Phase 1 분석을 비동기적으로 실행한다.
func (c *ClaudeCodeAdapter) SendPlanToClaudeAsync(mdFilePath, prompt, workDir string, onComplete func(*PlanResult, error)) {
	go func() {
		result, err := c.AnalyzeAndGeneratePlan(mdFilePath, prompt, workDir)
		if onComplete != nil {
			onComplete(result, err)
		}
	}()
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
