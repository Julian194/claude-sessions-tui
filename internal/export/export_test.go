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
			Timestamp: 1705312800,
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

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should start with DOCTYPE")
	}
	if !strings.Contains(html, "<html") {
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
	if !strings.Contains(html, "session-") {
		t.Error("HTML should contain session ID (truncated)")
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
	if !strings.Contains(html, "I'll help you with that") {
		t.Error("HTML should contain assistant message")
	}
}

func TestToHTML_ContainsToolCalls(t *testing.T) {
	messages := sampleMessages()

	html := ToHTML(messages, nil)

	if !strings.Contains(html, "Read") {
		t.Error("HTML should contain tool name")
	}
	if !strings.Contains(html, "/src/auth.ts") {
		t.Error("HTML should contain tool file path")
	}
}

func TestToHTML_NilInfo(t *testing.T) {
	messages := sampleMessages()

	html := ToHTML(messages, nil)

	if html == "" {
		t.Error("ToHTML should return non-empty string even with nil info")
	}
	if !strings.Contains(html, "Session Export") {
		t.Error("HTML should use default title when info is nil")
	}
}

func TestToMarkdown_Basic(t *testing.T) {
	messages := sampleMessages()
	info := sampleInfo()

	md := ToMarkdown(messages, info)

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

func TestToMarkdown_NilInfo(t *testing.T) {
	messages := sampleMessages()

	md := ToMarkdown(messages, nil)

	if md == "" {
		t.Error("ToMarkdown should return non-empty string even with nil info")
	}
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
		{1705312800, "2024-01-15"},
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

func TestToHTML_EmbeddedTemplate(t *testing.T) {
	if htmlTemplate == "" {
		t.Error("HTML template should be embedded")
	}
	if !strings.Contains(htmlTemplate, "{{.Title}}") {
		t.Error("Template should contain Title placeholder")
	}
	if !strings.Contains(htmlTemplate, "{{.MessagesJSON}}") {
		t.Error("Template should contain MessagesJSON placeholder")
	}
}

func TestConvertMessage_SkipsSystemMessages(t *testing.T) {
	msg := adapters.Message{
		Role:    "user",
		Content: "<system>internal message</system>",
	}

	result := convertMessage(msg)
	if result != nil {
		t.Error("convertMessage should skip messages starting with <")
	}
}

func TestConvertMessage_SkipsCaveat(t *testing.T) {
	msg := adapters.Message{
		Role:    "user",
		Content: "Caveat: This is a caveat message",
	}

	result := convertMessage(msg)
	if result != nil {
		t.Error("convertMessage should skip caveat messages")
	}
}

func TestFormatToolCall(t *testing.T) {
	tests := []struct {
		name  string
		tc    adapters.ToolCall
		want  string
	}{
		{
			name: "file_path tool",
			tc:   adapters.ToolCall{Name: "Read", Input: `{"file_path": "/test.txt"}`},
			want: "Read: /test.txt",
		},
		{
			name: "command tool",
			tc:   adapters.ToolCall{Name: "Bash", Input: `{"command": "ls -la"}`},
			want: "Bash: ls -la",
		},
		{
			name: "pattern tool",
			tc:   adapters.ToolCall{Name: "Grep", Input: `{"pattern": "TODO", "path": "/src"}`},
			want: "Grep: TODO in /src",
		},
		{
			name: "no input",
			tc:   adapters.ToolCall{Name: "Unknown", Input: ""},
			want: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolCall(tt.tc)
			if got != tt.want {
				t.Errorf("formatToolCall() = %q, want %q", got, tt.want)
			}
		})
	}
}
