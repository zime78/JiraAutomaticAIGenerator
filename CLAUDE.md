# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 빌드 및 실행 명령어

```bash
./scripts/test.sh                # 전체 테스트
./scripts/test.sh -v             # 상세 테스트
./scripts/test.sh -cover         # 커버리지 포함
./scripts/test.sh -coverprofile  # 커버리지 프로파일 생성
go test ./internal/usecase/...   # 특정 패키지 테스트
go test -run TestProcessIssue_Success ./internal/usecase/  # 단일 테스트

./scripts/build.sh         # 빌드 (clean → tidy → build)
./scripts/run.sh           # 개발 모드 실행 (FYNE_FONT 한글 설정 포함)
./scripts/release.sh 1.0.0 # 배포 빌드 (macOS/Linux 크로스 컴파일)
./scripts/clean.sh         # 빌드 산출물 정리
./scripts/check_deps.sh    # 시스템 의존성 확인 (Go, Xcode, ffmpeg)
```

## 아키텍처

Go 1.21 + Fyne GUI 앱. Clean Architecture (의존성은 항상 안쪽으로):

```
UI (internal/ui/) → UseCase (internal/usecase/) → Port (internal/port/) ← Adapter (internal/adapter/)
                                                    ↑
                                                  Domain (internal/domain/)
```

### 핵심 워크플로우

Jira URL 입력 → `ProcessIssueUseCase.Execute()` → Jira 이슈 조회 → 첨부파일 다운로드 → 동영상 프레임 추출(FFmpeg) → 마크다운 문서 생성 → Claude Code CLI로 백그라운드 AI 분석

### 의존성 주입

`ui/app.go`의 `NewApp()`에서 모든 Adapter를 생성:

- **Port를 통한 주입**: `JiraRepository`, `AttachmentDownloader`, `VideoProcessor`, `DocumentGenerator` → `ProcessIssueUseCase`에 인터페이스로 주입
- **App에서 직접 사용**: `MarkdownGenerator`(문서 클립보드 복사), `ClaudeCodeAdapter`(AI 분석) → Port 인터페이스 없이 App 필드로 직접 보유

### Port 인터페이스 (internal/port/interfaces.go)

`JiraRepository`, `AttachmentDownloader`, `VideoProcessor`, `DocumentGenerator`, `Clipboard` - 5개 인터페이스. 새 외부 서비스 연동 시 Port 인터페이스 정의 → Adapter 구현 → UseCase에서 주입받아 사용.

> **참고**: `ClaudeCodeAdapter`는 현재 Port 인터페이스 없이 UI에서 직접 사용 중.

### UI 모듈 분리 규칙

| 파일 | 용도 |
|------|------|
| `app.go` | App 구조체, 생성자(`NewApp`), 글로벌 상태 필드 |
| `app_ui.go` | UI 위젯 생성 (`createMainContent`, `createChannelTab`, `createHistoryPanel`) |
| `app_handlers.go` | 채널별 버튼 이벤트 핸들러 (`onChannelProcess`, `onCopyChannelResult`) |
| `app_queue.go` | `ChannelState` 구조체, 3채널 독립 큐 관리, Phase 1/2 실행 |
| `app_analysis.go` | Claude Code 연동, 완료 결과 로드, 이력 관리 |

### 채널별 완전 독립 워크스페이스

각 채널(1/2/3)은 완전히 독립된 워크스페이스로 동작:

- **독립 항목**: URL 입력, 프로젝트 경로, 분석 시작 버튼, 진행바, 큐, 상태 라벨, 이슈 정보, AI 분석 결과, 내부 서브탭
- **공유 항목**: 완료 이력(`completedJobs`), 글로벌 상태 라벨, 전체 중지 버튼, `processIssueUC`, `claudeAdapter`(인스턴스 공유하되 공유 상태 변경 없음)
- **`ClaudeCodeAdapter` 스레드 안전**: `AnalyzeAndGeneratePlan(mdPath, prompt, workDir)`, `ExecutePlan(planPath, workDir)` 메서드에 `workDir` 파라미터를 직접 전달하여 공유 상태 변경 없이 채널별 독립 실행

### ChannelState 구조체 (`app_queue.go`)

```go
type ChannelState struct {
    Index, Name                              // 채널 식별
    UrlEntry, ProjectPathEntry, ProcessBtn   // 채널별 입력 위젯
    ProgressBar, ResultText, AnalysisText    // 채널별 결과 위젯
    StatusLabel, CopyResultBtn, CopyAnalysisBtn, ExecutePlanBtn
    QueueList, InnerTabs                     // 큐 목록, [이슈 정보|AI 분석] 서브탭
    CurrentDoc, CurrentMDPath                // 채널별 상태
    CurrentAnalysisPath, CurrentPlanPath, CurrentScriptPath
}
```

## TDD 개발 순서

1. `internal/mock/mock.go`에 Mock 필요 시 추가 (Function 필드 패턴)
2. `*_test.go`에 테스트 작성 (RED) - Arrange/Act/Assert 패턴
3. 테스트 통과할 최소 코드 구현 (GREEN)
4. 리팩토링 (REFACTOR)
5. `./scripts/test.sh`로 전체 테스트 확인

## 코딩 규칙

- Port 인터페이스: `~Repository`, `~Processor`, `~Generator`
- Adapter 구현체: `~Client`, `~Processor`, `~Generator`
- UseCase: `~UseCase`
- 에러: `fmt.Errorf("failed to ...: %w", err)` 형태로 래핑
- 구체 타입이 아닌 인터페이스로 의존성 주입
- 설정: `config.ini` (INI 형식, `gopkg.in/ini.v1`) — 로딩 순서: 현재 디렉토리 → `~/.jira-ai-generator/config.ini`

## config.ini 설정

```ini
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
work_dir = ./
project_path_1 = /path/to/project1   # 채널 1 프로젝트 경로
project_path_2 = /path/to/project2   # 채널 2 프로젝트 경로
project_path_3 = /path/to/project3   # 채널 3 프로젝트 경로
enabled = true
```
