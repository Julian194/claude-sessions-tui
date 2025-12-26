package export

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

//go:embed template.html
var htmlTemplate string

// TemplateData holds data for the HTML template
type TemplateData struct {
	Title        string
	Date         string
	Branch       string
	Models       string
	SessionID    string
	MsgCount     int
	ToolCount    int
	MessagesJSON template.JS
}

// jsMessage represents a message for JavaScript rendering
type jsMessage struct {
	Type     string    `json:"type"`
	Text     string    `json:"text"`
	Thinking string    `json:"thinking,omitempty"`
	Tools    []jsTool  `json:"tools,omitempty"`
	Ts       string    `json:"ts,omitempty"`
}

// jsTool represents a tool call with optional result
type jsTool struct {
	Name   string `json:"name"`
	Detail string `json:"detail,omitempty"`
	Result string `json:"result,omitempty"`
}

// ToHTML converts messages to HTML format with full styling
func ToHTML(messages []adapters.Message, info *adapters.SessionInfo, models []string) string {
	// Prepare template data
	data := TemplateData{
		Title:     "Session Export",
		SessionID: "unknown",
	}

	if info != nil {
		if info.Project != "" {
			data.Title = info.Project
		}
		if !info.Date.IsZero() {
			data.Date = info.Date.Format("2006-01-02 15:04")
		}
		data.Branch = info.Branch
		if len(info.ID) > 8 {
			data.SessionID = info.ID[:8]
		} else {
			data.SessionID = info.ID
		}
	}

	if len(models) > 0 {
		data.Models = strings.Join(models, ", ")
	}

	// Convert messages to JS format
	var jsMessages []jsMessage
	for i, msg := range messages {
		// Get next message for tool result matching
		var nextMsg *adapters.Message
		if i+1 < len(messages) {
			nextMsg = &messages[i+1]
		}
		jsMsg := convertMessage(msg, nextMsg)
		if jsMsg != nil {
			jsMessages = append(jsMessages, *jsMsg)
			if msg.Role == "user" {
				data.MsgCount++
			}
			data.ToolCount += len(msg.ToolCalls)
		}
	}

	// Marshal messages to JSON
	msgJSON, _ := json.Marshal(jsMessages)
	data.MessagesJSON = template.JS(msgJSON)

	// Execute template
	tmpl, err := template.New("export").Parse(htmlTemplate)
	if err != nil {
		return fmt.Sprintf("Template error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Template execution error: %v", err)
	}

	return buf.String()
}

func convertMessage(msg adapters.Message, nextMsg *adapters.Message) *jsMessage {
	ts := ""
	if msg.Timestamp > 0 {
		ts = time.Unix(msg.Timestamp, 0).Format(time.RFC3339)
	}

	if msg.Role == "user" {
		// Skip empty or system messages
		if msg.Content == "" || strings.HasPrefix(msg.Content, "<") || strings.HasPrefix(msg.Content, "Caveat:") {
			return nil
		}
		return &jsMessage{
			Type: "user",
			Text: msg.Content,
			Ts:   ts,
		}
	}

	if msg.Role == "assistant" {
		// Build tool results map from next message (if it's a user message with tool results)
		resultMap := make(map[string]string)
		if nextMsg != nil && nextMsg.Role == "user" {
			for _, tr := range nextMsg.ToolResults {
				resultMap[tr.ToolUseID] = tr.Content
			}
		}

		var tools []jsTool
		for _, tc := range msg.ToolCalls {
			tool := formatToolCall(tc)
			// Match with result if available
			if result, ok := resultMap[tc.ID]; ok {
				tool.Result = truncateResult(result, 500)
			}
			tools = append(tools, tool)
		}

		if msg.Content != "" || msg.Thinking != "" || len(tools) > 0 {
			return &jsMessage{
				Type:     "assistant",
				Text:     msg.Content,
				Thinking: msg.Thinking,
				Tools:    tools,
				Ts:       ts,
			}
		}
	}

	return nil
}

func formatToolCall(tc adapters.ToolCall) jsTool {
	name := tc.Name
	detail := ""

	var input map[string]interface{}
	if tc.Input != "" {
		json.Unmarshal([]byte(tc.Input), &input)
	}

	// Extract meaningful display info based on tool type
	// Support both snake_case (Claude) and camelCase (OpenCode) field names
	if fp := getStringAny(input, "file_path", "filePath"); fp != "" {
		detail = fp
	} else if cmd := getStringAny(input, "command"); cmd != "" {
		detail = cmd
	} else if pattern := getStringAny(input, "pattern"); pattern != "" {
		if path := getStringAny(input, "path"); path != "" {
			detail = fmt.Sprintf("%s in %s", pattern, path)
		} else {
			detail = pattern
		}
	} else if query := getStringAny(input, "query"); query != "" {
		detail = query
	} else if url := getStringAny(input, "url"); url != "" {
		detail = url
	} else if skill := getStringAny(input, "skill"); skill != "" {
		detail = skill
	} else if content := getStringAny(input, "content"); content != "" {
		detail = truncateResult(content, 100)
	}

	return jsTool{
		Name:   name,
		Detail: detail,
	}
}

// getStringAny returns the first non-empty string value from input for any of the given keys
func getStringAny(input map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := input[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func truncateResult(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ToMarkdown converts messages to Markdown format
func ToMarkdown(messages []adapters.Message, info *adapters.SessionInfo, models []string) string {
	var sb strings.Builder

	if info != nil {
		sb.WriteString(fmt.Sprintf("# %s\n\n", info.Project))
		sb.WriteString(fmt.Sprintf("**Session:** %s\n", info.ID))
		if !info.Date.IsZero() {
			sb.WriteString(fmt.Sprintf("**Date:** %s\n", info.Date.Format("2006-01-02 15:04")))
		}
		if info.Branch != "" {
			sb.WriteString(fmt.Sprintf("**Branch:** %s\n", info.Branch))
		}
		if len(models) > 0 {
			sb.WriteString(fmt.Sprintf("**Models:** %s\n", strings.Join(models, ", ")))
		}
		sb.WriteString("\n---\n\n")
	}

	for _, msg := range messages {
		sb.WriteString(messageToMarkdown(msg))
		sb.WriteString("\n")
	}

	return sb.String()
}

func messageToMarkdown(msg adapters.Message) string {
	var sb strings.Builder

	if msg.Role == "user" {
		sb.WriteString("## 👤 User\n\n")
	} else {
		sb.WriteString("## 🤖 Assistant\n\n")
	}

	if msg.Thinking != "" {
		sb.WriteString("<details>\n<summary>💭 Thinking</summary>\n\n")
		sb.WriteString(msg.Thinking)
		sb.WriteString("\n\n</details>\n\n")
	}

	if msg.Content != "" {
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	for _, tc := range msg.ToolCalls {
		sb.WriteString(fmt.Sprintf("**Tool: %s**\n", tc.Name))
		if tc.Input != "" {
			sb.WriteString("```json\n")
			sb.WriteString(tc.Input)
			sb.WriteString("\n```\n\n")
		}
	}

	for _, tr := range msg.ToolResults {
		sb.WriteString("**Result:**\n```\n")
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
