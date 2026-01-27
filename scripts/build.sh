#!/bin/bash
# build.sh - 앱 빌드 스크립트

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🔨 Jira AI Generator 빌드 시작..."

# 클린
echo "🧹 이전 빌드 정리 중..."
"$SCRIPT_DIR/clean.sh"

# 의존성 정리
echo "📦 의존성 정리 중..."
cd "$PROJECT_DIR"
go mod tidy

# 빌드
echo "🏗️ 빌드 중..."
go build -o jira-ai-generator ./cmd/app

echo "✅ 빌드 완료: ./jira-ai-generator"
