package export

import (
	"strings"
	"testing"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

func sampleMessages() []adapters.Message {
	return []adapters.Message{
		{
			Role:      "user",
			Content:   "Help me with authentication",
			Timestamp: 1705312800, // 2024-01-15 10:00:00
		},
		{
			Role:      "assistant",
			Content:   "I'll help you with that.",
			Timestamp: 1705312805,
			ToolCalls: []adapters.ToolCall{
				{
					ID:    "tool-1",
					Name:  "Read",
					Input: `{"file_path": "/src/auth.ts"}`,
				},
			},
		},
		{
			Role:      "user",
			Content:   "",
			Timestamp: 1705312810,
			ToolResults: []adapters.ToolResult{
				{
					ToolUseID: "tool-1",
					Content:   "export function login() {}",
					Success:   true,
				},
			},
		},
	}
}

func sampleInfo() *adapters.SessionInfo {
	return &adapters.SessionInfo{
		ID:      "session-123",
		Project: "my-project",
		Date:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Branch:  "main",
		WorkDir: "/home/user/project",
	}
}

func TestToHTML_Basic(t *testing.T) {
	messages := sampleMessages()
	info := sampleInfo()

	html := ToHTML(messages, info)

	// Check structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should start with DOCTYPE")
	}
	if !strings.Contains(html, "<html>") {
		t.Error("HTML should contain <html> tag")
	}
	if !strings.Contains(html, "</html>") {
		t.Error("HTML should end with </html>")
	}
}

func TestToHTML_ContainsSessionInfo(t *testing.T) {
	messages := sampleMessages()
	info := sampleInfo()

	html := ToHTML(messages, info)

	if !strings.Contains(html, "my-project") {
		t.Error("HTML should contain project name")
	}
	if !strings.Contains(html, "session-123") {
		t.Error("HTML should contain session ID")
	}
	if !strings.Contains(html, "main") {
		t.Error("HTML should contain branch name")
	}
}

func TestToHTML_ContainsMessages(t *testing.T) {
	messages := sampleMessages()
	info := sampleInfo()

	html := ToHTML(messages, info)

	if !strings.Contains(html, "Help me with authentication") {
		t.Error("HTML should contain user message")
	}
	if !strings.Contains(html, "I&#39;ll help you with that.") {
		t.Error("HTML should contain assistant message (escaped)")
	}
}

func TestToHTML_ContainsToolCalls(t *testing.T) {
	messages := sampleMessages()

	html := ToHTML(messages, nil)

	if !strings.Contains(html, "Read") {
		t.Error("HTML should contain tool name")
	}
	if !strings.Contains(html, "file_path") {
		t.Error("HTML should contain tool input")
	}
}

func TestToHTML_ContainsToolResults(t *testing.T) {
	messages := sampleMessages()

	html := ToHTML(messages, nil)

	if !strings.Contains(html, "export function login()") {
		t.Error("HTML should contain tool result")
	}
}

func TestToHTML_EscapesSpecialChars(t *testing.T) {
	messages := []adapters.Message{
		{
			Role:    "user",
			Content: "Test <script>alert('xss')</script>",
		},
	}

	html := ToHTML(messages, nil)

	if strings.Contains(html, "<script>") {
		t.Error("HTML should escape script tags")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("HTML should contain escaped script tag")
	}
}

func TestToHTML_NilInfo(t *testing.T) {
	messages := sampleMessages()

	// Should not panic with nil info
	html := ToHTML(messages, nil)

	if html == "" {
		t.Error("ToHTML should return non-empty string even with nil info")
	}
}

func TestToMarkdown_Basic(t *testing.T) {
	messages := sampleMessages()
	info := sampleInfo()

	md := ToMarkdown(messages, info)

	// Check header
	if !strings.Contains(md, "# my-project") {
		t.Error("Markdown should contain project header")
	}
	if !strings.Contains(md, "**Session:** session-123") {
		t.Error("Markdown should contain session ID")
	}
}

func TestToMarkdown_ContainsMessages(t *testing.T) {
	messages := sampleMessages()

	md := ToMarkdown(messages, nil)

	if !strings.Contains(md, "## 👤 User") {
		t.Error("Markdown should contain user header")
	}
	if !strings.Contains(md, "## 🤖 Assistant") {
		t.Error("Markdown should contain assistant header")
	}
	if !strings.Contains(md, "Help me with authentication") {
		t.Error("Markdown should contain user message")
	}
}

func TestToMarkdown_ContainsToolCalls(t *testing.T) {
	messages := sampleMessages()

	md := ToMarkdown(messages, nil)

	if !strings.Contains(md, "**Tool: Read**") {
		t.Error("Markdown should contain tool name")
	}
	if !strings.Contains(md, "```json") {
		t.Error("Markdown should contain JSON code block for tool input")
	}
}

func TestToMarkdown_ContainsToolResults(t *testing.T) {
	messages := sampleMessages()

	md := ToMarkdown(messages, nil)

	if !strings.Contains(md, "**Result:**") {
		t.Error("Markdown should contain result header")
	}
	if !strings.Contains(md, "export function login()") {
		t.Error("Markdown should contain tool result")
	}
}

func TestToMarkdown_NilInfo(t *testing.T) {
	messages := sampleMessages()

	// Should not panic with nil info
	md := ToMarkdown(messages, nil)

	if md == "" {
		t.Error("ToMarkdown should return non-empty string even with nil info")
	}
	// Should not have header when no info
	if strings.Contains(md, "**Session:**") {
		t.Error("Markdown should not contain session info when info is nil")
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		ts   int64
		want string
	}{
		{0, ""},
		{1705312800, "2024-01-15"}, // Just check date part
	}

	for _, tt := range tests {
		got := FormatTimestamp(tt.ts)
		if tt.ts == 0 && got != "" {
			t.Errorf("FormatTimestamp(0) = %q, want empty", got)
		}
		if tt.ts != 0 && !strings.Contains(got, tt.want) {
			t.Errorf("FormatTimestamp(%d) = %q, should contain %q", tt.ts, got, tt.want)
		}
	}
}

func TestHTMLStyles(t *testing.T) {
	styles := htmlStyles()

	if !strings.Contains(styles, "body") {
		t.Error("Styles should contain body selector")
	}
	if !strings.Contains(styles, ".message") {
		t.Error("Styles should contain .message selector")
	}
	if !strings.Contains(styles, ".tool-call") {
		t.Error("Styles should contain .tool-call selector")
	}
}
