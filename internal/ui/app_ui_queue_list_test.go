package ui

import "testing"

// TestResolveQueueListItemView_StatusMapping은 큐 아이템이 상태별로 올바른 아이콘/텍스트로 매핑되는지 검증한다.
func TestResolveQueueListItemView_StatusMapping(t *testing.T) {
	queue := &AnalysisQueue{
		Current:   &AnalysisJob{IssueKey: "ITSM-200"},
		Pending:   []*AnalysisJob{{IssueKey: "ITSM-201"}},
		Completed: []*AnalysisJob{{IssueKey: "ITSM-202"}},
		Failed:    []*AnalysisJob{{IssueKey: "ITSM-203"}},
		Cancelled: []*AnalysisJob{{IssueKey: "ITSM-204"}},
	}

	cases := []struct {
		id       int
		wantIcon string
		wantText string
	}{
		{id: 0, wantIcon: "▶", wantText: "ITSM-200"},
		{id: 1, wantIcon: "⏳", wantText: "ITSM-201"},
		{id: 2, wantIcon: "✓", wantText: "ITSM-202 (완료)"},
		{id: 3, wantIcon: "✗", wantText: "ITSM-203 (실패)"},
		{id: 4, wantIcon: "⏹", wantText: "ITSM-204 (중단)"},
	}

	for _, tc := range cases {
		got, ok := resolveQueueListItemView(queue, tc.id)
		if !ok {
			t.Fatalf("resolveQueueListItemView(%d) returned ok=false", tc.id)
		}
		if got.Icon != tc.wantIcon {
			t.Fatalf("id=%d icon=%q, want %q", tc.id, got.Icon, tc.wantIcon)
		}
		if got.Text != tc.wantText {
			t.Fatalf("id=%d text=%q, want %q", tc.id, got.Text, tc.wantText)
		}
	}
}
