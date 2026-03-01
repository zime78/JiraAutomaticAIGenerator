package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"jira-ai-generator/internal/adapter"
	"jira-ai-generator/internal/config"
	"jira-ai-generator/internal/usecase"
)

// processOne은 단일 이슈 URL 또는 키에 대해 전체 워크플로우를 실행한다.
// Jira 이슈 조회 → 첨부파일 다운로드 → 프레임 추출 → 마크다운 문서 생성
func processOne(
	issueKeyOrURL string,
	uc *usecase.ProcessIssueUseCase,
) error {
	fmt.Printf("\n━━━ 처리 시작: %s ━━━\n", issueKeyOrURL)

	result, err := uc.Execute(issueKeyOrURL, func(progress float64, status string) {
		fmt.Printf("  [%3.0f%%] %s\n", progress*100, status)
	})
	if err != nil {
		return fmt.Errorf("처리 실패: %w", err)
	}

	fmt.Printf("  ✓ 문서 생성 완료: %s\n", result.MDPath)
	return nil
}

// processBatch는 파일에서 URL/키 목록을 읽어 순차적으로 처리한다.
func processBatch(
	filePath string,
	uc *usecase.ProcessIssueUseCase,
) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("배치 파일 열기 실패: %w", err)
	}
	defer f.Close()

	var urls []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 빈 줄, 주석 무시
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("배치 파일 읽기 실패: %w", err)
	}

	if len(urls) == 0 {
		return fmt.Errorf("배치 파일에 처리할 URL이 없습니다: %s", filePath)
	}

	fmt.Printf("총 %d건 처리 예정\n", len(urls))

	successCount := 0
	failCount := 0

	for i, url := range urls {
		fmt.Printf("\n[%d/%d] ", i+1, len(urls))
		if err := processOne(url, uc); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ 실패: %v\n", err)
			failCount++
		} else {
			successCount++
		}
	}

	fmt.Printf("\n━━━ 배치 처리 완료 ━━━\n")
	fmt.Printf("  성공: %d건\n", successCount)
	fmt.Printf("  실패: %d건\n", failCount)
	fmt.Printf("  합계: %d건\n", len(urls))

	if failCount > 0 {
		return fmt.Errorf("%d건 처리 실패", failCount)
	}
	return nil
}

func main() {
	// 플래그 정의
	batchFile := flag.String("batch", "", "URL 목록 파일로 일괄 처리 (한 줄에 하나)")
	outputDir := flag.String("output", "", "출력 디렉토리 (config-cli.ini의 output.dir 대신 CLI에서 지정)")
	configPath := flag.String("config", "", "설정 파일 경로 (기본: config-cli.ini → ~/.jira-ai-generator/config-cli.ini)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Jira AI Generator CLI\n\n")
		fmt.Fprintf(os.Stderr, "사용법:\n")
		fmt.Fprintf(os.Stderr, "  jira-ai-cli [옵션] <URL 또는 이슈키>\n")
		fmt.Fprintf(os.Stderr, "  jira-ai-cli [옵션] --batch <파일경로>\n\n")
		fmt.Fprintf(os.Stderr, "예시:\n")
		fmt.Fprintf(os.Stderr, "  jira-ai-cli https://example.atlassian.net/browse/PROJ-123\n")
		fmt.Fprintf(os.Stderr, "  jira-ai-cli PROJ-123\n")
		fmt.Fprintf(os.Stderr, "  jira-ai-cli --batch urls.txt\n")
		fmt.Fprintf(os.Stderr, "  jira-ai-cli --output ./my-output PROJ-123\n\n")
		fmt.Fprintf(os.Stderr, "옵션:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// 인자 검증: URL/키 또는 --batch 중 하나 필요
	if *batchFile == "" && flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// 설정 파일 로드
	var cfg *config.Config
	var err error
	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadDefaultCLI()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "설정 파일 로드 실패: %v\n", err)
		fmt.Fprintln(os.Stderr, "config-cli.ini 파일을 확인해주세요. (--config 옵션으로 경로 지정 가능)")
		os.Exit(1)
	}

	// CLI 플래그로 설정 오버라이드
	if *outputDir != "" {
		cfg.Output.Dir = *outputDir
	}

	// 설정 검증
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "설정 오류: %v\n", err)
		os.Exit(1)
	}

	// 출력 디렉토리 생성
	if err := os.MkdirAll(cfg.Output.Dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "출력 디렉토리 생성 실패: %v\n", err)
		os.Exit(1)
	}

	// 어댑터 생성
	jiraClient := adapter.NewJiraClient(cfg.Jira.URL, cfg.Jira.Email, cfg.Jira.APIKey)
	docGenerator := adapter.NewMarkdownGenerator(cfg.AI.PromptTemplate)
	videoProcessor := adapter.NewFFmpegVideoProcessor()
	downloader := adapter.NewAttachmentDownloader(jiraClient, cfg.Output.Dir)

	// UseCase 생성
	processIssueUC := usecase.NewProcessIssueUseCase(jiraClient, downloader, videoProcessor, docGenerator, cfg.Output.Dir)

	fmt.Println("Jira AI Generator CLI")
	fmt.Printf("출력 디렉토리: %s\n", cfg.Output.Dir)

	// 실행
	if *batchFile != "" {
		// 배치 모드
		if err := processBatch(*batchFile, processIssueUC); err != nil {
			fmt.Fprintf(os.Stderr, "\n오류: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 단일 URL 모드
		issueKeyOrURL := flag.Arg(0)
		if err := processOne(issueKeyOrURL, processIssueUC); err != nil {
			fmt.Fprintf(os.Stderr, "\n오류: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\n완료!")
}
