package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

// mockAdapter implements adapters.Adapter for testing
type mockAdapter struct {
	sessions    []string
	sessionFile map[string]string
	metas       map[string]*adapters.SessionMeta
}

func (m *mockAdapter) Name() string                    { return "mock" }
func (m *mockAdapter) DataDir() string                 { return "/mock/data" }
func (m *mockAdapter) CacheDir() string                { return "/mock/cache" }
func (m *mockAdapter) ResumeCmd(id string) string      { return "mock resume " + id }
func (m *mockAdapter) ListSessions() ([]string, error) { return m.sessions, nil }
func (m *mockAdapter) GetSessionFile(id string) string { return m.sessionFile[id] }
func (m *mockAdapter) ExtractMeta(id string) (*adapters.SessionMeta, error) {
	if meta, ok := m.metas[id]; ok {
		return meta, nil
	}
	return nil, os.ErrNotExist
}
func (m *mockAdapter) GetSessionInfo(id string) (*adapters.SessionInfo, error) { return nil, nil }
func (m *mockAdapter) GetSummaries(id string) ([]string, error)                { return nil, nil }
func (m *mockAdapter) GetFilesTouched(id string) ([]string, error)             { return nil, nil }
func (m *mockAdapter) GetSlashCommands(id string) ([]string, error)            { return nil, nil }
func (m *mockAdapter) GetStats(id string) (*adapters.Stats, error)             { return nil, nil }
func (m *mockAdapter) GetFirstMessage(id string) (string, error)               { return "", nil }
func (m *mockAdapter) ExportMessages(id string) ([]adapters.Message, error)    { return nil, nil }
func (m *mockAdapter) BranchSession(id string) (string, error)                 { return "", nil }

func TestWriteAndRead(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	c := New(cachePath)

	// Test data
	entries := []Entry{
		{
			SessionID: "session-001",
			Date:      time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			Project:   "my-project",
			Summary:   "First session summary",
		},
		{
			SessionID: "session-002",
			Date:      time.Date(2025, 1, 14, 9, 0, 0, 0, time.UTC),
			Project:   "another/project",
			Summary:   "Second session",
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
		if got[i].SessionID != want.SessionID {
			t.Errorf("entries[%d].SessionID = %q, want %q", i, got[i].SessionID, want.SessionID)
		}
		if got[i].Project != want.Project {
			t.Errorf("entries[%d].Project = %q, want %q", i, got[i].Project, want.Project)
		}
		if got[i].Summary != want.Summary {
			t.Errorf("entries[%d].Summary = %q, want %q", i, got[i].Summary, want.Summary)
		}
		// Date comparison (only to second precision since we store unix timestamp)
		if got[i].Date.Unix() != want.Date.Unix() {
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
			SessionID: "session-special",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Summary with\ttab and\nnewline",
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

	// Write malformed content directly (new format: sid, time, project, summary, mtime, parent_sid, full_date)
	content := "good-id\t10:00\tproject\tsummary\t1705312800\t-\t2025-01-15\nbad line\nanother-id\t09:00\tproject2\tsummary2\t1705226400\t-\t2025-01-14\n"
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

func TestBuildFrom(t *testing.T) {
	tmpDir := t.TempDir()

	sessionFile := filepath.Join(tmpDir, "session1.jsonl")
	os.WriteFile(sessionFile, []byte(`{"type":"test"}`), 0644)

	mock := &mockAdapter{
		sessions:    []string{"session1", "session2"},
		sessionFile: map[string]string{"session1": sessionFile},
		metas: map[string]*adapters.SessionMeta{
			"session1": {
				ID:      "session1",
				Date:    time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
				Project: "test-project",
				Summary: "Test session",
			},
		},
	}

	entries, err := BuildFrom(mock)
	if err != nil {
		t.Fatalf("BuildFrom() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("BuildFrom() returned %d entries, want 1", len(entries))
	}

	if entries[0].SessionID != "session1" {
		t.Errorf("SessionID = %q, want %q", entries[0].SessionID, "session1")
	}
	if entries[0].Project != "test-project" {
		t.Errorf("Project = %q, want %q", entries[0].Project, "test-project")
	}
}

func TestBuildIncremental(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	sessionFile := filepath.Join(tmpDir, "session1.jsonl")
	os.WriteFile(sessionFile, []byte(`{"type":"test"}`), 0644)

	session2File := filepath.Join(tmpDir, "session2.jsonl")
	os.WriteFile(session2File, []byte(`{"type":"test2"}`), 0644)

	mock := &mockAdapter{
		sessions:    []string{"session1", "session2"},
		sessionFile: map[string]string{"session1": sessionFile, "session2": session2File},
		metas: map[string]*adapters.SessionMeta{
			"session1": {
				ID:      "session1",
				Date:    time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
				Project: "project1",
				Summary: "Session 1",
			},
			"session2": {
				ID:      "session2",
				Date:    time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
				Project: "project2",
				Summary: "Session 2",
			},
		},
	}

	existing := []Entry{
		{
			SessionID: "session1",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project1",
			Summary:   "Cached session 1",
		},
	}

	c := New(cachePath)
	c.Write(existing)

	time.Sleep(10 * time.Millisecond)
	os.Chtimes(session2File, time.Now(), time.Now())

	entries, err := BuildIncremental(mock, cachePath, existing)
	if err != nil {
		t.Fatalf("BuildIncremental() error = %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("BuildIncremental() returned %d entries, want 2", len(entries))
	}
}

func TestCacheBuildFrom(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	sessionFile := filepath.Join(tmpDir, "session1.jsonl")
	os.WriteFile(sessionFile, []byte(`{"type":"test"}`), 0644)

	mock := &mockAdapter{
		sessions:    []string{"session1"},
		sessionFile: map[string]string{"session1": sessionFile},
		metas: map[string]*adapters.SessionMeta{
			"session1": {
				ID:      "session1",
				Date:    time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
				Project: "test-project",
				Summary: "Test session",
			},
		},
	}

	c := New(cachePath)
	err := c.BuildFrom(mock)
	if err != nil {
		t.Fatalf("Cache.BuildFrom() error = %v", err)
	}

	if !c.Exists() {
		t.Error("Cache file should exist after BuildFrom")
	}

	entries, err := c.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Read() returned %d entries, want 1", len(entries))
	}
}

func TestParentSIDRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	c := New(cachePath)

	// Entry with parent session ID (branch)
	entries := []Entry{
		{
			SessionID: "parent-session",
			Date:      time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Parent session",
			ParentSID: "",
		},
		{
			SessionID: "child-session",
			Date:      time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			Project:   "project",
			Summary:   "Branched session",
			ParentSID: "parent-session",
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

	if len(got) != 2 {
		t.Fatalf("Read() returned %d entries, want 2", len(got))
	}

	// Parent should have empty ParentSID
	if got[0].ParentSID != "" {
		t.Errorf("Parent ParentSID = %q, want empty", got[0].ParentSID)
	}

	// Child should have parent's ID
	if got[1].ParentSID != "parent-session" {
		t.Errorf("Child ParentSID = %q, want %q", got[1].ParentSID, "parent-session")
	}
}
