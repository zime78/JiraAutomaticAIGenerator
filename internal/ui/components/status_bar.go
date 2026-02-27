package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/ui/state"
)

// StatusBar 상태바 컴포넌트
type StatusBar struct {
	widget.BaseWidget

	container *fyne.Container

	// 글로벌 상태
	globalStatus *widget.Label
	globalIcon   *canvas.Text

	// 상태 인디케이터 (단일 채널)
	statusIndicator *StatusIndicator

	// 최근 활동
	recentActivity *widget.Label
}

// StatusIndicator 상태 인디케이터
type StatusIndicator struct {
	widget.BaseWidget

	container *fyne.Container
	icon      *canvas.Circle
	label     *widget.Label
	status    state.ProcessPhase
}

// NewStatusBar 새 StatusBar 생성
func NewStatusBar() *StatusBar {
	sb := &StatusBar{
		globalStatus:    widget.NewLabel("준비됨"),
		globalIcon:      canvas.NewText("●", color.RGBA{R: 34, G: 197, B: 94, A: 255}),
		statusIndicator: NewStatusIndicator(),
		recentActivity:  widget.NewLabel(""),
	}

	sb.globalIcon.TextSize = 12

	// 레이아웃 구성
	globalSection := container.NewHBox(
		sb.globalIcon,
		sb.globalStatus,
	)

	// 배경색 설정
	background := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 255})

	sb.container = container.NewStack(
		background,
		container.NewPadded(
			container.NewBorder(
				nil, nil,
				globalSection,
				sb.recentActivity,
				sb.statusIndicator,
			),
		),
	)

	sb.ExtendBaseWidget(sb)
	return sb
}

// CreateRenderer StatusBar 렌더러
func (sb *StatusBar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(sb.container)
}

// SetGlobalStatus 글로벌 상태 설정
func (sb *StatusBar) SetGlobalStatus(status string, isError bool) {
	fyne.Do(func() {
		sb.globalStatus.SetText(status)

		if isError {
			sb.globalIcon.Text = "●"
			sb.globalIcon.Color = color.RGBA{R: 239, G: 68, B: 68, A: 255} // 빨간색
		} else {
			sb.globalIcon.Text = "●"
			sb.globalIcon.Color = color.RGBA{R: 34, G: 197, B: 94, A: 255} // 녹색
		}
		sb.globalIcon.Refresh()
	})
}

// SetChannelStatus 상태 설정 (단일 채널)
func (sb *StatusBar) SetChannelStatus(phase state.ProcessPhase) {
	fyne.Do(func() {
		sb.statusIndicator.SetStatus(phase)
	})
}

// SetRecentActivity 최근 활동 설정
func (sb *StatusBar) SetRecentActivity(activity string) {
	sb.recentActivity.SetText(activity)
}

// UpdateFromState AppState에서 상태 업데이트
func (sb *StatusBar) UpdateFromState(appState *state.AppState) {
	sb.SetGlobalStatus(appState.GlobalStatus, false)

	ch := appState.GetActiveChannel()
	if ch != nil {
		sb.SetChannelStatus(ch.Phase)
	}
}

// NewStatusIndicator 새 상태 인디케이터 생성
func NewStatusIndicator() *StatusIndicator {
	si := &StatusIndicator{
		icon:   canvas.NewCircle(color.RGBA{R: 156, G: 163, B: 175, A: 255}),
		label:  widget.NewLabel("대기"),
		status: state.PhaseIdle,
	}

	si.icon.StrokeWidth = 0
	si.icon.Resize(fyne.NewSize(10, 10))

	si.label.TextStyle = fyne.TextStyle{Monospace: true}

	si.container = container.NewHBox(
		si.icon,
		si.label,
	)

	si.ExtendBaseWidget(si)
	return si
}

// CreateRenderer StatusIndicator 렌더러
func (si *StatusIndicator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(si.container)
}

// SetStatus 상태 설정
func (si *StatusIndicator) SetStatus(phase state.ProcessPhase) {
	si.status = phase

	var statusColor color.Color
	var statusText string

	switch phase {
	case state.PhaseIdle:
		statusColor = color.RGBA{R: 156, G: 163, B: 175, A: 255} // 회색
		statusText = "대기"
	case state.PhaseFetchingIssue:
		statusColor = color.RGBA{R: 59, G: 130, B: 246, A: 255} // 파란색
		statusText = "이슈 조회"
	case state.PhaseDownloadingAttachments:
		statusColor = color.RGBA{R: 59, G: 130, B: 246, A: 255}
		statusText = "첨부파일 다운로드"
	case state.PhaseExtractingFrames:
		statusColor = color.RGBA{R: 59, G: 130, B: 246, A: 255}
		statusText = "프레임 추출"
	case state.PhaseGeneratingDocument:
		statusColor = color.RGBA{R: 59, G: 130, B: 246, A: 255}
		statusText = "문서 생성"
	case state.PhaseAnalyzing:
		statusColor = color.RGBA{R: 147, G: 51, B: 234, A: 255} // 보라색
		statusText = "AI 분석"
	case state.PhaseCompleted:
		statusColor = color.RGBA{R: 34, G: 197, B: 94, A: 255} // 녹색
		statusText = "완료"
	case state.PhaseFailed:
		statusColor = color.RGBA{R: 239, G: 68, B: 68, A: 255} // 빨간색
		statusText = "실패"
	default:
		statusColor = theme.DisabledColor()
		statusText = "?"
	}

	si.icon.FillColor = statusColor
	si.icon.Refresh()

	si.label.SetText(statusText)
}

// GetStatus 현재 상태 조회
func (si *StatusIndicator) GetStatus() state.ProcessPhase {
	return si.status
}
