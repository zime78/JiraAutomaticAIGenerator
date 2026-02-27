package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/logger"
	"jira-ai-generator/internal/ui/state"
)

// Sidebar 사이드바 컴포넌트
type Sidebar struct {
	widget.BaseWidget

	container *fyne.Container

	// 1차 분석 UI
	urlEntry   *widget.Entry
	analyzeBtn *widget.Button
	eventBus   *state.EventBus

	// 대기열 패널
	queuePanel *QueuePanel

	// 이력 패널
	historyPanel *HistoryPanel

	// 설정 버튼
	settingsBtn *widget.Button

	// 콜백
	onQueueSelect   func(jobID string)
	onHistorySelect func(jobID string)
	onSettingsClick func()
}

// NewSidebar 새 Sidebar 생성
func NewSidebar(eventBus *state.EventBus) *Sidebar {
	s := &Sidebar{
		eventBus:     eventBus,
		urlEntry:     widget.NewEntry(),
		queuePanel:   NewQueuePanel(),
		historyPanel: NewHistoryPanel(),
		settingsBtn:  widget.NewButton("⚙️ 설정", nil),
	}

	// URL 입력 필드 설정
	s.urlEntry.SetPlaceHolder("Jira URL 입력...")

	// 분석 시작 버튼 생성
	s.analyzeBtn = widget.NewButton("분석 시작", s.onAnalyzeClick)

	s.settingsBtn.OnTapped = func() {
		if s.onSettingsClick != nil {
			s.onSettingsClick()
		}
	}

	// 컨테이너 구성
	// 1차 분석 섹션
	analyzeSection := container.NewVBox(
		widget.NewLabelWithStyle("🔍 1차 분석", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		s.urlEntry,
		s.analyzeBtn,
	)

	queueSection := container.NewVBox(
		widget.NewLabelWithStyle("📋 대기열", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		s.queuePanel,
	)

	historySection := container.NewVBox(
		widget.NewLabelWithStyle("📁 이력", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		s.historyPanel,
	)

	s.container = container.NewVBox(
		s.settingsBtn,
		widget.NewSeparator(),
		analyzeSection,
		widget.NewSeparator(),
		queueSection,
		widget.NewSeparator(),
		historySection,
	)

	s.ExtendBaseWidget(s)
	logger.Debug("NewSidebar created")
	return s
}

// CreateRenderer Sidebar 렌더러
func (s *Sidebar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.container)
}

// onAnalyzeClick 분석 시작 버튼 클릭 핸들러
func (s *Sidebar) onAnalyzeClick() {
	url := s.urlEntry.Text
	if url == "" {
		return
	}

	logger.Debug("onAnalyzeClick: url=%s", url)

	// EventSidebarAction 발행
	s.eventBus.Publish(state.Event{
		Type:    state.EventSidebarAction,
		Channel: 0,
		Data: map[string]interface{}{
			"action": "analyze",
			"url":    url,
		},
	})
}

// SetOnQueueSelect 대기열 선택 콜백 설정
func (s *Sidebar) SetOnQueueSelect(callback func(jobID string)) {
	s.onQueueSelect = callback
	s.queuePanel.SetOnSelect(callback)
}

// SetOnHistorySelect 이력 선택 콜백 설정
func (s *Sidebar) SetOnHistorySelect(callback func(jobID string)) {
	s.onHistorySelect = callback
	s.historyPanel.SetOnSelect(callback)
}

// SetOnSettingsClick 설정 버튼 콜백 설정
func (s *Sidebar) SetOnSettingsClick(callback func()) {
	s.onSettingsClick = callback
}

// AddQueueItem 대기열에 항목 추가
func (s *Sidebar) AddQueueItem(id, issueKey, status string) {
	s.queuePanel.AddItem(id, issueKey, status)
}

// RemoveQueueItem 대기열에서 항목 제거
func (s *Sidebar) RemoveQueueItem(id string) {
	s.queuePanel.RemoveItem(id)
}

// ClearQueue 대기열 초기화
func (s *Sidebar) ClearQueue() {
	s.queuePanel.Clear()
}

// AddHistoryItem 이력에 항목 추가
func (s *Sidebar) AddHistoryItem(id, issueKey, status, duration string) {
	fyne.Do(func() {
		s.historyPanel.AddItem(id, issueKey, status, duration)
	})
}

// RemoveHistoryItem 이력에서 항목 제거
func (s *Sidebar) RemoveHistoryItem(id string) {
	fyne.Do(func() {
		s.historyPanel.RemoveItem(id)
	})
}

// QueuePanel 대기열 패널
type QueuePanel struct {
	widget.BaseWidget

	container *fyne.Container
	list      *widget.List
	items     []QueueItem
	onSelect  func(jobID string)
}

// QueueItem 대기열 아이템
type QueueItem struct {
	ID       string
	IssueKey string
	Status   string
}

// NewQueuePanel 새 QueuePanel 생성
func NewQueuePanel() *QueuePanel {
	q := &QueuePanel{
		items: make([]QueueItem, 0),
	}

	q.list = widget.NewList(
		func() int { return len(q.items) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if label, ok := obj.(*widget.Label); ok {
				item := q.items[id]
				label.SetText(fmt.Sprintf("• %s (%s)", item.IssueKey, item.Status))
			}
		},
	)

	q.list.OnSelected = func(id widget.ListItemID) {
		if q.onSelect != nil && id < len(q.items) {
			q.onSelect(q.items[id].ID)
		}
	}

	q.container = container.NewVBox(q.list)
	q.ExtendBaseWidget(q)
	return q
}

// CreateRenderer QueuePanel 렌더러
func (q *QueuePanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(q.container)
}

// SetOnSelect 선택 콜백 설정
func (q *QueuePanel) SetOnSelect(callback func(jobID string)) {
	q.onSelect = callback
}

// AddItem 항목 추가
func (q *QueuePanel) AddItem(id, issueKey, status string) {
	q.items = append(q.items, QueueItem{ID: id, IssueKey: issueKey, Status: status})
	q.list.Refresh()
}

// RemoveItem 항목 제거
func (q *QueuePanel) RemoveItem(id string) {
	for i, item := range q.items {
		if item.ID == id {
			q.items = append(q.items[:i], q.items[i+1:]...)
			break
		}
	}
	q.list.Refresh()
}

// Clear 초기화
func (q *QueuePanel) Clear() {
	q.items = make([]QueueItem, 0)
	q.list.Refresh()
}

// HistoryPanel 이력 패널
type HistoryPanel struct {
	widget.BaseWidget

	container *fyne.Container
	list      *widget.List
	items     []HistoryItem
	onSelect  func(jobID string)
}

// HistoryItem 이력 아이템
type HistoryItem struct {
	ID       string
	IssueKey string
	Status   string
	Duration string
}

// NewHistoryPanel 새 HistoryPanel 생성
func NewHistoryPanel() *HistoryPanel {
	h := &HistoryPanel{
		items: make([]HistoryItem, 0),
	}

	h.list = widget.NewList(
		func() int { return len(h.items) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if label, ok := obj.(*widget.Label); ok {
				item := h.items[id]
				statusIcon := "✅"
				if item.Status == "failed" {
					statusIcon = "❌"
				}
				label.SetText(fmt.Sprintf("%s %s (%s)", statusIcon, item.IssueKey, item.Duration))
			}
		},
	)

	h.list.OnSelected = func(id widget.ListItemID) {
		if h.onSelect != nil && id < len(h.items) {
			h.onSelect(h.items[id].ID)
		}
	}

	h.container = container.NewVBox(h.list)
	h.ExtendBaseWidget(h)
	return h
}

// CreateRenderer HistoryPanel 렌더러
func (h *HistoryPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.container)
}

// SetOnSelect 선택 콜백 설정
func (h *HistoryPanel) SetOnSelect(callback func(jobID string)) {
	h.onSelect = callback
}

// AddItem 항목 추가 (최신 항목이 위로)
func (h *HistoryPanel) AddItem(id, issueKey, status, duration string) {
	item := HistoryItem{ID: id, IssueKey: issueKey, Status: status, Duration: duration}
	h.items = append([]HistoryItem{item}, h.items...)

	// 최대 50개 유지
	if len(h.items) > 50 {
		h.items = h.items[:50]
	}

	h.list.Refresh()
}

// RemoveItem 항목 제거
func (h *HistoryPanel) RemoveItem(id string) {
	for i, item := range h.items {
		if item.ID == id {
			h.items = append(h.items[:i], h.items[i+1:]...)
			break
		}
	}
	h.list.Refresh()
}

// Clear 초기화
func (h *HistoryPanel) Clear() {
	h.items = make([]HistoryItem, 0)
	h.list.Refresh()
}
