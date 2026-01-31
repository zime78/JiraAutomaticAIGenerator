package components

import (
	"fmt"
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

	// 채널별 상태 인디케이터
	channelIndicators [3]*ChannelIndicator

	// 최근 활동
	recentActivity *widget.Label
}

// ChannelIndicator 채널 상태 인디케이터
type ChannelIndicator struct {
	widget.BaseWidget

	container *fyne.Container
	icon      *canvas.Circle
	label     *widget.Label
	status    state.ProcessPhase
}

// NewStatusBar 새 StatusBar 생성
func NewStatusBar() *StatusBar {
	sb := &StatusBar{
		globalStatus:   widget.NewLabel("준비됨"),
		globalIcon:     canvas.NewText("●", color.RGBA{R: 34, G: 197, B: 94, A: 255}),
		recentActivity: widget.NewLabel(""),
	}

	sb.globalIcon.TextSize = 12

	// 채널 인디케이터 생성
	for i := 0; i < 3; i++ {
		sb.channelIndicators[i] = NewChannelIndicator(i)
	}

	// 레이아웃 구성
	channelSection := container.NewHBox(
		sb.channelIndicators[0],
		widget.NewSeparator(),
		sb.channelIndicators[1],
		widget.NewSeparator(),
		sb.channelIndicators[2],
	)

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
				channelSection,
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
	sb.globalStatus.SetText(status)

	if isError {
		sb.globalIcon.Text = "●"
		sb.globalIcon.Color = color.RGBA{R: 239, G: 68, B: 68, A: 255} // 빨간색
	} else {
		sb.globalIcon.Text = "●"
		sb.globalIcon.Color = color.RGBA{R: 34, G: 197, B: 94, A: 255} // 녹색
	}
	sb.globalIcon.Refresh()
}

// SetChannelStatus 채널 상태 설정
func (sb *StatusBar) SetChannelStatus(channelIndex int, phase state.ProcessPhase) {
	if channelIndex >= 0 && channelIndex < 3 {
		sb.channelIndicators[channelIndex].SetStatus(phase)
	}
}

// SetRecentActivity 최근 활동 설정
func (sb *StatusBar) SetRecentActivity(activity string) {
	sb.recentActivity.SetText(activity)
}

// UpdateFromState AppState에서 상태 업데이트
func (sb *StatusBar) UpdateFromState(appState *state.AppState) {
	sb.SetGlobalStatus(appState.GlobalStatus, false)

	for i := 0; i < 3; i++ {
		ch := appState.GetChannel(i)
		if ch != nil {
			sb.SetChannelStatus(i, ch.Phase)
		}
	}
}

// NewChannelIndicator 새 채널 인디케이터 생성
func NewChannelIndicator(index int) *ChannelIndicator {
	ci := &ChannelIndicator{
		icon:   canvas.NewCircle(color.RGBA{R: 156, G: 163, B: 175, A: 255}),
		label:  widget.NewLabel(fmt.Sprintf("CH%d", index+1)),
		status: state.PhaseIdle,
	}

	ci.icon.StrokeWidth = 0
	ci.icon.Resize(fyne.NewSize(10, 10))

	ci.label.TextStyle = fyne.TextStyle{Monospace: true}

	ci.container = container.NewHBox(
		ci.icon,
		ci.label,
	)

	ci.ExtendBaseWidget(ci)
	return ci
}

// CreateRenderer ChannelIndicator 렌더러
func (ci *ChannelIndicator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ci.container)
}

// SetStatus 상태 설정
func (ci *ChannelIndicator) SetStatus(phase state.ProcessPhase) {
	ci.status = phase

	var statusColor color.Color
	var statusText string

	switch phase {
	case state.PhaseIdle:
		statusColor = color.RGBA{R: 156, G: 163, B: 175, A: 255} // 회색
		statusText = "대기"
	case state.PhaseFetchingIssue, state.PhaseDownloadingAttachments,
		state.PhaseExtractingFrames, state.PhaseGeneratingDocument, state.PhaseAnalyzing:
		statusColor = color.RGBA{R: 59, G: 130, B: 246, A: 255} // 파란색
		statusText = "진행"
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

	ci.icon.FillColor = statusColor
	ci.icon.Refresh()

	ci.label.SetText(fmt.Sprintf("CH%d:%s", ci.getIndex()+1, statusText))
}

// getIndex 인덱스 조회 (label에서 추출)
func (ci *ChannelIndicator) getIndex() int {
	// label이 "CH1", "CH2", "CH3" 형식이므로 첫 번째 문자 제외
	if len(ci.label.Text) >= 3 {
		return int(ci.label.Text[2] - '1')
	}
	return 0
}

// GetStatus 현재 상태 조회
func (ci *ChannelIndicator) GetStatus() state.ProcessPhase {
	return ci.status
}
