package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
)

var (
	debugMode   = false
	debugLogger *log.Logger
)

func init() {
	// DEBUG 환경변수 또는 -tags debug로 활성화
	if os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true" {
		debugMode = true
	}
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.Ltime|log.Lmicroseconds)
}

// SetDebugMode enables/disables debug logging
func SetDebugMode(enabled bool) {
	debugMode = enabled
}

// IsDebugMode returns current debug mode status
func IsDebugMode() bool {
	return debugMode
}

// Debug logs a debug message with caller info
func Debug(format string, args ...interface{}) {
	if !debugMode {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	// 파일 경로에서 파일명만 추출
	parts := strings.Split(file, "/")
	shortFile := parts[len(parts)-1]
	msg := fmt.Sprintf(format, args...)
	debugLogger.Printf("%s:%d - %s", shortFile, line, msg)
}

// DebugFunc logs function entry/exit
func DebugFunc(name string) func() {
	if !debugMode {
		return func() {}
	}
	Debug("→ %s()", name)
	return func() {
		Debug("← %s()", name)
	}
}
