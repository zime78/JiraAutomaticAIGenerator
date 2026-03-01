# Jira AI Generator

Jira 티켓을 분석하여 AI가 처리할 수 있는 마크다운 문서를 자동 생성하는 macOS 앱입니다.
**GUI**와 **CLI** 두 가지 방식으로 사용할 수 있습니다.

## 주요 기능

- 🔍 Jira URL 입력 → 이슈 상세 정보 자동 조회
- 📷 이미지 첨부파일 자동 다운로드
- 🎬 동영상 첨부파일 → 프레임 이미지 추출 (ffmpeg 사용)
- 📝 AI 처리용 마크다운 문서 생성
- 📋 결과 클립보드 복사 기능
- 🤖 **Claude Code 연동** - AI 자동 분석
- 📜 **완료 이력** - 이전 분석 결과 조회
- 🖥️ **CLI 지원** - 터미널에서 단일/배치 처리

## 아키텍처

이 프로젝트는 **Clean Architecture** 패턴을 따릅니다.

```
┌─────────────────────────────────────────────┐
│                  UI Layer                    │
│              (Fyne GUI)                      │
└─────────────────┬───────────────────────────┘
                  │ depends on
┌─────────────────▼───────────────────────────┐
│              UseCase Layer                   │
│         (ProcessIssueUseCase)               │
└─────────────────┬───────────────────────────┘
                  │ depends on
┌─────────────────▼───────────────────────────┐
│               Port Layer                     │
│     (Interfaces: JiraRepository, etc.)      │
└─────────────────┬───────────────────────────┘
                  │ implemented by
┌─────────────────▼───────────────────────────┐
│             Adapter Layer                    │
│   (JiraClient, FFmpegVideoProcessor, etc.)  │
└─────────────────────────────────────────────┘
```

### 레이어 설명

| 레이어 | 경로 | 역할 |
| ------ | ---- | ---- |
| **Domain** | `internal/domain/` | 엔티티 정의 (JiraIssue, Attachment 등) |
| **Port** | `internal/port/` | 인터페이스 정의 (의존성 역전) |
| **UseCase** | `internal/usecase/` | 비즈니스 로직 (ProcessIssueUseCase) |
| **Adapter** | `internal/adapter/` | 외부 시스템 구현체 (Jira API, Claude Code, ffmpeg 등) |
| **UI** | `internal/ui/` | Fyne GUI (분리된 모듈 구조) |
| **CLI** | `cmd/cli/` | CLI 엔트리포인트 (터미널 실행) |

## 설치 및 실행

### 사전 요구사항

1. **Go 1.21 이상**
2. **Xcode Command Line Tools** (Fyne 빌드에 필요)

   ```bash
   xcode-select --install
   ```

3. **ffmpeg** (동영상 프레임 추출용, 선택사항)

   ```bash
   brew install ffmpeg
   ```

4. **Claude Code CLI** (AI 분석용, 선택사항)

   ```bash
   npm install -g @anthropic-ai/claude-code
   ```

의존성 확인:

```bash
./scripts/check_deps.sh
```

### 설정

1. `config.ini.example`을 `config.ini`로 복사:

   ```bash
   cp config.ini.example config.ini
   ```

2. `config.ini` 파일을 편집하여 설정 입력:

   ```ini
   [jira]
   url = https://your-domain.atlassian.net
   email = your-email@example.com
   api_key = your-api-token
   
   [output]
   dir = ./output
   
   [ai]
   prompt_template = 다음 Jira 이슈를 분석하고 수정 코드를 작성해주세요:
   
   [claude]
   enabled = true
   cli_path = /usr/local/bin/claude
   work_dir = ./
   project_path = /path/to/your/project
   ```

> **설정 파일 로딩 순서**: 현재 디렉토리의 `config.ini` → `~/.jira-ai-generator/config.ini`

### 실행

```bash
# GUI - 개발 모드 실행
./scripts/run.sh

# GUI - 직접 실행
go run ./cmd/app

# CLI - 단일 이슈 처리
go run ./cmd/cli https://example.atlassian.net/browse/PROJ-123

# CLI - 이슈 키로 처리
go run ./cmd/cli PROJ-123
```

## 사용 방법

### GUI 워크플로우

1. 앱 실행
2. **프로젝트 경로** 입력 (AI 분석에 사용될 소스코드 위치)
3. Jira 이슈 URL 입력 (예: `https://domain.atlassian.net/browse/PROJ-123`)
4. **"분석 시작"** 클릭 → 이슈 조회 + AI 분석 실행
5. 분석 완료 후 **"복사"** 클릭
6. AI 채팅에 붙여넣기

### CLI 사용법

```bash
# 단일 이슈 처리 (전체 워크플로우: 이슈 조회 → 문서 생성 → AI 분석)
jira-ai-cli https://example.atlassian.net/browse/PROJ-123
jira-ai-cli PROJ-123

# AI 분석 생략 (문서 생성까지만)
jira-ai-cli --no-ai PROJ-123

# 일괄 처리 (파일에 URL/키를 한 줄씩 작성)
jira-ai-cli --batch urls.txt

# 출력 디렉토리 / 프로젝트 경로 지정
jira-ai-cli --output ./my-output --project /path/to/project PROJ-123

# 설정 파일 지정
jira-ai-cli --config /path/to/config.ini PROJ-123
```

| 옵션 | 설명 |
|------|------|
| `--batch <file>` | URL 목록 파일로 일괄 처리 (한 줄에 하나) |
| `--no-ai` | AI 분석 생략 (1차 문서 생성까지만) |
| `--project <path>` | 프로젝트 경로 (config.ini 대신 지정) |
| `--output <path>` | 출력 디렉토리 (config.ini 대신 지정) |
| `--config <path>` | 설정 파일 경로 |

### 완료 이력 (GUI)

- 앱 시작 시 `output/` 폴더의 기존 분석 결과 자동 로드
- 완료된 분석 클릭 → 해당 결과 표시

## 스크립트

| 스크립트 | 용도 | 사용법 |
| -------- | ---- | ------ |
| `build.sh` | GUI + CLI 빌드 (`dist/`) | `./scripts/build.sh` |
| `build_cli.sh` | CLI 단독 빌드 (`dist/`) | `./scripts/build_cli.sh` |
| `run.sh` | 디버깅/개발 모드 | `./scripts/run.sh` |
| `release.sh` | 프로덕션 배포 빌드 (`dist/`) | `./scripts/release.sh [버전]` |
| `test_jira.sh` | Jira API 연결 테스트 | `./scripts/test_jira.sh ITSM-5239` |
| `clean.sh` | 빌드 산출물 정리 (`dist/` 삭제) | `./scripts/clean.sh` |
| `check_deps.sh` | 시스템 의존성 확인 | `./scripts/check_deps.sh` |
| `test.sh` | 테스트 실행 | `./scripts/test.sh [-v\|-cover\|-coverprofile]` |

### 빌드 결과물 (`dist/`)

```text
dist/
├── jira-ai-generator           # 개발용 GUI (build.sh)
├── jira-ai-cli                 # 개발용 CLI (build.sh / build_cli.sh)
│
│  # release.sh 배포 빌드:
├── JiraAIGenerator_apple       # GUI - Apple Silicon
├── JiraAIGenerator_intel       # GUI - Intel Mac
├── JiraAIGenerator_universal   # GUI - 유니버설
├── JiraAICLI_apple             # CLI - Apple Silicon
├── JiraAICLI_intel             # CLI - Intel Mac
├── JiraAICLI_universal         # CLI - 유니버설
└── JiraAICLI_linux             # CLI - Linux
```

## 출력 구조

```text
output/
└── PROJ-123/
    ├── PROJ-123.md           # 생성된 마크다운 문서
    ├── PROJ-123_analysis.md  # AI 분석 결과
    ├── PROJ-123_log.txt      # 분석 로그
    ├── image1.png            # 다운로드된 이미지
    ├── video.mp4             # 다운로드된 동영상
    └── frames/               # 동영상 프레임 추출
        ├── video_frame_0001.png
        └── ...
```

## 프로젝트 구조

```text
JiraAutomaticAIGenerator/
├── cmd/
│   ├── app/main.go              # GUI 진입점 (Fyne)
│   └── cli/main.go              # CLI 진입점 (터미널)
├── internal/
│   ├── domain/                  # 도메인 엔티티
│   ├── port/                    # 인터페이스 정의
│   ├── mock/                    # 테스트용 Mock 구현체
│   ├── usecase/                 # 비즈니스 로직
│   ├── adapter/                 # 외부 시스템 구현체
│   │   ├── jira_client.go       # Jira API 클라이언트
│   │   ├── attachment_downloader.go # 첨부파일 다운로더
│   │   ├── claude_code.go       # Claude Code CLI 어댑터
│   │   ├── video_processor.go   # ffmpeg 비디오 처리
│   │   └── markdown_generator.go # 마크다운 생성
│   ├── config/                  # 설정 로더
│   └── ui/                      # Fyne GUI (모듈화)
│       ├── app.go               # 앱 구조체 및 초기화
│       ├── app_ui.go            # UI 생성 코드
│       ├── app_handlers.go      # 이벤트 핸들러
│       ├── app_queue.go         # 분석 큐 관리
│       ├── app_analysis.go      # AI 분석 관련
│       └── theme.go             # 한글 테마
├── scripts/                     # 빌드/배포 스크립트
├── config.ini.example           # 설정 템플릿
├── DEVELOPMENT.md               # 개발 가이드
└── README.md
```

## 라이선스

MIT License
