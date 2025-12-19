package opencode

import (
	"os"
	"path/filepath"
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

	if len(sessions) != 1 {
		t.Errorf("ListSessions() returned %d sessions, want 1", len(sessions))
	}

	if sessions[0] != "ses_abc123" {
		t.Errorf("ListSessions()[0] = %q, want %q", sessions[0], "ses_abc123")
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
