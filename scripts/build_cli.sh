#!/bin/bash
# build_cli.sh - CLI 단독 빌드 스크립트

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

OUTPUT_DIR="${PROJECT_DIR}/dist"

echo "⌨️  Jira AI CLI 빌드 시작..."

# 의존성 정리
echo "📦 의존성 정리 중..."
cd "$PROJECT_DIR"
go mod tidy

# 출력 디렉터리 생성
mkdir -p "$OUTPUT_DIR"

# CLI 빌드 (Fyne 의존성 없으므로 CGO 불필요)
echo "🏗️ CLI 빌드 중..."
CGO_ENABLED=0 go build -o "${OUTPUT_DIR}/jira-ai-cli" ./cmd/cli

echo "✅ 빌드 완료!"
echo "📁 출력 위치: ${OUTPUT_DIR}/jira-ai-cli"
