package components

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"jira-ai-generator/internal/domain"
)

func TestNewCompletedList(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	list := NewCompletedList(2)

	if list == nil {
		t.Fatal("NewCompletedList() returned nil")
	}

	if list.items == nil {
		t.Error("items should be initialized")
	}

	if list.selected == nil {
		t.Error("selected map should be initialized")
	}
}

func TestCompletedList_SetItems(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	list := NewCompletedList(2)

	items := []*domain.IssueRecord{
		{
			ID:        1,
			IssueKey:  "TEST-1",
			Summary:   "Test Issue 1",
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			IssueKey:  "TEST-2",
			Summary:   "Test Issue 2",
			UpdatedAt: time.Now(),
		},
	}

	list.SetItems(items)

	if len(list.items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(list.items))
	}

	if list.items[0].IssueKey != "TEST-1" {
		t.Errorf("Expected first item to be TEST-1, got %s", list.items[0].IssueKey)
	}
}

func TestCompletedList_GetSelectedIDs(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	list := NewCompletedList(2)

	items := []*domain.IssueRecord{
		{ID: 1, IssueKey: "TEST-1"},
		{ID: 2, IssueKey: "TEST-2"},
		{ID: 3, IssueKey: "TEST-3"},
	}

	list.SetItems(items)

	// Select items 1 and 3
	list.selected[1] = true
	list.selected[3] = true

	selectedIDs := list.GetSelectedIDs()

	if len(selectedIDs) != 2 {
		t.Errorf("Expected 2 selected IDs, got %d", len(selectedIDs))
	}

	// Check that selected IDs contain 1 and 3
	found := make(map[int64]bool)
	for _, id := range selectedIDs {
		found[id] = true
	}

	if !found[1] || !found[3] {
		t.Error("Expected IDs 1 and 3 to be selected")
	}
}

func TestCompletedList_EmptySelection(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	list := NewCompletedList(2)

	items := []*domain.IssueRecord{
		{ID: 1, IssueKey: "TEST-1"},
	}

	list.SetItems(items)

	selectedIDs := list.GetSelectedIDs()

	if len(selectedIDs) != 0 {
		t.Errorf("Expected 0 selected IDs, got %d", len(selectedIDs))
	}
}
