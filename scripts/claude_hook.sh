#!/usr/bin/env bash

# claude_hook.sh는 Claude Code Hook에서 호출되는 프로젝트 전용 검증 훅이다.
# 실패 시 비정상 종료 코드를 반환하여 상위 실행기를 즉시 실패 처리하도록 한다.
set -euo pipefail

LOG_FILE="/tmp/jira_ai_generator_claude_hook.log"
NOW="$(date '+%Y-%m-%d %H:%M:%S')"

{
  echo "[$NOW] hook invoked"
  echo "  cwd: $(pwd)"
  echo "  tool: ${CLAUDE_TOOL_NAME:-unknown}"
  echo "  file: ${CLAUDE_FILE_PATH:-unknown}"
} >> "$LOG_FILE"

exit 0
