# JiraAutomaticAIGenerator - Architecture Document

> Jira 이슈 자동 분석 및 Claude AI 연동 macOS GUI + CLI 애플리케이션

## Overview

| 항목 | 내용 |
|------|------|
| **언어** | Go 1.21+ |
| **GUI** | Fyne v2.4.0 |
| **아키텍처** | Clean Architecture |
| **코드량** | 7,351줄 (35개 Go 파일) |

```
┌─────────────────────────────────────────────────────────────┐
│                    JiraAutomaticAIGenerator                 │
├─────────────────────────────────────────────────────────────┤
│  Jira URL 입력 → 이슈 조회 → 첨부파일 다운로드 → 프레임 추출  │
│  → 마크다운 생성 → Claude AI 분석 (백그라운드)               │
└─────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
JiraAutomaticAIGenerator/
├── cmd/
│   ├── app/                    # GUI 진입점 (Fyne)
│   │   └── main.go
│   └── cli/                    # CLI 진입점 (터미널)
│       └── main.go
│
├── internal/                   # 핵심 비즈니스 로직
│   ├── domain/                 # 엔티티 정의
│   ├── port/                   # 인터페이스 정의
│   ├── usecase/                # 비즈니스 로직
│   ├── adapter/                # 외부 시스템 구현
│   ├── config/                 # 설정 관리
│   ├── mock/                   # 테스트용 Mock
│   └── ui/                     # Fyne GUI
│       ├── state/              # 상태 관리
│       ├── components/         # UI 컴포넌트
│       └── utils/              # UI 유틸리티
│
├── scripts/                    # 빌드/배포 스크립트
├── docs/                       # 문서
├── output/                     # 생성된 결과물
└── config.ini                  # 설정 파일
```

---

## Clean Architecture Layers

```
┌──────────────────────────────┐ ┌────────────────────────────┐
│          UI Layer            │ │         CLI Layer           │
│  internal/ui/ (Fyne GUI)     │ │  cmd/cli/ (터미널 실행)     │
└──────────────┬───────────────┘ └─────────────┬──────────────┘
               └──────────────┬────────────────┘
                           │ 의존
┌──────────────────────────▼──────────────────────────────────┐
│                      UseCase Layer                           │
│    internal/usecase/ (ProcessIssueUseCase)                  │
└──────────────────────────┬──────────────────────────────────┘
                           │ 의존
┌──────────────────────────▼──────────────────────────────────┐
│                       Port Layer                             │
│    internal/port/ (인터페이스 정의)                          │
└──────────────────────────┬──────────────────────────────────┘
                           │ 구현
┌──────────────────────────▼──────────────────────────────────┐
│                     Adapter Layer                            │
│    internal/adapter/ (Jira, FFmpeg, Claude 등)              │
└─────────────────────────────────────────────────────────────┘
                           ▲
┌──────────────────────────┴──────────────────────────────────┐
│                      Domain Layer                            │
│    internal/domain/ (JiraIssue, Attachment 등)              │
└─────────────────────────────────────────────────────────────┘
```

**의존성 규칙**: 항상 안쪽(Domain)으로만 의존

---

## Layer Details

### 1. Domain Layer (`internal/domain/`)

비즈니스 엔티티 정의

| 구조체 | 설명 |
|--------|------|
| `JiraIssue` | Jira 이슈 (Key, Summary, Description, Attachments) |
| `Attachment` | 첨부파일 메타데이터 |
| `GeneratedDocument` | 생성된 마크다운 문서 |
| `ProcessResult` | 처리 결과 |
| `DownloadResult` | 다운로드 결과 |

### 2. Port Layer (`internal/port/`)

외부 의존성 인터페이스 정의

```go
type JiraRepository interface {
    GetIssue(issueKey string) (*domain.JiraIssue, error)
    DownloadAttachment(url string) ([]byte, error)
}

type AttachmentDownloader interface {
    DownloadAll(issueKey string, attachments []domain.Attachment) ([]domain.DownloadResult, error)
}

type VideoProcessor interface {
    IsAvailable() bool
    ExtractFrames(videoPath, outputDir string, interval float64, maxFrames int) ([]string, error)
}

type DocumentGenerator interface {
    Generate(issue *domain.JiraIssue, imagePaths, framePaths []string, outputDir string) (*domain.GeneratedDocument, error)
    SaveToFile(doc *domain.GeneratedDocument) (string, error)
}

type Clipboard interface {
    SetContent(content string)
}
```

### 3. UseCase Layer (`internal/usecase/`)

비즈니스 로직 구현

```go
type ProcessIssueUseCase struct {
    jiraRepo       port.JiraRepository
    downloader     port.AttachmentDownloader
    videoProcessor port.VideoProcessor
    docGenerator   port.DocumentGenerator
    outputDir      string
}

// Execute: 6단계 워크플로우
// 1. 이슈 조회 (0.1)
// 2. 첨부파일 다운로드 (0.3)
// 3. 동영상 프레임 추출 (0.5)
// 4. 문서 생성 (0.8)
// 5. 파일 저장 (0.9)
// 6. 완료 (1.0)
```

### 4. Adapter Layer (`internal/adapter/`)

외부 시스템 구현체

| Adapter | 역할 | Port 구현 |
|---------|------|-----------|
| `JiraClient` | Jira REST API v3 | JiraRepository |
| `AttachmentDownloader` | 파일 다운로드 | AttachmentDownloader |
| `FFmpegVideoProcessor` | 프레임 추출 | VideoProcessor |
| `MarkdownGenerator` | 마크다운 생성 | DocumentGenerator |
| `ClaudeCodeAdapter` | Claude CLI 연동 | (직접 사용) |

### 5. UI Layer (`internal/ui/`)

Fyne GUI 구현

```
ui/
├── app.go              # App 구조체, NewApp()
├── app_ui.go           # V1 UI (레거시)
├── app_ui_v2.go        # V2 UI (컴포넌트 기반)
├── app_handlers.go     # 이벤트 핸들러
├── app_queue.go        # 단일 큐 관리
├── app_analysis.go     # Claude 연동
├── theme.go            # 한글 테마
│
├── state/              # 상태 관리
│   ├── app_state.go    # AppState
│   ├── event_bus.go    # EventBus (발행-구독)
│   └── progress_callback.go
│
├── components/         # UI 컴포넌트
│   ├── sidebar.go
│   ├── progress_panel.go
│   ├── result_panel.go
│   ├── markdown_viewer.go
│   ├── log_viewer.go
│   └── status_bar.go
│
└── utils/              # 유틸리티
    ├── markdown_parser.go
    └── notifications.go
```

---

## Single Workspace Architecture

```
┌────────────────────────────────────────┐
│           단일 워크스페이스              │
├────────────────────────────────────────┤
│ URL 입력 / 프로젝트 경로               │
│ 분석 시작 / 중지 버튼                  │
│ 진행률 패널                            │
│ 결과 패널 (이슈 정보 + AI 분석)        │
│ 대기 큐                                │
└────────────────────────────────────────┘
                    │
     ┌──────────────▼──────────────┐
     │       공유 항목              │
     ├─────────────────────────────┤
     │ - completedJobs (완료 이력)  │
     │ - ProcessIssueUseCase       │
     │ - ClaudeCodeAdapter         │
     │ - SQLiteRepository          │
     └─────────────────────────────┘
```

---

## Data Flow

```
┌─────────────────────────┐  ┌────────────────────────────────┐
│  GUI: 사용자 입력 (URL)  │  │  CLI: jira-ai-cli <URL/Key>   │
└────────────┬────────────┘  └───────────────┬────────────────┘
             └───────────────┬───────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              ProcessIssueUseCase.Execute()                  │
├─────────────────────────────────────────────────────────────┤
│ 1. JiraClient.GetIssue()           → JiraIssue             │
│ 2. AttachmentDownloader.DownloadAll() → 로컬 파일          │
│ 3. FFmpegVideoProcessor.ExtractFrames() → 프레임 이미지    │
│ 4. MarkdownGenerator.Generate()     → GeneratedDocument    │
│ 5. MarkdownGenerator.SaveToFile()   → {IssueKey}.md        │
└──────────────────────────┬──────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              Claude Code 연동 (백그라운드)                   │
├─────────────────────────────────────────────────────────────┤
│ Phase 1: AnalyzeAndGeneratePlan()                           │
│   → nohup bash → Claude CLI → {IssueKey}_plan.md           │
│                                                             │
│ Phase 2: ExecutePlan()                                      │
│   → nohup bash → Claude CLI → {IssueKey}_execution.md      │
└──────────────────────────┬──────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    완료 이력 저장                            │
│                  (completedJobs 목록)                       │
└─────────────────────────────────────────────────────────────┘
```

---

## Event-Based State Management

```go
// EventBus: 발행-구독 패턴
type EventBus struct {
    subscribers map[EventType][]func(interface{})
}

// 이벤트 타입
const (
    EventProgressUpdate  // 진행률 업데이트
    EventPhaseChange     // 단계 변경
    EventJobStarted      // 작업 시작
    EventJobCompleted    // 작업 완료
    EventJobFailed       // 작업 실패
    EventChannelSwitch   // 채널 전환
    EventQueueUpdated    // 큐 업데이트
    EventHistoryAdded    // 이력 추가
)

// 사용 예시
eventBus.Subscribe(EventProgressUpdate, func(data interface{}) {
    progress := data.(float64)
    progressBar.SetValue(progress)
})

eventBus.Publish(EventProgressUpdate, 0.5)
```

---

## File Generation Pattern

```
output/
└── {IssueKey}/
    ├── {IssueKey}.md                    # 원본 문서
    ├── {IssueKey}_plan.md               # Phase 1: 분석 계획
    ├── {IssueKey}_plan_prompt.txt       # (임시)
    ├── {IssueKey}_plan_log.txt          # 로그
    ├── {IssueKey}_execution.md          # Phase 2: 실행 결과
    ├── {IssueKey}_exec_prompt.txt       # (임시)
    ├── {IssueKey}_exec_log.txt          # 로그
    ├── image1.png, video.mp4, ...       # 첨부파일
    └── frames/
        └── video_frame_0001.png, ...    # 추출된 프레임
```

---

## Dependency Injection

```go
// App이 DI 컨테이너 역할
func NewApp(cfg *config.Config) *App {
    // Adapter 생성
    jiraClient := adapter.NewJiraClient(cfg.Jira.URL, cfg.Jira.Email, cfg.Jira.APIKey)
    docGenerator := adapter.NewMarkdownGenerator(cfg.AI.PromptTemplate)
    claudeAdapter := adapter.NewClaudeCodeAdapter(cfg.Claude.CLIPath, cfg.Claude.Enabled)
    videoProcessor := adapter.NewFFmpegVideoProcessor()
    downloader := adapter.NewAttachmentDownloader(jiraClient, cfg.Output.Dir)

    // UseCase 생성 (Port 인터페이스로 주입)
    processIssueUC := usecase.NewProcessIssueUseCase(
        jiraClient,      // JiraRepository
        downloader,      // AttachmentDownloader
        videoProcessor,  // VideoProcessor
        docGenerator,    // DocumentGenerator
        cfg.Output.Dir,
    )

    return &App{
        processIssueUC: processIssueUC,
        claudeAdapter:  claudeAdapter,
        // ...
    }
}
```

---

## Configuration

```ini
# config.ini
[jira]
url = https://example.atlassian.net
email = user@example.com
api_key = YOUR_API_KEY

[output]
dir = ./output

[ai]
prompt_template = 다음 Jira 이슈를 분석하고 수정 코드를 작성해주세요:

[claude]
cli_path = claude
project_path_1 = /path/to/project   # 프로젝트 경로 (필수)
hook_script_path = /path/to/hook.sh  # Hook 스크립트 (필수)
enabled = true
```

---

## System Dependencies

| 구분 | 의존성 |
|------|--------|
| **필수** | Go 1.21+, Xcode CLI Tools |
| **선택** | ffmpeg (동영상), Claude Code CLI (AI) |

---

## Extension Points

### 새로운 외부 서비스 추가

```go
// 1. Port 인터페이스 정의
type NewServicePort interface {
    DoSomething() error
}

// 2. Adapter 구현
type NewServiceAdapter struct { ... }

// 3. UseCase에서 주입받아 사용
// 4. App.NewApp()에서 인스턴스 생성
```

### UI 컴포넌트 추가

```go
// 1. components/에 새 파일 생성
// 2. Fyne 위젯 조합
// 3. EventBus로 상태 동기화
// 4. app_ui_v2.go에서 호출
```

---

## Key Design Patterns

| 패턴 | 적용 위치 |
|------|----------|
| **Clean Architecture** | 전체 구조 |
| **Dependency Injection** | App.NewApp() |
| **Repository Pattern** | JiraRepository |
| **Observer Pattern** | EventBus |
| **Strategy Pattern** | ProgressReporter |
| **Adapter Pattern** | 모든 Adapter |

---

## Code Statistics

| 패키지 | 파일 | 라인 수 |
|--------|------|--------|
| adapter | 6 | ~2,500 |
| ui | 12 | ~2,800 |
| usecase | 3 | ~500 |
| domain | 2 | ~200 |
| port | 1 | ~150 |
| config | 1 | ~200 |
| mock | 1 | ~300 |
| **합계** | **35** | **~7,350** |
