package components

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"jira-ai-generator/internal/ui/state"
)

// LogViewer 로그 표시 컴포넌트
type LogViewer struct {
	widget.BaseWidget

	container  *fyne.Container
	list       *widget.List
	logs       []LogViewerEntry
	filter     state.LogLevel
	autoScroll bool
	collapsed  bool
	maxLines   int

	// 캐시
	filteredCache      []LogViewerEntry
	filteredCacheDirty bool

	// UI 요소
	filterSelect  *widget.Select
	clearBtn      *widget.Button
	collapseBtn   *widget.Button
	autoScrollChk *widget.Check
	countLabel    *widget.Label
}

// LogViewerEntry 로그 뷰어 항목
type LogViewerEntry struct {
	Timestamp time.Time
	Level     state.LogLevel
	Message   string
	Source    string
}

// NewLogViewer 새 LogViewer 생성
func NewLogViewer() *LogViewer {
	lv := &LogViewer{
		logs:               make([]LogViewerEntry, 0),
		filter:             state.LogDebug, // 모든 로그 표시
		autoScroll:         true,
		collapsed:          false,
		maxLines:           500,
		filteredCache:      make([]LogViewerEntry, 0),
		filteredCacheDirty: true,
	}

	// 필터 셀렉트
	lv.filterSelect = widget.NewSelect(
		[]string{"전체", "DEBUG", "INFO", "WARNING", "ERROR"},
		func(selected string) {
			switch selected {
			case "DEBUG":
				lv.filter = state.LogDebug
			case "INFO":
				lv.filter = state.LogInfo
			case "WARNING":
				lv.filter = state.LogWarning
			case "ERROR":
				lv.filter = state.LogError
			default:
				lv.filter = state.LogDebug
			}
			lv.invalidateCache()
			if lv.list != nil {
				lv.list.Refresh()
			}
		},
	)
	lv.filterSelect.SetSelected("전체")

	// 초기화 버튼
	lv.clearBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		lv.Clear()
	})

	// 접기/펼치기 버튼
	lv.collapseBtn = widget.NewButtonWithIcon("", theme.MenuDropUpIcon(), func() {
		lv.ToggleCollapse()
	})

	// 자동 스크롤 체크박스
	lv.autoScrollChk = widget.NewCheck("자동 스크롤", func(checked bool) {
		lv.autoScroll = checked
	})
	lv.autoScrollChk.SetChecked(true)

	// 로그 개수 라벨
	lv.countLabel = widget.NewLabel("0개")

	// 로그 목록
	lv.list = widget.NewList(
		func() int {
			return len(lv.getFilteredLogs())
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				canvas.NewText("00:00:00", theme.ForegroundColor()),
				canvas.NewText("[INFO]", color.RGBA{R: 59, G: 130, B: 246, A: 255}),
				widget.NewLabel("Message text here..."),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			logs := lv.getFilteredLogs()
			if id >= len(logs) {
				return
			}

			entry := logs[id]
			box := obj.(*fyne.Container)

			// 시간
			timeText := box.Objects[0].(*canvas.Text)
			timeText.Text = entry.Timestamp.Format("15:04:05")
			timeText.Refresh()

			// 레벨
			levelText := box.Objects[1].(*canvas.Text)
			levelText.Text = fmt.Sprintf("[%s]", entry.Level.String())
			levelText.Color = lv.getLevelColor(entry.Level)
			levelText.Refresh()

			// 메시지
			msgLabel := box.Objects[2].(*widget.Label)
			if entry.Source != "" {
				msgLabel.SetText(fmt.Sprintf("[%s] %s", entry.Source, entry.Message))
			} else {
				msgLabel.SetText(entry.Message)
			}
		},
	)

	// 헤더
	header := container.NewHBox(
		widget.NewLabelWithStyle("📋 로그", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		lv.countLabel,
		widget.NewLabel(" | "),
		lv.filterSelect,
		lv.autoScrollChk,
		lv.clearBtn,
		lv.collapseBtn,
	)

	// 스크롤 영역
	listScroll := container.NewScroll(lv.list)
	listScroll.SetMinSize(fyne.NewSize(0, 150))

	lv.container = container.NewBorder(
		header,
		nil, nil, nil,
		listScroll,
	)

	lv.ExtendBaseWidget(lv)
	return lv
}

// CreateRenderer LogViewer 렌더러
func (lv *LogViewer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(lv.container)
}

// AddLog 로그 추가
func (lv *LogViewer) AddLog(level state.LogLevel, message, source string) {
	entry := LogViewerEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
	}

	lv.logs = append(lv.logs, entry)

	// 최대 개수 제한
	if len(lv.logs) > lv.maxLines {
		lv.logs = lv.logs[len(lv.logs)-lv.maxLines:]
	}

	lv.invalidateCache()
	lv.updateCountLabel()
	lv.list.Refresh()

	// 자동 스크롤
	if lv.autoScroll {
		filteredLogs := lv.getFilteredLogs()
		if len(filteredLogs) > 0 {
			lv.list.ScrollToBottom()
		}
	}
}

// AddLogEntry LogEntry로 로그 추가
func (lv *LogViewer) AddLogEntry(entry state.LogEntry) {
	lv.AddLog(entry.Level, entry.Message, entry.Source)
}

// Clear 로그 초기화
func (lv *LogViewer) Clear() {
	lv.logs = lv.logs[:0]
	lv.invalidateCache()
	lv.updateCountLabel()
	lv.list.Refresh()
}

// ToggleCollapse 접기/펼치기 토글
func (lv *LogViewer) ToggleCollapse() {
	lv.collapsed = !lv.collapsed

	if lv.collapsed {
		lv.collapseBtn.SetIcon(theme.MenuDropDownIcon())
		lv.container.Objects[1].Hide()
	} else {
		lv.collapseBtn.SetIcon(theme.MenuDropUpIcon())
		lv.container.Objects[1].Show()
	}

	lv.Refresh()
}

// SetFilter 필터 설정
func (lv *LogViewer) SetFilter(level state.LogLevel) {
	lv.filter = level
	switch level {
	case state.LogDebug:
		lv.filterSelect.SetSelected("전체")
	case state.LogInfo:
		lv.filterSelect.SetSelected("INFO")
	case state.LogWarning:
		lv.filterSelect.SetSelected("WARNING")
	case state.LogError:
		lv.filterSelect.SetSelected("ERROR")
	}
	lv.list.Refresh()
}

// SetAutoScroll 자동 스크롤 설정
func (lv *LogViewer) SetAutoScroll(enabled bool) {
	lv.autoScroll = enabled
	lv.autoScrollChk.SetChecked(enabled)
}

// getFilteredLogs 필터링된 로그 조회 (캐시 사용)
func (lv *LogViewer) getFilteredLogs() []LogViewerEntry {
	if lv.filter == state.LogDebug {
		return lv.logs
	}

	// 캐시가 유효하면 반환
	if !lv.filteredCacheDirty {
		return lv.filteredCache
	}

	// 캐시 재구성
	lv.filteredCache = lv.filteredCache[:0] // 재사용
	for _, log := range lv.logs {
		if log.Level >= lv.filter {
			lv.filteredCache = append(lv.filteredCache, log)
		}
	}
	lv.filteredCacheDirty = false
	return lv.filteredCache
}

// invalidateCache 캐시 무효화
func (lv *LogViewer) invalidateCache() {
	lv.filteredCacheDirty = true
}

// updateCountLabel 개수 라벨 업데이트
func (lv *LogViewer) updateCountLabel() {
	total := len(lv.logs)
	filtered := len(lv.getFilteredLogs())

	if total == filtered {
		lv.countLabel.SetText(fmt.Sprintf("%d개", total))
	} else {
		lv.countLabel.SetText(fmt.Sprintf("%d/%d개", filtered, total))
	}
}

// getLevelColor 로그 레벨별 색상
func (lv *LogViewer) getLevelColor(level state.LogLevel) color.Color {
	switch level {
	case state.LogDebug:
		return color.RGBA{R: 156, G: 163, B: 175, A: 255} // 회색
	case state.LogInfo:
		return color.RGBA{R: 59, G: 130, B: 246, A: 255} // 파란색
	case state.LogWarning:
		return color.RGBA{R: 245, G: 158, B: 11, A: 255} // 주황색
	case state.LogError:
		return color.RGBA{R: 239, G: 68, B: 68, A: 255} // 빨간색
	default:
		return theme.ForegroundColor()
	}
}

// GetLogs 모든 로그 조회
func (lv *LogViewer) GetLogs() []LogViewerEntry {
	result := make([]LogViewerEntry, len(lv.logs))
	copy(result, lv.logs)
	return result
}

// GetLogCount 로그 개수 조회
func (lv *LogViewer) GetLogCount() int {
	return len(lv.logs)
}
