package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

func TestFormatForDisplay_BranchesFollowParents(t *testing.T) {
	// Create test entries: parent and branch in different order
	entries := []cache.Entry{
		{
			SessionID: "parent-session",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project-a",
			Summary:   "Parent session",
			ParentSID: "", // root
		},
		{
			SessionID: "other-session",
			Date:      time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			Project:   "project-b",
			Summary:   "Unrelated session",
			ParentSID: "", // root
		},
		{
			SessionID: "child-session",
			Date:      time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			Project:   "project-a",
			Summary:   "Child session",
			ParentSID: "parent-session", // branch of parent-session
		},
	}

	result := formatForDisplay(entries)

	// Find positions - use HasPrefix to match session ID at start of line
	parentIdx := -1
	childIdx := -1
	otherIdx := -1

	for i, line := range result {
		if strings.HasPrefix(line, "parent-session\t") {
			parentIdx = i
		}
		if strings.HasPrefix(line, "child-session\t") {
			childIdx = i
		}
		if strings.HasPrefix(line, "other-session\t") {
			otherIdx = i
		}
	}

	if parentIdx == -1 {
		t.Fatalf("parent-session not found in output: %v", result)
	}
	if childIdx == -1 {
		t.Fatalf("child-session not found in output: %v", result)
	}
	if otherIdx == -1 {
		t.Fatalf("other-session not found in output: %v", result)
	}

	// Branch must appear immediately BEFORE parent (fzf reverses display, so branch shows below parent)
	if childIdx != parentIdx-1 {
		t.Errorf("child-session (idx %d) should appear immediately before parent-session (idx %d) for correct fzf display\nOutput: %v",
			childIdx, parentIdx, result)
	}

	// Branch line should have the └─ indicator
	if !strings.Contains(result[childIdx], "└─") {
		t.Errorf("child-session line should contain branch indicator '└─', got: %s", result[childIdx])
	}
}

func TestFormatForDisplay_MultipleBranchesUnderSameParent(t *testing.T) {
	entries := []cache.Entry{
		{
			SessionID: "the-parent",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Parent",
			ParentSID: "",
		},
		{
			SessionID: "branch-1",
			Date:      time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Branch 1",
			ParentSID: "the-parent",
		},
		{
			SessionID: "branch-2",
			Date:      time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Branch 2",
			ParentSID: "the-parent",
		},
	}

	result := formatForDisplay(entries)

	parentIdx := -1
	branch1Idx := -1
	branch2Idx := -1

	for i, line := range result {
		if strings.HasPrefix(line, "the-parent\t") {
			parentIdx = i
		}
		if strings.HasPrefix(line, "branch-1\t") {
			branch1Idx = i
		}
		if strings.HasPrefix(line, "branch-2\t") {
			branch2Idx = i
		}
	}

	if parentIdx == -1 || branch1Idx == -1 || branch2Idx == -1 {
		t.Fatalf("Missing entries: parent=%d, b1=%d, b2=%d\nOutput: %v", parentIdx, branch1Idx, branch2Idx, result)
	}

	// Both branches should appear BEFORE parent (fzf reverses display)
	if branch1Idx >= parentIdx {
		t.Errorf("branch-1 should appear before parent for correct fzf display")
	}
	if branch2Idx >= parentIdx {
		t.Errorf("branch-2 should appear before parent for correct fzf display")
	}
}

func TestFormatForDisplay_OrphanedBranchAtEnd(t *testing.T) {
	entries := []cache.Entry{
		{
			SessionID: "root-session",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Root",
			ParentSID: "",
		},
		{
			SessionID: "orphan-branch",
			Date:      time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Orphan",
			ParentSID: "deleted-parent", // parent doesn't exist
		},
	}

	result := formatForDisplay(entries)

	rootIdx := -1
	orphanIdx := -1

	for i, line := range result {
		if strings.Contains(line, "root-session") {
			rootIdx = i
		}
		if strings.Contains(line, "orphan-branch") {
			orphanIdx = i
		}
	}

	if rootIdx == -1 {
		t.Fatal("root-session not found")
	}
	if orphanIdx == -1 {
		t.Fatal("orphan-branch not found")
	}

	// Orphan should appear after all roots (at the end)
	if orphanIdx <= rootIdx {
		t.Errorf("orphan-branch should appear after root sessions")
	}

	// Orphan should still have branch indicator
	if !strings.Contains(result[orphanIdx], "└─") {
		t.Errorf("orphan branch should still have '└─' indicator")
	}
}

func TestFormatForDisplay_BranchNotUnderWrongParent(t *testing.T) {
	// This test reproduces the bug where branches appeared under wrong parents
	entries := []cache.Entry{
		{
			SessionID: "session-a",
			Date:      time.Date(2025, 1, 15, 15, 37, 0, 0, time.UTC), // 15:37
			Project:   "code/big",
			Summary:   "Session A",
			ParentSID: "",
		},
		{
			SessionID: "session-b",
			Date:      time.Date(2025, 1, 15, 18, 18, 0, 0, time.UTC), // 18:18
			Project:   "code/server/setup",
			Summary:   "Session B - actual parent",
			ParentSID: "",
		},
		{
			SessionID: "branch-of-b",
			Date:      time.Date(2025, 1, 15, 20, 48, 0, 0, time.UTC), // 20:48
			Project:   "code/server/setup",
			Summary:   "Branch of B",
			ParentSID: "session-b", // NOT session-a!
		},
	}

	result := formatForDisplay(entries)

	sessionAIdx := -1
	sessionBIdx := -1
	branchIdx := -1

	for i, line := range result {
		if strings.Contains(line, "session-a") {
			sessionAIdx = i
		}
		if strings.Contains(line, "session-b") && !strings.Contains(line, "branch") {
			sessionBIdx = i
		}
		if strings.Contains(line, "branch-of-b") {
			branchIdx = i
		}
	}

	if sessionAIdx == -1 || sessionBIdx == -1 || branchIdx == -1 {
		t.Fatalf("Missing entries: a=%d, b=%d, branch=%d", sessionAIdx, sessionBIdx, branchIdx)
	}

	// Branch must appear immediately BEFORE session-b (fzf reverses display)
	if branchIdx != sessionBIdx-1 {
		t.Errorf("branch-of-b (idx %d) should appear immediately before session-b (idx %d) for correct fzf display",
			branchIdx, sessionBIdx)
	}

	// Branch should NOT appear immediately before session-a (which would show it below session-a in fzf)
	if branchIdx == sessionAIdx-1 {
		t.Errorf("branch-of-b incorrectly appears before session-a in output (would show below session-a in fzf)")
	}
}
