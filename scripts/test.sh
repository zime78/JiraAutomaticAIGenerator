#!/bin/bash
# test.sh - 테스트 실행 스크립트

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "🧪 테스트 실행 중..."
echo ""

# 기본 테스트 실행
if [ "$1" == "-v" ]; then
    go test -v ./...
elif [ "$1" == "-cover" ]; then
    echo "📊 커버리지 측정 중..."
    go test -cover ./...
elif [ "$1" == "-coverprofile" ]; then
    echo "📊 커버리지 리포트 생성 중..."
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    echo "✅ 커버리지 리포트: coverage.html"
else
    go test ./...
fi

echo ""
echo "✅ 테스트 완료"
