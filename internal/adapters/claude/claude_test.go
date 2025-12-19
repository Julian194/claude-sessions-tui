package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func testDataDir(t *testing.T) string {
	t.Helper()
	// Get the testdata directory relative to this test file
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "testdata")
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
	if a.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", a.Name(), "claude")
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

	// Check that both test sessions are found
	found := make(map[string]bool)
	for _, s := range sessions {
		found[s] = true
	}
	if !found["test-session"] {
		t.Error("ListSessions() missing test-session")
	}
	if !found["minimal-session"] {
		t.Error("ListSessions() missing minimal-session")
	}
}

func TestGetSessionFile(t *testing.T) {
	a := setupTestAdapter(t)

	path := a.GetSessionFile("test-session")
	if path == "" {
		t.Fatal("GetSessionFile() returned empty path")
	}
	if filepath.Base(path) != "test-session.jsonl" {
		t.Errorf("GetSessionFile() = %q, want test-session.jsonl", filepath.Base(path))
	}
}

func TestExtractMeta(t *testing.T) {
	a := setupTestAdapter(t)

	meta, err := a.ExtractMeta("test-session")
	if err != nil {
		t.Fatalf("ExtractMeta() error = %v", err)
	}

	if meta.ID != "test-session" {
		t.Errorf("ID = %q, want %q", meta.ID, "test-session")
	}
	if meta.Summary != "Refactoring authentication module" {
		t.Errorf("Summary = %q, want %q", meta.Summary, "Refactoring authentication module")
	}
}

func TestExtractMeta_NoSummary(t *testing.T) {
	a := setupTestAdapter(t)

	meta, err := a.ExtractMeta("minimal-session")
	if err != nil {
		t.Fatalf("ExtractMeta() error = %v", err)
	}

	// Should fall back to first user message
	if meta.Summary != "Hello" {
		t.Errorf("Summary = %q, want %q", meta.Summary, "Hello")
	}
}

func TestGetSessionInfo(t *testing.T) {
	a := setupTestAdapter(t)

	info, err := a.GetSessionInfo("test-session")
	if err != nil {
		t.Fatalf("GetSessionInfo() error = %v", err)
	}

	if info.ID != "test-session" {
		t.Errorf("ID = %q, want %q", info.ID, "test-session")
	}
	if info.Branch != "main" {
		t.Errorf("Branch = %q, want %q", info.Branch, "main")
	}
	if info.WorkDir != "/Users/test/projects/my-app" {
		t.Errorf("WorkDir = %q, want %q", info.WorkDir, "/Users/test/projects/my-app")
	}
}

func TestGetSummaries(t *testing.T) {
	a := setupTestAdapter(t)

	summaries, err := a.GetSummaries("test-session")
	if err != nil {
		t.Fatalf("GetSummaries() error = %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("GetSummaries() returned %d summaries, want 1", len(summaries))
	}
	if summaries[0] != "Refactoring authentication module" {
		t.Errorf("summaries[0] = %q, want %q", summaries[0], "Refactoring authentication module")
	}
}

func TestGetFilesTouched(t *testing.T) {
	a := setupTestAdapter(t)

	files, err := a.GetFilesTouched("test-session")
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

	stats, err := a.GetStats("test-session")
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.UserMessages != 4 {
		t.Errorf("UserMessages = %d, want 4", stats.UserMessages)
	}
	if stats.AssistantMessages != 4 {
		t.Errorf("AssistantMessages = %d, want 4", stats.AssistantMessages)
	}

	// Check token counts
	expectedInput := 1500 + 100 + 800 + 500
	if stats.InputTokens != expectedInput {
		t.Errorf("InputTokens = %d, want %d", stats.InputTokens, expectedInput)
	}

	expectedOutput := 250 + 50 + 150 + 100
	if stats.OutputTokens != expectedOutput {
		t.Errorf("OutputTokens = %d, want %d", stats.OutputTokens, expectedOutput)
	}

	// Check tool calls
	if stats.ToolCalls["Read"] != 1 {
		t.Errorf("ToolCalls[Read] = %d, want 1", stats.ToolCalls["Read"])
	}
	if stats.ToolCalls["Edit"] != 1 {
		t.Errorf("ToolCalls[Edit] = %d, want 1", stats.ToolCalls["Edit"])
	}
}

func TestGetFirstMessage(t *testing.T) {
	a := setupTestAdapter(t)

	msg, err := a.GetFirstMessage("test-session")
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

	messages, err := a.ExportMessages("test-session")
	if err != nil {
		t.Fatalf("ExportMessages() error = %v", err)
	}

	// Should have user and assistant messages (excluding tool results as separate messages)
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
	cmd := a.ResumeCmd("abc123")
	expected := "claude --resume abc123"
	if cmd != expected {
		t.Errorf("ResumeCmd() = %q, want %q", cmd, expected)
	}
}

func TestExtractProject(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/.claude/projects/-Users-julian-code-myapp/session.jsonl", "code/myapp"},
		{"/home/user/.claude/projects/simple-project/session.jsonl", "simple-project"},
	}

	for _, tt := range tests {
		got := extractProject(tt.path)
		if got != tt.want {
			t.Errorf("extractProject(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
