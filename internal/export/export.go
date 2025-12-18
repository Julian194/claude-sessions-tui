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
	SessionID    string
	MsgCount     int
	ToolCount    int
	MessagesJSON template.JS
}

// jsMessage represents a message for JavaScript rendering
type jsMessage struct {
	Type  string   `json:"type"`
	Text  string   `json:"text"`
	Tools []string `json:"tools,omitempty"`
	Ts    string   `json:"ts,omitempty"`
}

// ToHTML converts messages to HTML format with full styling
func ToHTML(messages []adapters.Message, info *adapters.SessionInfo) string {
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

	// Convert messages to JS format
	var jsMessages []jsMessage
	for _, msg := range messages {
		jsMsg := convertMessage(msg)
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

func convertMessage(msg adapters.Message) *jsMessage {
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
		var tools []string
		for _, tc := range msg.ToolCalls {
			tools = append(tools, formatToolCall(tc))
		}

		if msg.Content != "" || len(tools) > 0 {
			return &jsMessage{
				Type:  "assistant",
				Text:  msg.Content,
				Tools: tools,
				Ts:    ts,
			}
		}
	}

	return nil
}

func formatToolCall(tc adapters.ToolCall) string {
	name := tc.Name
	detail := ""

	var input map[string]interface{}
	if tc.Input != "" {
		json.Unmarshal([]byte(tc.Input), &input)
	}

	// Extract meaningful display info based on tool type
	if fp, ok := input["file_path"].(string); ok && fp != "" {
		detail = fp
	} else if cmd, ok := input["command"].(string); ok && cmd != "" {
		detail = cmd
	} else if pattern, ok := input["pattern"].(string); ok && pattern != "" {
		if path, ok := input["path"].(string); ok && path != "" {
			detail = fmt.Sprintf("%s in %s", pattern, path)
		} else {
			detail = pattern
		}
	} else if query, ok := input["query"].(string); ok && query != "" {
		detail = query
	} else if url, ok := input["url"].(string); ok && url != "" {
		detail = url
	} else if skill, ok := input["skill"].(string); ok && skill != "" {
		detail = skill
	}

	if detail != "" {
		return fmt.Sprintf("%s: %s", name, detail)
	}
	return name
}

// ToMarkdown converts messages to Markdown format
func ToMarkdown(messages []adapters.Message, info *adapters.SessionInfo) string {
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
