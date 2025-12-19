package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

func TestFormatForDisplay_ChildIndicator(t *testing.T) {
	entries := []cache.Entry{
		{
			SessionID: "parent-session",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project-a",
			Summary:   "Parent session",
			ParentSID: "",
		},
		{
			SessionID: "child-session",
			Date:      time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			Project:   "project-a",
			Summary:   "Child session",
			ParentSID: "parent-session",
		},
	}

	result := formatForDisplay(entries)

	var parentLine, childLine string
	for _, line := range result {
		if strings.HasPrefix(line, "parent-session\t") {
			parentLine = line
		}
		if strings.HasPrefix(line, "child-session\t") {
			childLine = line
		}
	}

	if parentLine == "" {
		t.Fatal("parent-session not found")
	}
	if childLine == "" {
		t.Fatal("child-session not found")
	}

	if strings.Contains(parentLine, "↳") {
		t.Error("parent session should NOT have child indicator")
	}
	if !strings.Contains(childLine, "↳") {
		t.Errorf("child session should have ↳ indicator, got: %s", childLine)
	}
}

func TestFormatForDisplay_DateHeaders(t *testing.T) {
	entries := []cache.Entry{
		{
			SessionID: "session-day1",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Day 1",
			ParentSID: "",
		},
		{
			SessionID: "session-day2",
			Date:      time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Day 2",
			ParentSID: "",
		},
	}

	result := formatForDisplay(entries)

	headerCount := 0
	for _, line := range result {
		if strings.HasPrefix(line, "---HEADER---") {
			headerCount++
		}
	}

	if headerCount != 2 {
		t.Errorf("expected 2 date headers, got %d", headerCount)
	}
}

func TestFormatForDisplay_OrphanedChildStillHasIndicator(t *testing.T) {
	entries := []cache.Entry{
		{
			SessionID: "root-session",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Root",
			ParentSID: "",
		},
		{
			SessionID: "orphan-child",
			Date:      time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Orphan",
			ParentSID: "deleted-parent",
		},
	}

	result := formatForDisplay(entries)

	var orphanLine string
	for _, line := range result {
		if strings.HasPrefix(line, "orphan-child\t") {
			orphanLine = line
		}
	}

	if orphanLine == "" {
		t.Fatal("orphan-child not found")
	}

	if !strings.Contains(orphanLine, "↳") {
		t.Errorf("orphan child should still have ↳ indicator, got: %s", orphanLine)
	}
}

func TestFormatForDisplay_EmptyEntries(t *testing.T) {
	result := formatForDisplay(nil)
	if result != nil {
		t.Errorf("expected nil for empty entries, got %v", result)
	}

	result = formatForDisplay([]cache.Entry{})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestFormatForDisplay_PreservesOrder(t *testing.T) {
	entries := []cache.Entry{
		{SessionID: "first", Date: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC), Project: "p", Summary: "First"},
		{SessionID: "second", Date: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC), Project: "p", Summary: "Second"},
		{SessionID: "third", Date: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), Project: "p", Summary: "Third"},
	}

	result := formatForDisplay(entries)

	var indices []int
	for i, line := range result {
		if strings.HasPrefix(line, "first\t") {
			indices = append(indices, i)
		} else if strings.HasPrefix(line, "second\t") {
			indices = append(indices, i)
		} else if strings.HasPrefix(line, "third\t") {
			indices = append(indices, i)
		}
	}

	if len(indices) != 3 {
		t.Fatalf("expected 3 sessions, found %d", len(indices))
	}

	if indices[0] > indices[1] || indices[1] > indices[2] {
		t.Errorf("sessions not in expected order: %v", indices)
	}
}
