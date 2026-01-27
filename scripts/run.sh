#!/bin/bash
# run.sh - 개발 모드 실행 (디버깅용)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# 한글 폰트 설정 (macOS)
if [ -f "/System/Library/Fonts/Supplemental/Arial Unicode.ttf" ]; then
    export FYNE_FONT="/System/Library/Fonts/Supplemental/Arial Unicode.ttf"
elif [ -f "/Library/Fonts/Arial Unicode.ttf" ]; then
    export FYNE_FONT="/Library/Fonts/Arial Unicode.ttf"
fi

echo "🚀 개발 모드로 실행 중..."
echo "   FYNE_FONT: ${FYNE_FONT:-기본값}"
go run ./cmd/app
