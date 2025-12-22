package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testDataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "testdata", "storage")
}

func setupTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	return New(testDataDir(t))
}

func TestNew(t *testing.T) {
	a := New("/test/path")
	if a.DataDir() != "/test/path" {
		t.Errorf("DataDir() = %q, want %q", a.DataDir(), "/test/path")
	}
	if a.Name() != "opencode" {
		t.Errorf("Name() = %q, want %q", a.Name(), "opencode")
	}
}

func TestListSessions(t *testing.T) {
	a := setupTestAdapter(t)

	sessions, err := a.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("ListSessions() returned %d sessions, want 2", len(sessions))
	}

	// Sessions are sorted by mtime, so subagent should be first (newer)
	found := make(map[string]bool)
	for _, s := range sessions {
		found[s] = true
	}
	if !found["ses_abc123"] {
		t.Error("ListSessions() missing ses_abc123")
	}
	if !found["ses_subagent456"] {
		t.Error("ListSessions() missing ses_subagent456")
	}
}

func TestGetSessionFile(t *testing.T) {
	a := setupTestAdapter(t)

	path := a.GetSessionFile("ses_abc123")
	if path == "" {
		t.Fatal("GetSessionFile() returned empty path")
	}
	if filepath.Base(path) != "ses_abc123.json" {
		t.Errorf("GetSessionFile() = %q, want ses_abc123.json", filepath.Base(path))
	}
}

func TestExtractMeta(t *testing.T) {
	a := setupTestAdapter(t)

	meta, err := a.ExtractMeta("ses_abc123")
	if err != nil {
		t.Fatalf("ExtractMeta() error = %v", err)
	}

	if meta.ID != "ses_abc123" {
		t.Errorf("ID = %q, want %q", meta.ID, "ses_abc123")
	}
	if meta.Summary != "Refactoring authentication module" {
		t.Errorf("Summary = %q, want %q", meta.Summary, "Refactoring authentication module")
	}
	if meta.Project != "my-app" {
		t.Errorf("Project = %q, want %q", meta.Project, "my-app")
	}
	if meta.ParentSID != "" {
		t.Errorf("ParentSID = %q, want empty for main session", meta.ParentSID)
	}
}

func TestExtractMeta_Subagent(t *testing.T) {
	a := setupTestAdapter(t)

	meta, err := a.ExtractMeta("ses_subagent456")
	if err != nil {
		t.Fatalf("ExtractMeta() error = %v", err)
	}

	if meta.ID != "ses_subagent456" {
		t.Errorf("ID = %q, want %q", meta.ID, "ses_subagent456")
	}
	if meta.Summary != "Background: Explore: Finding auth patterns" {
		t.Errorf("Summary = %q, want %q", meta.Summary, "Background: Explore: Finding auth patterns")
	}
	if meta.ParentSID != "ses_abc123" {
		t.Errorf("ParentSID = %q, want %q", meta.ParentSID, "ses_abc123")
	}
}

func TestGetSessionInfo(t *testing.T) {
	a := setupTestAdapter(t)

	info, err := a.GetSessionInfo("ses_abc123")
	if err != nil {
		t.Fatalf("GetSessionInfo() error = %v", err)
	}

	if info.ID != "ses_abc123" {
		t.Errorf("ID = %q, want %q", info.ID, "ses_abc123")
	}
	if info.WorkDir != "/Users/test/projects/my-app" {
		t.Errorf("WorkDir = %q, want %q", info.WorkDir, "/Users/test/projects/my-app")
	}
	if info.Project != "my-app" {
		t.Errorf("Project = %q, want %q", info.Project, "my-app")
	}
}

func TestGetSummaries(t *testing.T) {
	a := setupTestAdapter(t)

	summaries, err := a.GetSummaries("ses_abc123")
	if err != nil {
		t.Fatalf("GetSummaries() error = %v", err)
	}

	// OpenCode uses session title as summary
	if len(summaries) != 1 {
		t.Fatalf("GetSummaries() returned %d summaries, want 1", len(summaries))
	}
	if summaries[0] != "Refactoring authentication module" {
		t.Errorf("summaries[0] = %q, want %q", summaries[0], "Refactoring authentication module")
	}
}

func TestGetFilesTouched(t *testing.T) {
	a := setupTestAdapter(t)

	files, err := a.GetFilesTouched("ses_abc123")
	if err != nil {
		t.Fatalf("GetFilesTouched() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("GetFilesTouched() returned %d files, want 1", len(files))
	}
	if files[0] != "/Users/test/projects/my-app/src/auth.ts" {
		t.Errorf("files[0] = %q, want %q", files[0], "/Users/test/projects/my-app/src/auth.ts")
	}
}

func TestGetStats(t *testing.T) {
	a := setupTestAdapter(t)

	stats, err := a.GetStats("ses_abc123")
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.UserMessages != 2 {
		t.Errorf("UserMessages = %d, want 2", stats.UserMessages)
	}
	if stats.AssistantMessages != 2 {
		t.Errorf("AssistantMessages = %d, want 2", stats.AssistantMessages)
	}

	// Check token counts (sum of both assistant messages)
	expectedInput := 1500 + 800 // msg_asst1 + msg_asst2
	if stats.InputTokens != expectedInput {
		t.Errorf("InputTokens = %d, want %d", stats.InputTokens, expectedInput)
	}

	expectedOutput := 250 + 150 // msg_asst1 + msg_asst2
	if stats.OutputTokens != expectedOutput {
		t.Errorf("OutputTokens = %d, want %d", stats.OutputTokens, expectedOutput)
	}

	// Check tool calls
	if stats.ToolCalls["edit"] != 1 {
		t.Errorf("ToolCalls[edit] = %d, want 1", stats.ToolCalls["edit"])
	}
}

func TestGetFirstMessage(t *testing.T) {
	a := setupTestAdapter(t)

	msg, err := a.GetFirstMessage("ses_abc123")
	if err != nil {
		t.Fatalf("GetFirstMessage() error = %v", err)
	}

	expected := "Help me refactor the authentication module"
	if msg != expected {
		t.Errorf("GetFirstMessage() = %q, want %q", msg, expected)
	}
}

func TestExportMessages(t *testing.T) {
	a := setupTestAdapter(t)

	messages, err := a.ExportMessages("ses_abc123")
	if err != nil {
		t.Fatalf("ExportMessages() error = %v", err)
	}

	if len(messages) < 4 {
		t.Errorf("ExportMessages() returned %d messages, want at least 4", len(messages))
	}

	// First message should be user
	if messages[0].Role != "user" {
		t.Errorf("messages[0].Role = %q, want %q", messages[0].Role, "user")
	}
	if messages[0].Content != "Help me refactor the authentication module" {
		t.Errorf("messages[0].Content = %q, want %q", messages[0].Content, "Help me refactor the authentication module")
	}

	// Check that we captured tool calls
	foundToolCall := false
	for _, m := range messages {
		if len(m.ToolCalls) > 0 {
			foundToolCall = true
			break
		}
	}
	if !foundToolCall {
		t.Error("ExportMessages() did not capture any tool calls")
	}
}

func TestResumeCmd(t *testing.T) {
	a := New("")
	cmd := a.ResumeCmd("ses_abc123")
	expected := "opencode --session ses_abc123"
	if cmd != expected {
		t.Errorf("ResumeCmd() = %q, want %q", cmd, expected)
	}
}

func TestGetSlashCommands(t *testing.T) {
	a := setupTestAdapter(t)

	cmds, err := a.GetSlashCommands("ses_abc123")
	if err != nil {
		t.Fatalf("GetSlashCommands() error = %v", err)
	}

	if len(cmds) != 1 {
		t.Fatalf("GetSlashCommands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0] != "/commit" {
		t.Errorf("cmds[0] = %q, want %q", cmds[0], "/commit")
	}
}

func TestGetSlashCommands_IgnoresAbsolutePaths(t *testing.T) {
	a := setupTestAdapter(t)

	cmds, err := a.GetSlashCommands("ses_abc123")
	if err != nil {
		t.Fatalf("GetSlashCommands() error = %v", err)
	}

	for _, cmd := range cmds {
		if cmd == "/Users" || cmd == "/Users/test/projects/my-app/src/auth.ts" {
			t.Errorf("GetSlashCommands() should not return absolute paths, got %q", cmd)
		}
	}
}

func TestBranchSession(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := testDataDir(t)
	copyDir(t, srcDir, tmpDir)

	a := New(tmpDir)

	newID, err := a.BranchSession("ses_abc123")
	if err != nil {
		t.Fatalf("BranchSession() error = %v", err)
	}

	if newID == "" || newID == "ses_abc123" {
		t.Errorf("BranchSession() returned invalid ID %q", newID)
	}

	newSessionPath := a.GetSessionFile(newID)
	if newSessionPath == "" {
		t.Fatal("BranchSession() did not create session file")
	}

	data, err := os.ReadFile(newSessionPath)
	if err != nil {
		t.Fatalf("Failed to read new session: %v", err)
	}

	var session map[string]interface{}
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatalf("Failed to parse new session: %v", err)
	}

	if session["parentID"] != "ses_abc123" {
		t.Errorf("New session parentID = %q, want %q", session["parentID"], "ses_abc123")
	}

	msgDir := filepath.Join(tmpDir, "message", newID)
	entries, err := os.ReadDir(msgDir)
	if err != nil {
		t.Fatalf("Failed to read message dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("BranchSession() did not copy any messages")
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0644)
	})
	if err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}
}

// Test that ListSessions populates the path cache
func TestListSessions_PopulatesPathCache(t *testing.T) {
	a := setupTestAdapter(t)

	sessions, err := a.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	// Verify cache is populated
	for _, sid := range sessions {
		path := a.GetSessionFile(sid)
		if path == "" {
			t.Errorf("GetSessionFile(%q) returned empty after ListSessions()", sid)
		}
	}
}

// Test that GetSessionFile uses cache
func TestGetSessionFile_UsesCache(t *testing.T) {
	a := setupTestAdapter(t)

	// Populate cache
	sessions, _ := a.ListSessions()
	if len(sessions) == 0 {
		t.Skip("No test sessions")
	}

	sid := sessions[0]
	path1 := a.GetSessionFile(sid)
	path2 := a.GetSessionFile(sid)

	if path1 != path2 {
		t.Errorf("GetSessionFile returned different paths: %q vs %q", path1, path2)
	}
}

// Test thread safety
func TestPathCache_ThreadSafety(t *testing.T) {
	a := setupTestAdapter(t)

	sessions, _ := a.ListSessions()
	if len(sessions) == 0 {
		t.Skip("No sessions for concurrency test")
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, sid := range sessions {
				_ = a.GetSessionFile(sid)
			}
		}()
	}
	wg.Wait()
}
