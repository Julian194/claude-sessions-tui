package export

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

// ToHTML converts messages to HTML format
func ToHTML(messages []adapters.Message, info *adapters.SessionInfo) string {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")

	title := "Session Export"
	if info != nil && info.Project != "" {
		title = info.Project
	}
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))

	sb.WriteString("<style>\n")
	sb.WriteString(htmlStyles())
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// Header
	sb.WriteString("<div class=\"header\">\n")
	if info != nil {
		sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", html.EscapeString(info.Project)))
		sb.WriteString(fmt.Sprintf("<p class=\"meta\">Session: %s</p>\n", html.EscapeString(info.ID)))
		if !info.Date.IsZero() {
			sb.WriteString(fmt.Sprintf("<p class=\"meta\">Date: %s</p>\n", info.Date.Format("2006-01-02 15:04")))
		}
		if info.Branch != "" {
			sb.WriteString(fmt.Sprintf("<p class=\"meta\">Branch: %s</p>\n", html.EscapeString(info.Branch)))
		}
	}
	sb.WriteString("</div>\n")

	// Messages
	sb.WriteString("<div class=\"messages\">\n")
	for _, msg := range messages {
		sb.WriteString(messageToHTML(msg))
	}
	sb.WriteString("</div>\n")

	sb.WriteString("</body>\n</html>")
	return sb.String()
}

func messageToHTML(msg adapters.Message) string {
	var sb strings.Builder

	roleClass := "user"
	if msg.Role == "assistant" {
		roleClass = "assistant"
	}

	sb.WriteString(fmt.Sprintf("<div class=\"message %s\">\n", roleClass))
	sb.WriteString(fmt.Sprintf("<div class=\"role\">%s</div>\n", strings.Title(msg.Role)))

	if msg.Content != "" {
		sb.WriteString("<div class=\"content\">\n")
		sb.WriteString(html.EscapeString(msg.Content))
		sb.WriteString("\n</div>\n")
	}

	// Tool calls
	for _, tc := range msg.ToolCalls {
		sb.WriteString("<div class=\"tool-call\">\n")
		sb.WriteString(fmt.Sprintf("<span class=\"tool-name\">%s</span>\n", html.EscapeString(tc.Name)))
		if tc.Input != "" {
			sb.WriteString(fmt.Sprintf("<pre class=\"tool-input\">%s</pre>\n", html.EscapeString(tc.Input)))
		}
		sb.WriteString("</div>\n")
	}

	// Tool results
	for _, tr := range msg.ToolResults {
		sb.WriteString("<div class=\"tool-result\">\n")
		sb.WriteString(fmt.Sprintf("<pre>%s</pre>\n", html.EscapeString(tr.Content)))
		sb.WriteString("</div>\n")
	}

	sb.WriteString("</div>\n")
	return sb.String()
}

func htmlStyles() string {
	return `
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    max-width: 900px;
    margin: 0 auto;
    padding: 20px;
    background: #f5f5f5;
}
.header {
    background: #fff;
    padding: 20px;
    border-radius: 8px;
    margin-bottom: 20px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
.header h1 { margin: 0 0 10px 0; }
.meta { color: #666; margin: 5px 0; }
.messages { display: flex; flex-direction: column; gap: 15px; }
.message {
    padding: 15px;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
.message.user { background: #e3f2fd; }
.message.assistant { background: #fff; }
.role {
    font-weight: bold;
    margin-bottom: 10px;
    color: #333;
}
.content { white-space: pre-wrap; }
.tool-call {
    background: #f0f0f0;
    padding: 10px;
    border-radius: 4px;
    margin-top: 10px;
}
.tool-name {
    font-weight: bold;
    color: #1976d2;
}
.tool-input {
    margin: 5px 0 0 0;
    font-size: 12px;
    overflow-x: auto;
}
.tool-result {
    background: #e8f5e9;
    padding: 10px;
    border-radius: 4px;
    margin-top: 10px;
}
.tool-result pre {
    margin: 0;
    white-space: pre-wrap;
    font-size: 12px;
}
`
}

// ToMarkdown converts messages to Markdown format
func ToMarkdown(messages []adapters.Message, info *adapters.SessionInfo) string {
	var sb strings.Builder

	// Header
	if info != nil {
		sb.WriteString(fmt.Sprintf("# %s\n\n", info.Project))
		sb.WriteString(fmt.Sprintf("**Session:** %s\n", info.ID))
		if !info.Date.IsZero() {
			sb.WriteString(fmt.Sprintf("**Date:** %s\n", info.Date.Format("2006-01-02 15:04")))
		}
		if info.Branch != "" {
			sb.WriteString(fmt.Sprintf("**Branch:** %s\n", info.Branch))
		}
		sb.WriteString("\n---\n\n")
	}

	// Messages
	for _, msg := range messages {
		sb.WriteString(messageToMarkdown(msg))
		sb.WriteString("\n")
	}

	return sb.String()
}

func messageToMarkdown(msg adapters.Message) string {
	var sb strings.Builder

	// Role header
	if msg.Role == "user" {
		sb.WriteString("## 👤 User\n\n")
	} else {
		sb.WriteString("## 🤖 Assistant\n\n")
	}

	if msg.Content != "" {
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	// Tool calls
	for _, tc := range msg.ToolCalls {
		sb.WriteString(fmt.Sprintf("**Tool: %s**\n", tc.Name))
		if tc.Input != "" {
			sb.WriteString("```json\n")
			sb.WriteString(tc.Input)
			sb.WriteString("\n```\n\n")
		}
	}

	// Tool results
	for _, tr := range msg.ToolResults {
		sb.WriteString("**Result:**\n")
		sb.WriteString("```\n")
		sb.WriteString(tr.Content)
		sb.WriteString("\n```\n\n")
	}

	return sb.String()
}

// FormatTimestamp formats a Unix timestamp
func FormatTimestamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}
