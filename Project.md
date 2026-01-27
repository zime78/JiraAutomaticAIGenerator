# Jira AI Generator 프로젝트

## 프로젝트 개요

Jira 티켓을 분석하여 AI가 해당 이슈를 수정할 수 있는 문서를 자동 생성하는 macOS 앱

## 기능 설명

1. **Jira 이슈 조회**: URL 입력 → 이슈 상세 정보 자동 조회
2. **이미지 처리**: 첨부 이미지 로컬 다운로드 후 문서에 링크 삽입
3. **동영상 처리**: ffmpeg로 프레임 추출하여 이미지로 변환
4. **마크다운 생성**: 이슈번호 폴더 (예: `ITSM-5239/`) 생성 후 `.md` 파일로 저장
5. **AI 분석**: Claude Code CLI 연동하여 자동 분석
6. **다중 채널 큐**: 3개 채널에서 동시 분석 가능
7. **완료 이력**: 이전 분석 결과 조회 및 재확인

## 기술 스택

- **언어**: Go
- **GUI**: Fyne (macOS 네이티브)
- **외부 도구**: ffmpeg (동영상), Claude Code CLI (AI)
- **API**: Jira REST API

## UI 모듈 구조

| 파일 | 역할 |
|------|------|
| `app.go` | 앱 구조체, 생성자, Run |
| `app_ui.go` | UI 위젯 생성 |
| `app_handlers.go` | 버튼 이벤트 핸들러 |
| `app_queue.go` | 분석 큐 관리 (3채널) |
| `app_analysis.go` | AI 분석 및 결과 처리 |
| `theme.go` | 한글 폰트 테마 |

## 워크플로우

```
Jira URL 입력 → 이슈 조회 → 마크다운 생성 → 채널 큐에 추가 → Claude 분석 → 결과 표시
```

## 설정

`config.ini` 파일에서 설정:

- Jira URL, API Key
- Claude CLI 경로 및 프로젝트 경로
- 출력 디렉토리
- AI 프롬프트 템플릿
