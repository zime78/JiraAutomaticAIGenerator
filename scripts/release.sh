#!/bin/bash
# release.sh - 프로덕션 배포용 빌드

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

VERSION=${1:-"1.0.0"}
APP_NAME="JiraAIGenerator"
CLI_NAME="JiraAICLI"
OUTPUT_DIR="${PROJECT_DIR}/dist"

echo "🚀 Release 빌드 시작 (v${VERSION})..."

# 클린
echo "🧹 이전 빌드 정리 중..."
"$SCRIPT_DIR/clean.sh"

cd "$PROJECT_DIR"

# 출력 디렉터리 생성
mkdir -p "$OUTPUT_DIR"

# macOS GUI 빌드 (Apple Silicon)
echo "🍎 GUI (Apple Silicon) 빌드 중..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${APP_NAME}_apple" ./cmd/app

# macOS GUI 빌드 (Intel)
echo "🍎 GUI (Intel) 빌드 중..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${APP_NAME}_intel" ./cmd/app

# 유니버설 바이너리 생성 (Apple Silicon + Intel)
echo "🔗 GUI 유니버설 바이너리 생성 중..."
lipo -create -output "${OUTPUT_DIR}/${APP_NAME}_universal" \
    "${OUTPUT_DIR}/${APP_NAME}_apple" \
    "${OUTPUT_DIR}/${APP_NAME}_intel"

# CLI 빌드 (CGO 불필요 — Fyne 의존성 없음)
echo "⌨️  CLI (Apple Silicon) 빌드 중..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${CLI_NAME}_apple" ./cmd/cli

echo "⌨️  CLI (Intel) 빌드 중..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${CLI_NAME}_intel" ./cmd/cli

echo "🔗 CLI 유니버설 바이너리 생성 중..."
lipo -create -output "${OUTPUT_DIR}/${CLI_NAME}_universal" \
    "${OUTPUT_DIR}/${CLI_NAME}_apple" \
    "${OUTPUT_DIR}/${CLI_NAME}_intel"

# Linux CLI 빌드
echo "🐧 CLI (Linux) 빌드 중..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${CLI_NAME}_linux" ./cmd/cli

echo "✅ Release 빌드 완료!"
echo "📁 출력 위치: ${OUTPUT_DIR}/"
ls -lh "${OUTPUT_DIR}/"
