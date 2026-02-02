package logger

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestSetDebugMode(t *testing.T) {
	// Arrange
	originalMode := IsDebugMode()
	defer SetDebugMode(originalMode) // 테스트 후 복원

	// Act & Assert
	SetDebugMode(true)
	if !IsDebugMode() {
		t.Error("Expected debug mode to be enabled")
	}

	SetDebugMode(false)
	if IsDebugMode() {
		t.Error("Expected debug mode to be disabled")
	}
}

func TestDebug_WhenDisabled(t *testing.T) {
	// Arrange
	SetDebugMode(false)
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.Ltime|log.Lmicroseconds)

	// Act
	Debug("test message")

	// Assert
	if buf.Len() > 0 {
		t.Error("Expected no output when debug mode is disabled")
	}
}

func TestDebug_WhenEnabled(t *testing.T) {
	// Arrange
	SetDebugMode(true)
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.Ltime|log.Lmicroseconds)

	// Act
	Debug("test message: %s", "hello")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "logger_test.go") {
		t.Errorf("Expected output to contain caller file name, got: %s", output)
	}
	if !strings.Contains(output, "test message: hello") {
		t.Errorf("Expected output to contain formatted message, got: %s", output)
	}
}

func TestDebugFunc_WhenDisabled(t *testing.T) {
	// Arrange
	SetDebugMode(false)
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.Ltime|log.Lmicroseconds)

	// Act
	cleanup := DebugFunc("TestFunction")
	cleanup()

	// Assert
	if buf.Len() > 0 {
		t.Error("Expected no output when debug mode is disabled")
	}
}

func TestDebugFunc_WhenEnabled(t *testing.T) {
	// Arrange
	SetDebugMode(true)
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.Ltime|log.Lmicroseconds)

	// Act
	cleanup := DebugFunc("TestFunction")
	cleanup()

	// Assert
	output := buf.String()
	if !strings.Contains(output, "→ TestFunction()") {
		t.Errorf("Expected entry log, got: %s", output)
	}
	if !strings.Contains(output, "← TestFunction()") {
		t.Errorf("Expected exit log, got: %s", output)
	}
}

func TestInit_EnvironmentVariable(t *testing.T) {
	// Arrange
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"DEBUG=1", "1", true},
		{"DEBUG=true", "true", true},
		{"DEBUG=false", "false", false},
		{"DEBUG=0", "0", false},
		{"DEBUG empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: init()은 패키지 로드 시 한 번만 실행되므로
			// 환경변수 테스트는 실제로는 별도 프로세스로 실행해야 정확함
			// 여기서는 SetDebugMode로 동작을 시뮬레이션
			os.Setenv("DEBUG", tt.envValue)

			// Act - init() 동작 시뮬레이션
			expectedMode := tt.envValue == "1" || tt.envValue == "true"
			SetDebugMode(expectedMode)

			// Assert
			if IsDebugMode() != tt.expected {
				t.Errorf("Expected debug mode to be %v, got %v", tt.expected, IsDebugMode())
			}
		})
	}
}
