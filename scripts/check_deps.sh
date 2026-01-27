#!/bin/bash
# check_deps.sh - 의존성 확인 스크립트

echo "🔍 시스템 의존성 확인 중..."
echo ""

# Go 버전
echo "📦 Go:"
if command -v go &> /dev/null; then
    go version
else
    echo "  ❌ Go가 설치되어 있지 않습니다. (https://go.dev/dl/)"
fi
echo ""

# Xcode CLI Tools
echo "🛠️ Xcode CLI Tools:"
if xcode-select -p &> /dev/null; then
    echo "  ✅ 설치됨: $(xcode-select -p)"
else
    echo "  ❌ 설치되지 않음. 실행: xcode-select --install"
fi
echo ""

# ffmpeg (선택)
echo "🎬 ffmpeg (동영상 프레임 추출용):"
if command -v ffmpeg &> /dev/null; then
    ffmpeg -version | head -1
else
    echo "  ⚠️ 설치되지 않음 (선택사항). 설치: brew install ffmpeg"
fi
echo ""

# config.ini 확인
echo "⚙️ config.ini:"
if [ -f "config.ini" ]; then
    echo "  ✅ 존재함"
else
    echo "  ❌ 없음. 실행: cp config.ini.example config.ini"
fi
echo ""

echo "✅ 의존성 확인 완료"
