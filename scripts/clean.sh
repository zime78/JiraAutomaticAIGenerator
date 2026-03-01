#!/bin/bash
# clean.sh - 빌드 산출물 정리

set -e

echo "🧹 정리 중..."

# 바이너리 삭제
rm -f jira-ai-generator
rm -f jira-ai-cli
rm -rf dist/

# 출력 디렉터리 정리 (선택)
# rm -rf output/

echo "✅ 정리 완료"
