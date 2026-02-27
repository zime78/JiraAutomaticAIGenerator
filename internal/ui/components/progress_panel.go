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

// ProgressPanel 진행 상황 표시 패널
type ProgressPanel struct {
	widget.BaseWidget

	// UI 요소
	container    *fyne.Container
	progressBar  *widget.ProgressBar
	stepItems    []*StepItem
	statusLabel  *widget.Label
	messageLabel *widget.Label

	// 상태
	currentPhase state.ProcessPhase
	progress     float64
}

// StepItem 단계별 진행 상황 아이템
type StepItem struct {
	widget.BaseWidget

	container  *fyne.Container
	icon       *canvas.Text
	nameLabel  *widget.Label
	status     state.StepStatus
	progress   float64
}

// NewProgressPanel 새 ProgressPanel 생성
func NewProgressPanel() *ProgressPanel {
	p := &ProgressPanel{
		progressBar:  widget.NewProgressBar(),
		statusLabel:  widget.NewLabel("준비됨"),
		messageLabel: widget.NewLabel(""),
		stepItems:    make([]*StepItem, 5),
	}

	// 단계 아이템 생성
	stepNames := []string{
		"이슈 조회",
		"첨부파일 다운로드",
		"프레임 추출",
		"문서 생성",
		"AI 분석",
	}

	stepsContainer := container.NewVBox()
	for i, name := range stepNames {
		p.stepItems[i] = NewStepItem(name)
		stepsContainer.Add(p.stepItems[i])
	}

	// 진행률 바 스타일링
	p.progressBar.Min = 0
	p.progressBar.Max = 1

	// 메시지 라벨 스타일 - Wrapping 제거하여 세로 줄바꿈 방지
	p.messageLabel.Wrapping = fyne.TextTruncate

	// 전체 컨테이너 구성
	header := container.NewVBox(
		widget.NewLabelWithStyle("📊 진행 상황", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
	)

	// Border 레이아웃으로 messageLabel이 남은 공간을 차지하도록 함
	statusRow := container.NewBorder(
		nil, nil,
		container.NewHBox(p.statusLabel, widget.NewLabel(" | ")), // 왼쪽 고정
		nil,
		p.messageLabel, // 중앙 - 남은 공간 차지
	)

	progressSection := container.NewVBox(
		p.progressBar,
		statusRow,
	)

	p.container = container.NewVBox(
		header,
		progressSection,
		widget.NewSeparator(),
		stepsContainer,
	)

	p.ExtendBaseWidget(p)
	return p
}

// CreateRenderer Fyne 위젯 렌더러 구현
func (p *ProgressPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

// SetPhase 현재 단계 설정
func (p *ProgressPanel) SetPhase(phase state.ProcessPhase) {
	// 변경이 없으면 스킵
	if p.currentPhase == phase {
		return
	}

	fyne.Do(func() {
		p.currentPhase = phase
		p.progress = phase.Progress()
		p.progressBar.SetValue(p.progress)
		p.statusLabel.SetText(phase.String())

		// 단계별 상태 업데이트
		phaseIndex := int(phase) - 1 // PhaseIdle은 0이므로 -1

		for i, item := range p.stepItems {
			if i < phaseIndex {
				item.SetStatus(state.StepCompleted)
			} else if i == phaseIndex {
				item.SetStatus(state.StepRunning)
			} else {
				item.SetStatus(state.StepPending)
			}
		}
	})
}

// SetProgress 진행률 설정
func (p *ProgressPanel) SetProgress(progress float64, message string) {
	fyne.Do(func() {
		p.progress = progress
		p.progressBar.SetValue(progress)
		p.messageLabel.SetText(message)

		// 현재 진행 중인 단계의 진행률 업데이트
		for _, item := range p.stepItems {
			if item.status == state.StepRunning {
				item.SetProgress(progress)
				break
			}
		}
	})
}

// SetStepProgress 특정 단계의 진행률 설정
func (p *ProgressPanel) SetStepProgress(stepIndex int, progress float64, message string) {
	if stepIndex >= 0 && stepIndex < len(p.stepItems) {
		p.stepItems[stepIndex].SetProgress(progress)
		fyne.Do(func() {
			p.messageLabel.SetText(message)
		})
	}
}

// Reset 상태 초기화
func (p *ProgressPanel) Reset() {
	p.currentPhase = state.PhaseIdle
	p.progress = 0
	fyne.Do(func() {
		p.progressBar.SetValue(0)
		p.statusLabel.SetText("준비됨")
		p.messageLabel.SetText("")

		for _, item := range p.stepItems {
			item.SetStatus(state.StepPending)
			item.SetProgress(0)
		}
	})
}

// SetError 에러 상태 표시
func (p *ProgressPanel) SetError(errMsg string) {
	fyne.Do(func() {
		p.statusLabel.SetText("❌ 오류 발생")
		p.messageLabel.SetText(errMsg)

		// 현재 진행 중인 단계를 실패로 표시
		for _, item := range p.stepItems {
			if item.status == state.StepRunning {
				item.SetStatus(state.StepFailed)
				break
			}
		}
	})
}

// SetComplete 완료 상태 표시
func (p *ProgressPanel) SetComplete() {
	fyne.Do(func() {
		p.currentPhase = state.PhaseCompleted
		p.progress = 1.0
		p.progressBar.SetValue(1.0)
		p.statusLabel.SetText("✅ 완료")
		p.messageLabel.SetText("")

		for _, item := range p.stepItems {
			item.SetStatus(state.StepCompleted)
		}
	})
}

// NewStepItem 새 StepItem 생성
func NewStepItem(name string) *StepItem {
	s := &StepItem{
		nameLabel: widget.NewLabel(name),
		icon:      canvas.NewText("○", theme.ForegroundColor()),
		status:    state.StepPending,
		progress:  0,
	}

	s.icon.TextSize = 14
	s.nameLabel.TextStyle = fyne.TextStyle{}

	s.container = container.NewHBox(
		s.icon,
		s.nameLabel,
	)

	s.ExtendBaseWidget(s)
	return s
}

// CreateRenderer StepItem 렌더러
func (s *StepItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.container)
}

// SetStatus 상태 설정
func (s *StepItem) SetStatus(status state.StepStatus) {
	// 변경이 없으면 스킵
	if s.status == status {
		return
	}

	s.status = status

	fyne.Do(func() {
		switch status {
		case state.StepPending:
			s.icon.Text = "○"
			s.icon.Color = theme.DisabledColor()
			s.nameLabel.TextStyle = fyne.TextStyle{}
		case state.StepRunning:
			s.icon.Text = "◉"
			s.icon.Color = color.RGBA{R: 59, G: 130, B: 246, A: 255} // 파란색
			s.nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		case state.StepCompleted:
			s.icon.Text = "✓"
			s.icon.Color = color.RGBA{R: 34, G: 197, B: 94, A: 255} // 녹색
			s.nameLabel.TextStyle = fyne.TextStyle{}
		case state.StepFailed:
			s.icon.Text = "✗"
			s.icon.Color = color.RGBA{R: 239, G: 68, B: 68, A: 255} // 빨간색
			s.nameLabel.TextStyle = fyne.TextStyle{}
		}

		s.icon.Refresh()
		s.nameLabel.Refresh()
	})
}

// SetProgress 진행률 설정
func (s *StepItem) SetProgress(progress float64) {
	s.progress = progress

	if s.status == state.StepRunning && progress > 0 && progress < 1 {
		fyne.Do(func() {
			s.icon.Text = fmt.Sprintf("%.0f%%", progress*100)
			s.icon.Refresh()
		})
	}
}
