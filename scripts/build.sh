#!/bin/bash
# build.sh - 앱 빌드 스크립트

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

OUTPUT_DIR="${PROJECT_DIR}/dist"

echo "🔨 Jira AI Generator 빌드 시작..."

# 클린
echo "🧹 이전 빌드 정리 중..."
"$SCRIPT_DIR/clean.sh"

# 의존성 정리
echo "📦 의존성 정리 중..."
cd "$PROJECT_DIR"
go mod tidy

# 출력 디렉터리 생성
mkdir -p "$OUTPUT_DIR"

# GUI 빌드
echo "🏗️ GUI 빌드 중..."
go build -o "${OUTPUT_DIR}/jira-ai-generator" ./cmd/app

# CLI 빌드
echo "🏗️ CLI 빌드 중..."
CGO_ENABLED=0 go build -o "${OUTPUT_DIR}/jira-ai-cli" ./cmd/cli

echo "✅ 빌드 완료!"
echo "📁 출력 위치: ${OUTPUT_DIR}/"
ls -lh "${OUTPUT_DIR}/"
