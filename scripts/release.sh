#!/bin/bash
# release.sh - 프로덕션 배포용 빌드

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

VERSION=${1:-"1.0.0"}
APP_NAME="JiraAIGenerator"
OUTPUT_DIR="${PROJECT_DIR}/dist"

echo "🚀 Release 빌드 시작 (v${VERSION})..."

# 클린
echo "🧹 이전 빌드 정리 중..."
"$SCRIPT_DIR/clean.sh"

cd "$PROJECT_DIR"

# 출력 디렉터리 생성
mkdir -p "$OUTPUT_DIR"

# macOS 빌드 (Apple Silicon)
echo "🍎 macOS (arm64) 빌드 중..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${APP_NAME}_darwin_arm64" ./cmd/app

# macOS 빌드 (Intel)
echo "🍎 macOS (amd64) 빌드 중..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${APP_NAME}_darwin_amd64" ./cmd/app

# 유니버설 바이너리 생성 (Apple Silicon + Intel)
echo "🔗 유니버설 바이너리 생성 중..."
lipo -create -output "${OUTPUT_DIR}/${APP_NAME}" \
    "${OUTPUT_DIR}/${APP_NAME}_darwin_arm64" \
    "${OUTPUT_DIR}/${APP_NAME}_darwin_amd64"

# 개별 아키텍처 바이너리 삭제 (선택)
# rm "${OUTPUT_DIR}/${APP_NAME}_darwin_arm64"
# rm "${OUTPUT_DIR}/${APP_NAME}_darwin_amd64"

echo "✅ Release 빌드 완료!"
echo "📁 출력 위치: ${OUTPUT_DIR}/"
ls -lh "${OUTPUT_DIR}/"
