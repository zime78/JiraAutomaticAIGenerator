package main

import (
	"fmt"
	"log"
	"os"

	"jira-ai-generator/internal/config"
	"jira-ai-generator/internal/ui"
)

func main() {
	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		// Try to create default config if not exists
		fmt.Println("설정 파일을 찾을 수 없습니다.")
		fmt.Println("config.ini.example 파일을 config.ini로 복사하고 설정을 입력해주세요.")
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("설정 오류: %v", err)
	}

	// Create and run application
	app := ui.NewApp(cfg)
	app.Run()
}
