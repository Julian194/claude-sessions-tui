package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndRead(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	c := New(cachePath)

	// Test data
	entries := []Entry{
		{
			ID:      "session-001",
			Date:    time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			Project: "my-project",
			Summary: "First session summary",
		},
		{
			ID:      "session-002",
			Date:    time.Date(2025, 1, 14, 9, 0, 0, 0, time.UTC),
			Project: "another/project",
			Summary: "Second session",
		},
	}

	// Write
	err := c.Write(entries)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read back
	got, err := c.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if len(got) != len(entries) {
		t.Fatalf("Read() returned %d entries, want %d", len(got), len(entries))
	}

	// Verify entries
	for i, want := range entries {
		if got[i].ID != want.ID {
			t.Errorf("entries[%d].ID = %q, want %q", i, got[i].ID, want.ID)
		}
		if got[i].Project != want.Project {
			t.Errorf("entries[%d].Project = %q, want %q", i, got[i].Project, want.Project)
		}
		if got[i].Summary != want.Summary {
			t.Errorf("entries[%d].Summary = %q, want %q", i, got[i].Summary, want.Summary)
		}
		// Date comparison (only to minute precision)
		if got[i].Date.Format("2006-01-02 15:04") != want.Date.Format("2006-01-02 15:04") {
			t.Errorf("entries[%d].Date = %v, want %v", i, got[i].Date, want.Date)
		}
	}
}

func TestEscapeSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	c := New(cachePath)

	// Entry with special characters
	entries := []Entry{
		{
			ID:      "session-special",
			Date:    time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project: "project",
			Summary: "Summary with\ttab and\nnewline",
		},
	}

	err := c.Write(entries)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := c.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	// Tabs and newlines should be escaped to spaces
	expected := "Summary with tab and newline"
	if got[0].Summary != expected {
		t.Errorf("Summary = %q, want %q", got[0].Summary, expected)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	c := New(cachePath)

	if c.Exists() {
		t.Error("Exists() = true for non-existent file")
	}

	// Create the file
	c.Write([]Entry{})

	if !c.Exists() {
		t.Error("Exists() = false after Write()")
	}
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	c := New(cachePath)

	// Create the file
	c.Write([]Entry{})

	if !c.Exists() {
		t.Fatal("File should exist after Write()")
	}

	err := c.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if c.Exists() {
		t.Error("Exists() = true after Clear()")
	}
}

func TestClearNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent.tsv")

	c := New(cachePath)

	// Should not error on non-existent file
	err := c.Clear()
	if err != nil {
		t.Errorf("Clear() error = %v, want nil", err)
	}
}

func TestModTime(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	c := New(cachePath)

	// Create the file
	c.Write([]Entry{})

	mtime, err := c.ModTime()
	if err != nil {
		t.Fatalf("ModTime() error = %v", err)
	}

	// Should be recent
	if time.Since(mtime) > time.Minute {
		t.Errorf("ModTime() = %v, expected recent time", mtime)
	}
}

func TestReadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent.tsv")

	c := New(cachePath)

	_, err := c.Read()
	if err == nil {
		t.Error("Read() should error on non-existent file")
	}
}

func TestReadMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	// Write malformed content directly
	content := "good-id\t2025-01-15 10:00\tproject\tsummary\nbad line with not enough tabs\nanother-id\t2025-01-14 09:00\tproject2\tsummary2\n"
	os.WriteFile(cachePath, []byte(content), 0644)

	c := New(cachePath)
	entries, err := c.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	// Should skip malformed line and return 2 entries
	if len(entries) != 2 {
		t.Errorf("Read() returned %d entries, want 2", len(entries))
	}
}

func TestPath(t *testing.T) {
	c := New("/test/path/cache.tsv")
	if c.Path() != "/test/path/cache.tsv" {
		t.Errorf("Path() = %q, want %q", c.Path(), "/test/path/cache.tsv")
	}
}

func TestEscapeTSV(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"with\ttab", "with tab"},
		{"with\nnewline", "with newline"},
		{"with\rcarriage", "with carriage"},
		{"multiple\t\n\rchars", "multiple   chars"},
	}

	for _, tt := range tests {
		got := escapeTSV(tt.input)
		if got != tt.want {
			t.Errorf("escapeTSV(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
