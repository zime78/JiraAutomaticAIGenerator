package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/logger"
	"jira-ai-generator/internal/ui/state"
)

var (
	// completedListPrimaryColor는 완료 항목 강조에 사용하는 연두-녹색 중간 톤이다.
	completedListPrimaryColor = color.RGBA{R: 122, G: 204, B: 90, A: 255}
)

// completedListTheme는 CompletedList 전용 색상 테마를 제공한다.
type completedListTheme struct {
	base    fyne.Theme
	primary color.Color
}

// Color는 primary 색상을 연녹색 계열로 오버라이드한다.
func (t *completedListTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return t.primary
	}
	return t.base.Color(name, variant)
}

// Font는 기본 테마 폰트를 그대로 사용한다.
func (t *completedListTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

// Icon은 기본 테마 아이콘을 그대로 사용한다.
func (t *completedListTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

// Size는 기본 테마 사이즈를 그대로 사용한다.
func (t *completedListTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.base.Size(name)
}

// AnalysisSelector 2차/3차 분석 선택 UI 컴포넌트
type AnalysisSelector struct {
	widget.BaseWidget

	containerObj fyne.CanvasObject

	// 1차 완료 항목 (2차 분석 대상)
	phase2List     *CompletedList
	startPhase2    *widget.Button
	phase2LoadIcon *widget.Icon
	phase2Status   *widget.Label

	// 2차 완료 항목 (3차 분석 대상)
	phase3List     *CompletedList
	startPhase3    *widget.Button
	phase3LoadIcon *widget.Icon
	phase3Status   *widget.Label

	eventBus   *state.EventBus
	channelIdx int

	// 선택된 항목
	selectedPhase2Item *domain.IssueRecord
	selectedPhase3Item *domain.IssueRecord

	// 현재 실행 중인 Phase
	currentPhase state.ProcessPhase

	// 목록 로딩 상태(메뉴 전환/새로고침 시 표시)
	phase2ListLoading bool
	phase3ListLoading bool
	phase2PrevStatus  string
	phase3PrevStatus  string
}

// NewAnalysisSelector 새 AnalysisSelector 생성
func NewAnalysisSelector(eventBus *state.EventBus, channelIdx int) *AnalysisSelector {
	a := &AnalysisSelector{
		eventBus:   eventBus,
		channelIdx: channelIdx,
	}

	// 2차 분석 섹션 (1차 완료 항목 선택)
	phase2Label := widget.NewLabelWithStyle("1차 완료 항목 (2차 분석 대상)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	a.phase2List = NewCompletedList(2) // Phase 2 이상이면 완료
	a.phase2List.SetOnSelect(func(record *domain.IssueRecord) {
		a.selectedPhase2Item = record
		if a.currentPhase != state.PhaseAIPlanGeneration {
			a.startPhase2.Enable()
		}
	})
	a.phase2List.SetOnDelete(func(record *domain.IssueRecord) {
		a.onDeletePhase2Item(record)
	})

	a.startPhase2 = widget.NewButton("AI 플랜 생성", a.onStartPhase2)
	a.startPhase2.Disable() // 초기에는 비활성화

	a.phase2LoadIcon = widget.NewIcon(theme.ViewRefreshIcon())
	a.phase2LoadIcon.Hide()
	a.phase2Status = widget.NewLabel("대기 중")

	// 새로고침 버튼 추가
	refreshPhase2Btn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		// 1차 완료 목록 새로고침 이벤트 발행
		a.eventBus.Publish(state.Event{
			Type:    state.EventIssueListRefresh,
			Channel: a.channelIdx,
			Data:    map[string]interface{}{"phase": 1},
		})
	})

	// 헤더: 라벨 + (새로고침 | 상태 | 버튼)
	phase2StatusBox := container.NewHBox(a.phase2LoadIcon, a.phase2Status)
	phase2Header := container.NewVBox(
		phase2Label,
		container.NewHBox(refreshPhase2Btn, phase2StatusBox, layout.NewSpacer(), a.startPhase2),
	)

	phase2Section := container.NewBorder(
		phase2Header, // Top - 라벨 + 상태 + 버튼
		nil,          // Bottom - 없음
		nil, nil,
		a.phase2List, // Center - 리스트
	)

	// 3차 분석 섹션 (2차 완료 항목 선택)
	phase3Label := widget.NewLabelWithStyle("2차 완료 항목 (3차 분석 대상)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	a.phase3List = NewCompletedList(3) // Phase 3 이상이면 완료
	a.phase3List.SetOnSelect(func(record *domain.IssueRecord) {
		a.selectedPhase3Item = record
		if a.currentPhase != state.PhaseAIExecution {
			a.startPhase3.Enable()
		}
	})
	a.phase3List.SetOnDelete(func(record *domain.IssueRecord) {
		a.onDeletePhase3Item(record)
	})

	a.startPhase3 = widget.NewButton("AI 실행", a.onStartPhase3)
	a.startPhase3.Disable() // 초기에는 비활성화

	a.phase3LoadIcon = widget.NewIcon(theme.ViewRefreshIcon())
	a.phase3LoadIcon.Hide()
	a.phase3Status = widget.NewLabel("대기 중")

	// 새로고침 버튼 추가
	refreshPhase3Btn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		// 2차 완료 목록 새로고침 이벤트 발행
		a.eventBus.Publish(state.Event{
			Type:    state.EventIssueListRefresh,
			Channel: a.channelIdx,
			Data:    map[string]interface{}{"phase": 2},
		})
	})

	// 헤더: 라벨 + (새로고침 | 상태 | 버튼)
	phase3StatusBox := container.NewHBox(a.phase3LoadIcon, a.phase3Status)
	phase3Header := container.NewVBox(
		phase3Label,
		container.NewHBox(refreshPhase3Btn, phase3StatusBox, layout.NewSpacer(), a.startPhase3),
	)

	phase3Section := container.NewBorder(
		phase3Header, // Top - 라벨 + 상태 + 버튼
		nil,          // Bottom - 없음
		nil, nil,
		a.phase3List, // Center - 리스트
	)

	// 전체 레이아웃: 2차/3차 섹션을 수직 분할로 배치 (각각 50%)
	split := container.NewVSplit(phase2Section, phase3Section)
	split.SetOffset(0.5)
	a.containerObj = split

	a.ExtendBaseWidget(a)
	a.subscribeToEvents()
	logger.Debug("NewAnalysisSelector created for channel %d", channelIdx)
	return a
}

// subscribeToEvents EventBus 이벤트 구독
func (a *AnalysisSelector) subscribeToEvents() {
	// Phase 변경 이벤트 구독
	a.eventBus.Subscribe(state.EventPhaseChange, func(event state.Event) {
		if event.Channel != a.channelIdx {
			return
		}

		if phase, ok := event.Data.(state.ProcessPhase); ok {
			a.currentPhase = phase
			a.runOnUIThread(func() {
				a.updateUIForPhase(phase)
			})
		}
	})

	// Phase1 완료 이벤트 구독 - Phase2 리스트 갱신
	a.eventBus.Subscribe(state.EventPhase1Complete, func(event state.Event) {
		if event.Channel != a.channelIdx {
			return
		}
		// Phase2 리스트 갱신 트리거
		a.runOnUIThread(func() {
			a.phase2Status.SetText("새 항목이 반영되었습니다 (체크 후 AI 플랜 생성)")
		})
	})

	// Phase2 완료 이벤트 구독 - Phase3 리스트 갱신
	a.eventBus.Subscribe(state.EventPhase2Complete, func(event state.Event) {
		if event.Channel != a.channelIdx {
			return
		}
		// Phase3 리스트 갱신 트리거
		a.runOnUIThread(func() {
			a.phase3Status.SetText("새 항목이 반영되었습니다 (체크 후 AI 실행)")

			// Phase2 완료 시 Phase2 버튼 다시 활성화 (다음 작업 가능)
			if a.selectedPhase2Item != nil {
				a.startPhase2.Enable()
			}
		})
	})

	// Phase3 완료 이벤트 구독
	a.eventBus.Subscribe(state.EventPhase3Complete, func(event state.Event) {
		if event.Channel != a.channelIdx {
			return
		}

		a.runOnUIThread(func() {
			// Phase3 완료 시 Phase3 버튼 다시 활성화 (다음 작업 가능)
			if a.selectedPhase3Item != nil {
				a.startPhase3.Enable()
			}
		})
	})
}

// runOnUIThread는 Fyne 메인 UI 스레드에서 위젯 업데이트를 실행한다.
// 테스트 환경처럼 현재 앱이 없을 때는 즉시 실행해 테스트 안정성을 유지한다.
func (a *AnalysisSelector) runOnUIThread(fn func()) {
	if fn == nil {
		return
	}
	if fyne.CurrentApp() == nil {
		fn()
		return
	}
	fyne.Do(fn)
}

// SetPhase1ListLoading은 1차 완료 목록(2차 분석 대상)의 로딩 표시를 제어한다.
func (a *AnalysisSelector) SetPhase1ListLoading(loading bool) {
	a.runOnUIThread(func() {
		if loading {
			if !a.phase2ListLoading {
				a.phase2PrevStatus = a.phase2Status.Text
			}
			a.phase2ListLoading = true
			a.phase2LoadIcon.Show()
			a.phase2Status.SetText("로딩 중...")
			return
		}

		if a.phase2ListLoading {
			a.phase2ListLoading = false
			a.phase2LoadIcon.Hide()
			if a.phase2Status.Text == "로딩 중..." {
				if a.phase2PrevStatus != "" {
					a.phase2Status.SetText(a.phase2PrevStatus)
				} else {
					a.phase2Status.SetText("대기 중")
				}
			}
		}
	})
}

// SetPhase2ListLoading은 2차 완료 목록(3차 분석 대상)의 로딩 표시를 제어한다.
func (a *AnalysisSelector) SetPhase2ListLoading(loading bool) {
	a.runOnUIThread(func() {
		if loading {
			if !a.phase3ListLoading {
				a.phase3PrevStatus = a.phase3Status.Text
			}
			a.phase3ListLoading = true
			a.phase3LoadIcon.Show()
			a.phase3Status.SetText("로딩 중...")
			return
		}

		if a.phase3ListLoading {
			a.phase3ListLoading = false
			a.phase3LoadIcon.Hide()
			if a.phase3Status.Text == "로딩 중..." {
				if a.phase3PrevStatus != "" {
					a.phase3Status.SetText(a.phase3PrevStatus)
				} else {
					a.phase3Status.SetText("대기 중")
				}
			}
		}
	})
}

// updateUIForPhase Phase에 따라 UI 업데이트
func (a *AnalysisSelector) updateUIForPhase(phase state.ProcessPhase) {
	switch phase {
	case state.PhaseAIPlanGeneration:
		a.phase2Status.SetText("AI 플랜 생성 중...")
		a.startPhase2.Disable()

	case state.PhaseAIPlanReady:
		a.phase2Status.SetText("🟢 AI 플랜 준비 완료")
		if a.selectedPhase2Item != nil {
			a.startPhase2.Enable()
		}

	case state.PhaseAIExecution:
		a.phase3Status.SetText("AI 플랜 실행 중...")
		a.startPhase3.Disable()

	case state.PhaseCompleted:
		a.phase3Status.SetText("🟢 AI 실행 완료")
		if a.selectedPhase3Item != nil {
			a.startPhase3.Enable()
		}

	case state.PhaseFailed:
		a.phase2Status.SetText("실패")
		a.phase3Status.SetText("실패")
		if a.selectedPhase2Item != nil {
			a.startPhase2.Enable()
		}
		if a.selectedPhase3Item != nil {
			a.startPhase3.Enable()
		}

	case state.PhaseIdle:
		a.phase2Status.SetText("대기 중")
		a.phase3Status.SetText("대기 중")
	}
}

// SetPhase1Items 1차 완료 항목 설정 (2차 분석 대상)
func (a *AnalysisSelector) SetPhase1Items(items []*domain.IssueRecord) {
	logger.Debug("SetPhase1Items: channel=%d, item_count=%d", a.channelIdx, len(items))
	a.phase2List.SetItems(items)
	a.selectedPhase2Item = nil
	a.startPhase2.Disable()
}

// SetPhase2Items 2차 완료 항목 설정 (3차 분석 대상)
func (a *AnalysisSelector) SetPhase2Items(items []*domain.IssueRecord) {
	logger.Debug("SetPhase2Items: channel=%d, item_count=%d", a.channelIdx, len(items))
	a.phase3List.SetItems(items)
	a.selectedPhase3Item = nil
	a.startPhase3.Disable()
}

// onStartPhase2 2차 분석 시작 핸들러
func (a *AnalysisSelector) onStartPhase2() {
	selectedItems := a.phase2List.GetSelectedItems()
	if len(selectedItems) == 0 {
		return
	}

	logger.Debug("onStartPhase2: starting phase2 analysis, channel=%d, selected_count=%d", a.channelIdx, len(selectedItems))
	for i, item := range selectedItems {
		logger.Debug("onStartPhase2: selected item[%d]: id=%d, key=%s", i, item.ID, item.IssueKey)
	}

	// Phase 변경: PhaseAIPlanGeneration으로 전환
	a.eventBus.PublishSync(state.Event{
		Type:    state.EventPhaseChange,
		Channel: a.channelIdx,
		Data:    state.PhaseAIPlanGeneration,
	})

	// 2차 분석 시작 이벤트 발행 (선택된 모든 항목)
	a.eventBus.PublishSync(state.Event{
		Type:    state.EventJobStarted,
		Channel: a.channelIdx,
		Data: map[string]interface{}{
			"phase":        "phase2",
			"issueRecords": selectedItems,
			"count":        len(selectedItems),
		},
	})

	// 버튼 비활성화 (Phase 완료 시 다시 활성화)
	a.startPhase2.Disable()
	a.phase2Status.SetText("AI 플랜 생성 중...")
}

// onStartPhase3 3차 분석 시작 핸들러
func (a *AnalysisSelector) onStartPhase3() {
	selectedItems := a.phase3List.GetSelectedItems()
	if len(selectedItems) == 0 {
		return
	}

	logger.Debug("onStartPhase3: starting phase3 analysis, channel=%d, selected_count=%d", a.channelIdx, len(selectedItems))
	for i, item := range selectedItems {
		logger.Debug("onStartPhase3: selected item[%d]: id=%d, key=%s", i, item.ID, item.IssueKey)
	}

	// Phase 변경: PhaseAIExecution으로 전환
	a.eventBus.PublishSync(state.Event{
		Type:    state.EventPhaseChange,
		Channel: a.channelIdx,
		Data:    state.PhaseAIExecution,
	})

	// 3차 분석 시작 이벤트 발행 (선택된 모든 항목)
	a.eventBus.PublishSync(state.Event{
		Type:    state.EventJobStarted,
		Channel: a.channelIdx,
		Data: map[string]interface{}{
			"phase":        "phase3",
			"issueRecords": selectedItems,
			"count":        len(selectedItems),
		},
	})

	// 버튼 비활성화 (Phase 완료 시 다시 활성화)
	a.startPhase3.Disable()
	a.phase3Status.SetText("AI 플랜 실행 중...")
}

// onDeletePhase2Item은 2차 섹션 목록 항목 삭제 요청 이벤트를 발행한다.
func (a *AnalysisSelector) onDeletePhase2Item(record *domain.IssueRecord) {
	if record == nil {
		return
	}
	a.eventBus.PublishSync(state.Event{
		Type:    state.EventIssueDeleteRequest,
		Channel: a.channelIdx,
		Data: map[string]interface{}{
			"listPhase":   2,
			"issueRecord": record,
		},
	})
}

// onDeletePhase3Item은 3차 섹션 목록 항목 삭제 요청 이벤트를 발행한다.
func (a *AnalysisSelector) onDeletePhase3Item(record *domain.IssueRecord) {
	if record == nil {
		return
	}
	a.eventBus.PublishSync(state.Event{
		Type:    state.EventIssueDeleteRequest,
		Channel: a.channelIdx,
		Data: map[string]interface{}{
			"listPhase":   3,
			"issueRecord": record,
		},
	})
}

// CreateRenderer AnalysisSelector 렌더러
func (a *AnalysisSelector) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.containerObj)
}

// CompletedList 완료된 이슈 목록 컴포넌트 (체크박스 지원)
type CompletedList struct {
	widget.BaseWidget

	containerObj   fyne.CanvasObject
	list           *widget.List
	items          []*domain.IssueRecord
	selected       map[int64]bool
	checkboxes     map[int]*widget.Check
	onSelect       func(*domain.IssueRecord)
	onDelete       func(*domain.IssueRecord)
	completedPhase int // 이 Phase 이상이면 완료로 간주
}

// NewCompletedList 새 CompletedList 생성
func NewCompletedList(completedPhase int) *CompletedList {
	c := &CompletedList{
		items:          make([]*domain.IssueRecord, 0),
		selected:       make(map[int64]bool),
		checkboxes:     make(map[int]*widget.Check),
		completedPhase: completedPhase,
	}

	c.list = widget.NewList(
		func() int { return len(c.items) },
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			icon := canvas.NewText("✓", completedListPrimaryColor)
			icon.TextSize = 14
			icon.Hide()
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextTruncate
			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			deleteBtn.Importance = widget.LowImportance
			// Border: Left=check/icon, Right=delete, Center=label (label이 남은 공간 전체 사용)
			return container.NewBorder(nil, nil, container.NewStack(check, icon), deleteBtn, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(c.items) {
				return
			}

			item := c.items[id]
			isCompleted := item.Phase >= c.completedPhase

			if border, ok := obj.(*fyne.Container); ok {
				// Border 컨테이너에서 자식 요소 찾기
				var stackContainer *fyne.Container
				var label *widget.Label
				var deleteBtn *widget.Button

				for _, child := range border.Objects {
					if stack, ok := child.(*fyne.Container); ok {
						stackContainer = stack
					}
					if l, ok := child.(*widget.Label); ok {
						label = l
					}
					if btn, ok := child.(*widget.Button); ok {
						deleteBtn = btn
					}
				}

				if stackContainer != nil {
					var check *widget.Check
					var icon *canvas.Text

					for _, child := range stackContainer.Objects {
						if c, ok := child.(*widget.Check); ok {
							check = c
						}
						if i, ok := child.(*canvas.Text); ok {
							icon = i
						}
					}

					if isCompleted {
						// 완료된 항목: 녹색 완료 아이콘 표시, 체크박스 숨김(선택 불가)
						if check != nil {
							check.Hide()
						}
						if icon != nil {
							icon.Show()
						}
					} else {
						// 미완료 항목: 체크박스 표시, 아이콘 숨김
						if check != nil {
							check.Show()
							check.SetChecked(c.selected[item.ID])

							currentItem := item
							check.OnChanged = func(checked bool) {
								if checked {
									c.selected[currentItem.ID] = true
									if c.onSelect != nil {
										c.onSelect(currentItem)
									}
								} else {
									delete(c.selected, currentItem.ID)
								}
							}

							c.checkboxes[id] = check
						}
						if icon != nil {
							icon.Hide()
						}
					}
				}

				if label != nil {
					label.SetText(item.Summary)
					// 완료 항목은 강조, 미완료는 기본 스타일
					if isCompleted {
						label.TextStyle = fyne.TextStyle{Bold: true}
						label.Importance = widget.HighImportance
					} else {
						label.TextStyle = fyne.TextStyle{}
						label.Importance = widget.MediumImportance
					}
					label.Refresh()
				}
				if deleteBtn != nil {
					currentItem := item
					deleteBtn.OnTapped = func() {
						if c.onDelete != nil {
							c.onDelete(currentItem)
						}
					}
				}
			}
		},
	)

	// 리스트 행 선택 상태가 텍스트 색상을 덮어쓰지 않도록 선택을 즉시 해제한다.
	c.list.OnSelected = func(id widget.ListItemID) {
		c.list.Unselect(id)
	}

	// CompletedList 내부에서만 완료 강조 색상을 연녹색 계열로 사용한다.
	c.containerObj = container.NewThemeOverride(
		container.NewStack(c.list),
		&completedListTheme{
			base:    theme.DefaultTheme(),
			primary: completedListPrimaryColor,
		},
	)
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer CompletedList 렌더러
func (c *CompletedList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.containerObj)
}

// MinSize 최소 크기 반환 - 리스트가 충분한 공간을 차지하도록 설정
func (c *CompletedList) MinSize() fyne.Size {
	return fyne.NewSize(250, 150)
}

// SetItems 항목 설정
func (c *CompletedList) SetItems(items []*domain.IssueRecord) {
	c.items = items
	c.selected = make(map[int64]bool)
	c.checkboxes = make(map[int]*widget.Check)
	c.list.Refresh()
}

// SetOnSelect 선택 콜백 설정
func (c *CompletedList) SetOnSelect(callback func(*domain.IssueRecord)) {
	c.onSelect = callback
}

// SetOnDelete 삭제 콜백 설정
func (c *CompletedList) SetOnDelete(callback func(*domain.IssueRecord)) {
	c.onDelete = callback
}

// Clear 목록 초기화
func (c *CompletedList) Clear() {
	c.items = make([]*domain.IssueRecord, 0)
	c.selected = make(map[int64]bool)
	c.checkboxes = make(map[int]*widget.Check)
	c.list.Refresh()
}

// GetSelectedIDs returns the IDs of selected items
func (c *CompletedList) GetSelectedIDs() []int64 {
	ids := make([]int64, 0, len(c.selected))
	for id := range c.selected {
		ids = append(ids, id)
	}
	return ids
}

// GetSelectedItems returns selected IssueRecords
func (c *CompletedList) GetSelectedItems() []*domain.IssueRecord {
	selectedItems := make([]*domain.IssueRecord, 0, len(c.selected))
	for _, item := range c.items {
		if c.selected[item.ID] {
			selectedItems = append(selectedItems, item)
		}
	}
	return selectedItems
}

// ClearSelection clears all selections
func (c *CompletedList) ClearSelection() {
	c.selected = make(map[int64]bool)
	for _, check := range c.checkboxes {
		check.SetChecked(false)
	}
}
