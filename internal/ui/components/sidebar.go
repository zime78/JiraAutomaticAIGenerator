package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Sidebar 사이드바 컴포넌트
type Sidebar struct {
	widget.BaseWidget

	container *fyne.Container

	// 채널 목록
	channelList   *widget.List
	channelData   []ChannelInfo
	activeChannel int

	// 대기열 패널
	queuePanel *QueuePanel

	// 이력 패널
	historyPanel *HistoryPanel

	// 설정 버튼
	settingsBtn *widget.Button

	// 콜백
	onChannelSelect func(index int)
	onQueueSelect   func(jobID string)
	onHistorySelect func(jobID string)
	onSettingsClick func()
}

// ChannelInfo 채널 정보
type ChannelInfo struct {
	Index  int
	Name   string
	Status string
	Count  int // 대기 중인 작업 수
}

// NewSidebar 새 Sidebar 생성
func NewSidebar() *Sidebar {
	s := &Sidebar{
		channelData: []ChannelInfo{
			{Index: 0, Name: "채널 1", Status: "대기", Count: 0},
			{Index: 1, Name: "채널 2", Status: "대기", Count: 0},
			{Index: 2, Name: "채널 3", Status: "대기", Count: 0},
		},
		activeChannel: 0,
		queuePanel:    NewQueuePanel(),
		historyPanel:  NewHistoryPanel(),
		settingsBtn:   widget.NewButton("⚙️ 설정", nil),
	}

	s.settingsBtn.OnTapped = func() {
		if s.onSettingsClick != nil {
			s.onSettingsClick()
		}
	}

	// 채널 목록
	s.channelList = widget.NewList(
		func() int { return len(s.channelData) },
		func() fyne.CanvasObject {
			return NewChannelItem("", "", 0)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if item, ok := obj.(*ChannelItem); ok {
				ch := s.channelData[id]
				item.SetData(ch.Name, ch.Status, ch.Count)
				item.SetActive(id == s.activeChannel)
			}
		},
	)

	s.channelList.OnSelected = func(id widget.ListItemID) {
		s.activeChannel = id
		s.channelList.Refresh()
		if s.onChannelSelect != nil {
			s.onChannelSelect(id)
		}
	}

	// 컨테이너 구성
	channelSection := container.NewVBox(
		widget.NewLabelWithStyle("채널", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewVBox(s.channelList),
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
		channelSection,
		widget.NewSeparator(),
		queueSection,
		widget.NewSeparator(),
		historySection,
		widget.NewSeparator(),
		s.settingsBtn,
	)

	s.ExtendBaseWidget(s)
	return s
}

// CreateRenderer Sidebar 렌더러
func (s *Sidebar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.container)
}

// SetOnChannelSelect 채널 선택 콜백 설정
func (s *Sidebar) SetOnChannelSelect(callback func(index int)) {
	s.onChannelSelect = callback
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

// UpdateChannel 채널 상태 업데이트
func (s *Sidebar) UpdateChannel(index int, status string, count int) {
	if index >= 0 && index < len(s.channelData) {
		s.channelData[index].Status = status
		s.channelData[index].Count = count
		s.channelList.Refresh()
	}
}

// SetActiveChannel 활성 채널 설정
func (s *Sidebar) SetActiveChannel(index int) {
	if index >= 0 && index < len(s.channelData) {
		s.activeChannel = index
		s.channelList.Select(index)
	}
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
	s.historyPanel.AddItem(id, issueKey, status, duration)
}

// ChannelItem 채널 목록 아이템
type ChannelItem struct {
	widget.BaseWidget

	container   *fyne.Container
	nameLabel   *widget.Label
	statusLabel *widget.Label
	badge       *canvas.Text
	background  *canvas.Rectangle
	isActive    bool
}

// NewChannelItem 새 ChannelItem 생성
func NewChannelItem(name, status string, count int) *ChannelItem {
	c := &ChannelItem{
		nameLabel:   widget.NewLabel(name),
		statusLabel: widget.NewLabel(status),
		badge:       canvas.NewText("", theme.ForegroundColor()),
		background:  canvas.NewRectangle(color.Transparent),
	}

	c.badge.TextSize = 10
	c.statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	if count > 0 {
		c.badge.Text = fmt.Sprintf("(%d)", count)
	}

	c.container = container.NewHBox(
		c.nameLabel,
		c.badge,
	)

	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer ChannelItem 렌더러
func (c *ChannelItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.container)
}

// SetData 데이터 설정
func (c *ChannelItem) SetData(name, status string, count int) {
	c.nameLabel.SetText(name)
	c.statusLabel.SetText(status)
	if count > 0 {
		c.badge.Text = fmt.Sprintf("(%d)", count)
	} else {
		c.badge.Text = ""
	}
	c.badge.Refresh()
}

// SetActive 활성 상태 설정
func (c *ChannelItem) SetActive(active bool) {
	c.isActive = active
	if active {
		c.nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		c.nameLabel.TextStyle = fyne.TextStyle{}
	}
	c.nameLabel.Refresh()
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

// Clear 초기화
func (h *HistoryPanel) Clear() {
	h.items = make([]HistoryItem, 0)
	h.list.Refresh()
}
