package state

// ProgressCallback UseCase에서 UI로 진행률을 전달하는 콜백 함수 타입
type ProgressCallback func(phase ProcessPhase, step, total int, message string)

// ProgressReporter 진행률 보고 인터페이스
type ProgressReporter interface {
	// ReportProgress 진행률 보고
	ReportProgress(phase ProcessPhase, step, total int, message string)
	// ReportPhaseChange 단계 변경 보고
	ReportPhaseChange(phase ProcessPhase)
	// ReportLog 로그 보고
	ReportLog(level LogLevel, message string)
	// ReportError 에러 보고
	ReportError(err error)
}

// ChannelProgressReporter 진행률 리포터
type ChannelProgressReporter struct {
	appState *AppState
	source   string
}

// NewChannelProgressReporter 새 진행률 리포터 생성 (channelIndex는 하위 호환용으로 무시)
func NewChannelProgressReporter(channelIndex int, appState *AppState, source string) *ChannelProgressReporter {
	return &ChannelProgressReporter{
		appState: appState,
		source:   source,
	}
}

// ReportProgress 진행률 보고 구현
func (r *ChannelProgressReporter) ReportProgress(phase ProcessPhase, step, total int, message string) {
	r.appState.UpdateProgress(0, step, total, message)
}

// ReportPhaseChange 단계 변경 보고 구현
func (r *ChannelProgressReporter) ReportPhaseChange(phase ProcessPhase) {
	r.appState.UpdatePhase(0, phase)
}

// ReportLog 로그 보고 구현
func (r *ChannelProgressReporter) ReportLog(level LogLevel, message string) {
	r.appState.AddLog(level, message, r.source)
}

// ReportError 에러 보고 구현
func (r *ChannelProgressReporter) ReportError(err error) {
	r.appState.AddLog(LogError, err.Error(), r.source)
}

// ToCallback ProgressCallback 함수로 변환
func (r *ChannelProgressReporter) ToCallback() ProgressCallback {
	return func(phase ProcessPhase, step, total int, message string) {
		r.ReportProgress(phase, step, total, message)
	}
}

// NullProgressReporter 아무 동작도 하지 않는 리포터 (테스트용)
type NullProgressReporter struct{}

// NewNullProgressReporter 새 Null 리포터 생성
func NewNullProgressReporter() *NullProgressReporter {
	return &NullProgressReporter{}
}

// ReportProgress 구현 (no-op)
func (r *NullProgressReporter) ReportProgress(phase ProcessPhase, step, total int, message string) {}

// ReportPhaseChange 구현 (no-op)
func (r *NullProgressReporter) ReportPhaseChange(phase ProcessPhase) {}

// ReportLog 구현 (no-op)
func (r *NullProgressReporter) ReportLog(level LogLevel, message string) {}

// ReportError 구현 (no-op)
func (r *NullProgressReporter) ReportError(err error) {}

// ToCallback 구현 (no-op callback)
func (r *NullProgressReporter) ToCallback() ProgressCallback {
	return func(phase ProcessPhase, step, total int, message string) {}
}

// ConsoleProgressReporter 콘솔 출력 리포터 (디버깅용)
type ConsoleProgressReporter struct {
	channelIndex int
	source       string
}

// NewConsoleProgressReporter 새 콘솔 리포터 생성
func NewConsoleProgressReporter(channelIndex int, source string) *ConsoleProgressReporter {
	return &ConsoleProgressReporter{
		channelIndex: channelIndex,
		source:       source,
	}
}

// ReportProgress 구현
func (r *ConsoleProgressReporter) ReportProgress(phase ProcessPhase, step, total int, message string) {
	// fmt.Printf 대신 로깅 시스템 사용 권장
	// 여기서는 의도적으로 비워둠 (디버그 빌드에서만 활성화)
}

// ReportPhaseChange 구현
func (r *ConsoleProgressReporter) ReportPhaseChange(phase ProcessPhase) {}

// ReportLog 구현
func (r *ConsoleProgressReporter) ReportLog(level LogLevel, message string) {}

// ReportError 구현
func (r *ConsoleProgressReporter) ReportError(err error) {}

// ToCallback 구현
func (r *ConsoleProgressReporter) ToCallback() ProgressCallback {
	return func(phase ProcessPhase, step, total int, message string) {}
}

// CompositeProgressReporter 여러 리포터를 조합하는 리포터
type CompositeProgressReporter struct {
	reporters []ProgressReporter
}

// NewCompositeProgressReporter 새 조합 리포터 생성
func NewCompositeProgressReporter(reporters ...ProgressReporter) *CompositeProgressReporter {
	return &CompositeProgressReporter{
		reporters: reporters,
	}
}

// ReportProgress 모든 리포터에 전달
func (r *CompositeProgressReporter) ReportProgress(phase ProcessPhase, step, total int, message string) {
	for _, reporter := range r.reporters {
		reporter.ReportProgress(phase, step, total, message)
	}
}

// ReportPhaseChange 모든 리포터에 전달
func (r *CompositeProgressReporter) ReportPhaseChange(phase ProcessPhase) {
	for _, reporter := range r.reporters {
		reporter.ReportPhaseChange(phase)
	}
}

// ReportLog 모든 리포터에 전달
func (r *CompositeProgressReporter) ReportLog(level LogLevel, message string) {
	for _, reporter := range r.reporters {
		reporter.ReportLog(level, message)
	}
}

// ReportError 모든 리포터에 전달
func (r *CompositeProgressReporter) ReportError(err error) {
	for _, reporter := range r.reporters {
		reporter.ReportError(err)
	}
}

// ToCallback 첫 번째 리포터의 콜백 반환
func (r *CompositeProgressReporter) ToCallback() ProgressCallback {
	if len(r.reporters) > 0 {
		if cr, ok := r.reporters[0].(*ChannelProgressReporter); ok {
			return cr.ToCallback()
		}
	}
	return func(phase ProcessPhase, step, total int, message string) {}
}
