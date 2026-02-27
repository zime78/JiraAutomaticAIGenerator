package components

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"jira-ai-generator/internal/ui/state"
)

// TestSidebar_OnAnalyzeClickUsesActiveChannel은 분석 시작 이벤트가 현재 선택 채널로 발행되는지 검증한다.
func TestSidebar_OnAnalyzeClickUsesActiveChannel(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	eb := state.NewEventBus()
	sidebar := NewSidebar(eb, 0)
	sidebar.urlEntry.SetText("https://example.atlassian.net/browse/TEST-100")
	sidebar.activeChannel = 2

	eventCh := make(chan state.Event, 1)
	eb.Subscribe(state.EventSidebarAction, func(event state.Event) {
		eventCh <- event
	})

	sidebar.onAnalyzeClick()

	select {
	case got := <-eventCh:
		if got.Channel != 2 {
			t.Fatalf("expected channel=2, got %d", got.Channel)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected EventSidebarAction event")
	}
}

// TestHistoryPanel_RemoveItem은 지정한 이력 항목이 목록에서 제거되는지 검증한다.
func TestHistoryPanel_RemoveItem(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	panel := NewHistoryPanel()
	panel.AddItem("0:10", "ITSM-10", "completed", "")
	panel.AddItem("0:11", "ITSM-11", "completed", "")

	if got := len(panel.items); got != 2 {
		t.Fatalf("expected 2 items before remove, got %d", got)
	}

	panel.RemoveItem("0:10")

	if got := len(panel.items); got != 1 {
		t.Fatalf("expected 1 item after remove, got %d", got)
	}
	if panel.items[0].ID != "0:11" {
		t.Fatalf("expected remaining item id=0:11, got %s", panel.items[0].ID)
	}
}
