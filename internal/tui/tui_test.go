package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

func TestSessionItem_FilterValue(t *testing.T) {
	item := SessionItem{
		entry: cache.Entry{
			SessionID: "abc-123",
			Project:   "my-project",
			Summary:   "Test summary",
		},
	}

	fv := item.FilterValue()

	if fv == "" {
		t.Error("FilterValue should not be empty")
	}
	if !contains(fv, "my-project") {
		t.Error("FilterValue should contain project")
	}
	if !contains(fv, "Test summary") {
		t.Error("FilterValue should contain summary")
	}
	if !contains(fv, "abc-123") {
		t.Error("FilterValue should contain session ID")
	}
}

func TestSessionItem_Title(t *testing.T) {
	// Regular session
	item := SessionItem{
		entry: cache.Entry{
			Date:    time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC),
			Project: "test-project",
		},
		isPinned: false,
	}

	title := item.Title()
	if !contains(title, "14:30") {
		t.Errorf("Title should contain time, got: %s", title)
	}
	if !contains(title, "test-project") {
		t.Errorf("Title should contain project, got: %s", title)
	}

	// Pinned session
	item.isPinned = true
	title = item.Title()
	if !contains(title, "★") {
		t.Errorf("Pinned session should have star indicator, got: %s", title)
	}

	// Child/agent session
	item.isPinned = false
	item.depth = 1
	item.isAgent = true
	title = item.Title()
	if !contains(title, "↳") {
		t.Errorf("Child session should have arrow indicator, got: %s", title)
	}
}

func TestPins_ToggleAndPersistence(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()

	pins := NewPins(tmpDir)

	// Toggle on
	result := pins.Toggle("session-1")
	if !result {
		t.Error("Toggle should return true when pinning")
	}
	if !pins.IsPinned("session-1") {
		t.Error("Session should be pinned")
	}
	if pins.Count() != 1 {
		t.Errorf("Count should be 1, got %d", pins.Count())
	}

	// Toggle off
	result = pins.Toggle("session-1")
	if result {
		t.Error("Toggle should return false when unpinning")
	}
	if pins.IsPinned("session-1") {
		t.Error("Session should not be pinned")
	}
	if pins.Count() != 0 {
		t.Errorf("Count should be 0, got %d", pins.Count())
	}
}

func TestPins_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and save pins
	pins1 := NewPins(tmpDir)
	pins1.Toggle("session-a")
	pins1.Toggle("session-b")
	if err := pins1.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	pinsFile := filepath.Join(tmpDir, "pinned-sessions.txt")
	if _, err := os.Stat(pinsFile); os.IsNotExist(err) {
		t.Fatal("Pins file should exist after save")
	}

	// Load into new instance
	pins2 := NewPins(tmpDir)
	if err := pins2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !pins2.IsPinned("session-a") {
		t.Error("session-a should be pinned after load")
	}
	if !pins2.IsPinned("session-b") {
		t.Error("session-b should be pinned after load")
	}
	if pins2.Count() != 2 {
		t.Errorf("Count should be 2 after load, got %d", pins2.Count())
	}
}

func TestPins_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	pins := NewPins(tmpDir)

	// Should not error on missing file
	if err := pins.Load(); err != nil {
		t.Errorf("Load should not error on missing file: %v", err)
	}
	if pins.Count() != 0 {
		t.Errorf("Count should be 0 for missing file, got %d", pins.Count())
	}
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Check that bindings are defined
	if len(km.Up.Keys()) == 0 {
		t.Error("Up keys should be defined")
	}
	if len(km.Down.Keys()) == 0 {
		t.Error("Down keys should be defined")
	}
	if len(km.Select.Keys()) == 0 {
		t.Error("Select keys should be defined")
	}
	if len(km.Quit.Keys()) == 0 {
		t.Error("Quit keys should be defined")
	}
}

func TestKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.ShortHelp()

	if len(help) == 0 {
		t.Error("ShortHelp should return bindings")
	}
}

func TestKeyMap_FullHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.FullHelp()

	if len(help) == 0 {
		t.Error("FullHelp should return binding groups")
	}
	for i, group := range help {
		if len(group) == 0 {
			t.Errorf("FullHelp group %d should not be empty", i)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
