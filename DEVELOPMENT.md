# 개발 지침 (Development Guidelines)

## 아키텍처 원칙

이 프로젝트는 **Clean Architecture**를 따릅니다. 새로운 기능을 추가할 때 반드시 아래 원칙을 준수하세요.

### 의존성 규칙

```
UI → UseCase → Port ← Adapter
                ↑
              Domain
```

- 내부 레이어는 외부 레이어에 의존하지 않습니다
- 의존성은 항상 **안쪽**으로 향합니다
- Port(인터페이스)를 통해 의존성을 역전합니다

### 레이어별 책임

| 레이어 | 파일 위치 | 책임 | 의존 가능 대상 |
| ------ | --------- | ---- | -------------- |
| **Domain** | `internal/domain/` | 핵심 비즈니스 엔티티 정의 | 없음 (순수) |
| **Port** | `internal/port/` | 인터페이스 정의 | Domain |
| **UseCase** | `internal/usecase/` | 비즈니스 로직 오케스트레이션 | Domain, Port |
| **Adapter** | `internal/adapter/` | 외부 시스템 연동 구현 | Domain, Port |
| **UI** | `internal/ui/` | 사용자 인터페이스 | UseCase, Adapter(일부 직접 사용) |
| **Config** | `internal/config/` | 설정 관리 | 없음 |
| **Mock** | `internal/mock/` | 테스트용 Mock 구현체 | Domain, Port |

---

## UI 파일 구조

UI 레이어는 기능별로 **모듈화**되어 있습니다:

| 파일 | 역할 | 주요 함수/구조체 |
|------|------|-----------------|
| `app.go` | 앱 구조체, 생성자, 실행 | `App`, `NewApp()`, `Run()` |
| `app_ui.go` | UI 생성 코드 | `createMainContent()`, `createChannelPanels()`, `createHistoryPanel()` |
| `app_handlers.go` | 이벤트 핸들러 | `onProcess()`, `processIssue()`, `onCopy()` |
| `app_queue.go` | 분석 큐 관리 | `AnalysisJob`, `AnalysisQueue`, `addToQueue()`, `processQueue()` |
| `app_analysis.go` | AI 분석 관련 | `onAnalyze()`, `monitorAnalysisLog()`, `loadPreviousAnalysis()` |
| `theme.go` | 한글 테마 | `KoreanTheme` |

### 새 UI 기능 추가 시

1. **기능 분류 확인**: 어느 파일에 해당하는지 확인
2. **적절한 파일에 메서드 추가**:
   - UI 생성 → `app_ui.go`
   - 버튼 핸들러 → `app_handlers.go`
   - 큐/작업 관리 → `app_queue.go`
   - 분석 기능 → `app_analysis.go`
3. **상태 필드 필요 시**: `app.go`의 `App` 구조체에 추가

---

## TDD 개발 프로세스

이 프로젝트는 **TDD (Test-Driven Development)** 방식을 따릅니다.

### TDD 사이클

```
1. RED    → 실패하는 테스트 작성
2. GREEN  → 테스트를 통과하는 최소 코드 작성
3. REFACTOR → 코드 개선 (테스트는 계속 통과해야 함)
```

### 새 기능 개발 순서

1. **테스트 먼저 작성** (`*_test.go`)

   ```go
   func TestNewFeature_Success(t *testing.T) {
       // Arrange: Mock 설정
       mockRepo := &mock.JiraRepository{
           GetIssueFunc: func(key string) (*domain.JiraIssue, error) {
               return &domain.JiraIssue{Key: key}, nil
           },
       }
       
       // Act: 기능 실행
       result, err := uc.Execute("TEST-123")
       
       // Assert: 결과 검증
       if err != nil {
           t.Fatalf("expected no error, got %v", err)
       }
   }
   ```

2. **테스트 실행 (실패 확인)**

   ```bash
   ./scripts/test.sh
   ```

3. **최소 구현** - 테스트를 통과할 수 있는 최소한의 코드만 작성

4. **테스트 통과 확인**

5. **리팩토링** - 테스트가 통과하는 상태에서 코드 개선

### Mock 사용

모든 Port 인터페이스에 대한 Mock이 `internal/mock/` 에 있습니다:

```go
import "jira-ai-generator/internal/mock"

mockJira := &mock.JiraRepository{
    GetIssueFunc: func(key string) (*domain.JiraIssue, error) {
        return &domain.JiraIssue{Key: key}, nil
    },
}
```

### 테스트 명령어

```bash
# 전체 테스트
./scripts/test.sh

# 특정 패키지만
go test ./internal/usecase/...

# 커버리지
go test -cover ./...
```

## 새 기능 추가 가이드

### 1. 새로운 외부 서비스 연동 시

1. **Port 인터페이스 정의** (`internal/port/interfaces.go`)

   ```go
   type NewServiceRepository interface {
       DoSomething(input string) (Output, error)
   }
   ```

2. **Adapter 구현체 생성** (`internal/adapter/new_service.go`)

   ```go
   type NewServiceClient struct { ... }
   
   func (c *NewServiceClient) DoSomething(input string) (Output, error) {
       // 실제 구현
   }
   ```

3. **UseCase에서 주입받아 사용**

   ```go
   type MyUseCase struct {
       newService port.NewServiceRepository
   }
   ```

### 2. 새로운 비즈니스 로직 추가 시

1. `internal/usecase/` 에 새 UseCase 파일 생성
2. 필요한 Port 인터페이스를 생성자에서 주입받음
3. `Execute()` 메서드에 비즈니스 로직 구현

### 3. UI 변경 시

1. 적절한 UI 파일 선택:
   - 새 위젯 → `app_ui.go`
   - 버튼 동작 → `app_handlers.go`
   - 백그라운드 작업 → `app_queue.go` 또는 `app_analysis.go`
2. 비즈니스 로직은 UseCase를 통해 호출
3. 현재 `MarkdownGenerator`, `ClaudeCodeAdapter`는 Port 인터페이스 없이 App에서 직접 사용 중

---

## 코딩 컨벤션

### 네이밍

- **Port 인터페이스**: `~Repository`, `~Processor`, `~Generator`
- **Adapter 구현체**: `~Client`, `~Processor`, `~Generator`
- **UseCase**: `~UseCase`

### 에러 처리

```go
// Good: 에러 래핑
return nil, fmt.Errorf("failed to fetch issue: %w", err)

// Bad: 에러 무시 또는 단순 반환
return nil, err
```

### 의존성 주입

```go
// Good: 인터페이스로 주입
func NewProcessIssueUseCase(jiraRepo port.JiraRepository) *ProcessIssueUseCase

// Bad: 구체 타입으로 주입
func NewProcessIssueUseCase(jiraClient *adapter.JiraClient) *ProcessIssueUseCase
```

---

## 빌드 및 배포

```bash
# 개발 모드
./scripts/run.sh

# 프로덕션 빌드
./scripts/release.sh 1.0.0

# 정리
./scripts/clean.sh
```

---

## 체크리스트

새 기능 추가 전 확인:

- [ ] 적절한 레이어에 코드를 배치했는가?
- [ ] 의존성 방향이 올바른가? (안쪽으로만)
- [ ] 인터페이스를 통해 의존성을 주입받는가?
- [ ] 에러를 적절히 래핑했는가?
- [ ] 테스트 가능한 구조인가?
- [ ] UI 코드는 적절한 모듈 파일에 배치했는가?
