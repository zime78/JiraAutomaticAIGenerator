# Jira AI Generator

Jira 티켓을 분석하여 AI가 처리할 수 있는 마크다운 문서를 자동 생성하는 macOS 앱입니다.

## 주요 기능

- 🔍 Jira URL 입력 → 이슈 상세 정보 자동 조회
- 📷 이미지 첨부파일 자동 다운로드
- 🎬 동영상 첨부파일 → 프레임 이미지 추출 (ffmpeg 사용)
- 📝 AI 처리용 마크다운 문서 생성
- 📋 결과 클립보드 복사 기능
- 🤖 **Claude Code 연동** - AI 자동 분석
- 📊 **3채널 분석 큐** - 동시 3개 분석 지원
- 📜 **완료 이력** - 이전 분석 결과 조회

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
# 개발 모드 실행
./scripts/run.sh

# 또는 직접 실행
go run ./cmd/app
```

## 사용 방법

### 기본 워크플로우

1. 앱 실행
2. **프로젝트 경로** 입력 (AI 분석에 사용될 소스코드 위치)
3. Jira 이슈 URL 입력 (예: `https://domain.atlassian.net/browse/PROJ-123`)
4. **"분석 시작"** 클릭 → 이슈 정보 조회
5. AI 분석 탭에서 채널별 **"추가"** 클릭 → 분석 큐에 추가
6. 분석 완료 후 **"분석 결과 복사"** 클릭
7. AI 채팅에 붙여넣기

### 다중 채널 분석

| 기능 | 설명 |
|------|------|
| **채널 1~3** | 동시에 최대 3개 분석 가능 |
| **추가** | 현재 이슈를 해당 채널 큐에 추가 |
| **중지** | 해당 채널의 현재 분석 중지 |
| **전체 중지** | 모든 채널의 분석 중지 |

### 완료 이력

- 앱 시작 시 `output/` 폴더의 기존 분석 결과 자동 로드
- 완료된 분석 클릭 → 해당 결과 표시

## 스크립트

| 스크립트 | 용도 | 사용법 |
| -------- | ---- | ------ |
| `build.sh` | 기본 빌드 | `./scripts/build.sh` |
| `run.sh` | 디버깅/개발 모드 | `./scripts/run.sh` |
| `release.sh` | 프로덕션 배포 빌드 | `./scripts/release.sh [버전]` |
| `test_jira.sh` | Jira API 연결 테스트 | `./scripts/test_jira.sh ITSM-5239` |
| `clean.sh` | 빌드 산출물 정리 | `./scripts/clean.sh` |
| `check_deps.sh` | 시스템 의존성 확인 | `./scripts/check_deps.sh` |
| `test.sh` | 테스트 실행 | `./scripts/test.sh [-v\|-cover\|-coverprofile]` |

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
├── cmd/app/main.go              # 앱 진입점
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
