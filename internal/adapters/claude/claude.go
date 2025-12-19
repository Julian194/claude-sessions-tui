package claude

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

// Adapter implements the Claude Code session adapter
type Adapter struct {
	dataDir  string
	cacheDir string
}

// New creates a new Claude adapter
func New(dataDir string) *Adapter {
	if dataDir == "" {
		if envDir := os.Getenv("CLAUDE_DIR"); envDir != "" {
			dataDir = filepath.Join(envDir, "projects")
		} else {
			home, _ := os.UserHomeDir()
			dataDir = filepath.Join(home, ".claude", "projects")
		}
	}
	return &Adapter{
		dataDir:  dataDir,
		cacheDir: filepath.Join(dataDir, "..", ".cache"),
	}
}

func (a *Adapter) Name() string {
	return "claude"
}

func (a *Adapter) DataDir() string {
	return a.dataDir
}

func (a *Adapter) CacheDir() string {
	return a.cacheDir
}

func (a *Adapter) ResumeCmd(id string) string {
	return "claude --resume " + id
}

// ListSessions returns all session IDs sorted by modification time (newest first)
func (a *Adapter) ListSessions() ([]string, error) {
	var sessions []sessionFile

	err := filepath.Walk(a.dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			base := filepath.Base(path)
			// Filter out agent sessions
			if strings.HasPrefix(base, "agent-") {
				return nil
			}
			sessions = append(sessions, sessionFile{
				id:    strings.TrimSuffix(base, ".jsonl"),
				mtime: info.ModTime(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by modification time, newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].mtime.After(sessions[j].mtime)
	})

	ids := make([]string, len(sessions))
	for i, s := range sessions {
		ids[i] = s.id
	}
	return ids, nil
}

type sessionFile struct {
	id    string
	mtime time.Time
}

// GetSessionFile returns the path to a session file
func (a *Adapter) GetSessionFile(id string) string {
	// Search for the session file across all project directories
	var found string
	filepath.Walk(a.dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Base(path) == id+".jsonl" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// ExtractMeta extracts metadata from a session for cache building
func (a *Adapter) ExtractMeta(id string) (*adapters.SessionMeta, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Extract project from path
	project := extractProject(path)

	// Parse file for summary and parent session
	summary := ""
	parentSID := ""
	records, err := a.parseFile(path)
	if err == nil {
		for _, r := range records {
			if r.Type == "summary" && summary == "" {
				summary = r.Summary
			}
			if r.Type == "branch" && r.ParentSession != "" {
				parentSID = r.ParentSession
			}
		}
		// Fallback to first user message if no summary
		if summary == "" {
			for _, r := range records {
				if r.Type == "user" && r.Message.Role == "user" {
					if content, ok := r.Message.Content.(string); ok {
						summary = truncate(content, 100)
						break
					}
				}
			}
		}
	}

	return &adapters.SessionMeta{
		ID:        id,
		Date:      info.ModTime(),
		Project:   project,
		Summary:   summary,
		ParentSID: parentSID,
	}, nil
}

// GetSessionInfo returns detailed session information
func (a *Adapter) GetSessionInfo(id string) (*adapters.SessionInfo, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	project := extractProject(path)
	branch := ""
	workDir := ""

	records, err := a.parseFile(path)
	if err == nil {
		for _, r := range records {
			if r.GitBranch != "" && branch == "" {
				branch = r.GitBranch
			}
			if r.Cwd != "" && workDir == "" {
				workDir = r.Cwd
			}
			if branch != "" && workDir != "" {
				break
			}
		}
	}

	return &adapters.SessionInfo{
		ID:      id,
		Project: project,
		Date:    info.ModTime(),
		Branch:  branch,
		WorkDir: workDir,
	}, nil
}

// GetSummaries returns all topic summaries from the session
func (a *Adapter) GetSummaries(id string) ([]string, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	records, err := a.parseFile(path)
	if err != nil {
		return nil, err
	}

	var summaries []string
	for _, r := range records {
		if r.Type == "summary" && r.Summary != "" {
			summaries = append(summaries, r.Summary)
		}
	}
	return summaries, nil
}

// GetFilesTouched returns files modified in the session
func (a *Adapter) GetFilesTouched(id string) ([]string, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	records, err := a.parseFile(path)
	if err != nil {
		return nil, err
	}

	fileSet := make(map[string]bool)
	for _, r := range records {
		if r.Type == "assistant" {
			// Look for tool calls that modify files
			if content, ok := r.Message.Content.([]interface{}); ok {
				for _, item := range content {
					if m, ok := item.(map[string]interface{}); ok {
						if m["type"] == "tool_use" {
							name, _ := m["name"].(string)
							if name == "Edit" || name == "Write" {
								if input, ok := m["input"].(map[string]interface{}); ok {
									if fp, ok := input["file_path"].(string); ok {
										fileSet[fp] = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	files := make([]string, 0, len(fileSet))
	for f := range fileSet {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// GetSlashCommands returns slash commands used in the session
func (a *Adapter) GetSlashCommands(id string) ([]string, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	records, err := a.parseFile(path)
	if err != nil {
		return nil, err
	}

	cmdSet := make(map[string]bool)
	cmdRegex := regexp.MustCompile(`<command-name>(/[^<]*)</command-name>`)

	for _, r := range records {
		// Check for command-name tags in user messages
		if r.Type == "user" {
			if content, ok := r.Message.Content.(string); ok {
				matches := cmdRegex.FindAllStringSubmatch(content, -1)
				for _, m := range matches {
					if len(m) > 1 {
						cmdSet[m[1]] = true
					}
				}
			}
		}
		// Check for SlashCommand tool calls
		if r.Type == "assistant" {
			if content, ok := r.Message.Content.([]interface{}); ok {
				for _, item := range content {
					if m, ok := item.(map[string]interface{}); ok {
						if m["type"] == "tool_use" && m["name"] == "SlashCommand" {
							if input, ok := m["input"].(map[string]interface{}); ok {
								if cmd, ok := input["command"].(string); ok {
									cmdSet[cmd] = true
								}
							}
						}
					}
				}
			}
		}
	}

	cmds := make([]string, 0, len(cmdSet))
	for cmd := range cmdSet {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	return cmds, nil
}

// GetStats returns session statistics
func (a *Adapter) GetStats(id string) (*adapters.Stats, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	records, err := a.parseFile(path)
	if err != nil {
		return nil, err
	}

	stats := &adapters.Stats{
		ToolCalls: make(map[string]int),
	}

	for _, r := range records {
		switch r.Type {
		case "user":
			if !r.IsMeta {
				stats.UserMessages++
			}
		case "assistant":
			stats.AssistantMessages++
			if r.Message.Usage != nil {
				stats.InputTokens += r.Message.Usage.InputTokens
				stats.OutputTokens += r.Message.Usage.OutputTokens
				stats.CacheRead += r.Message.Usage.CacheReadInputTokens
				stats.CacheWrite += r.Message.Usage.CacheCreationInputTokens
			}
			// Count tool calls
			if content, ok := r.Message.Content.([]interface{}); ok {
				for _, item := range content {
					if m, ok := item.(map[string]interface{}); ok {
						if m["type"] == "tool_use" {
							name, _ := m["name"].(string)
							if name != "" {
								stats.ToolCalls[name]++
							}
						}
					}
				}
			}
		}
	}

	// Calculate cost (approximate)
	stats.Cost = calculateCost(stats.InputTokens, stats.OutputTokens, stats.CacheRead, stats.CacheWrite)

	return stats, nil
}

// GetFirstMessage returns the first user message
func (a *Adapter) GetFirstMessage(id string) (string, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return "", os.ErrNotExist
	}

	records, err := a.parseFile(path)
	if err != nil {
		return "", err
	}

	for _, r := range records {
		if r.Type == "user" && !r.IsMeta && r.Message.Role == "user" {
			if content, ok := r.Message.Content.(string); ok {
				return truncate(content, 200), nil
			}
		}
	}
	return "", nil
}

// ExportMessages returns all messages in normalized format
func (a *Adapter) ExportMessages(id string) ([]adapters.Message, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	records, err := a.parseFile(path)
	if err != nil {
		return nil, err
	}

	var messages []adapters.Message
	for _, r := range records {
		if r.Type == "user" || r.Type == "assistant" {
			if r.IsMeta {
				continue
			}
			msg := adapters.Message{
				Role:      r.Message.Role,
				Timestamp: parseTimestamp(r.Timestamp),
			}

			// Extract content
			switch c := r.Message.Content.(type) {
			case string:
				msg.Content = c
			case []interface{}:
				for _, item := range c {
					if m, ok := item.(map[string]interface{}); ok {
						switch m["type"] {
						case "text":
							if text, ok := m["text"].(string); ok {
								msg.Content += text
							}
						case "tool_use":
							tc := adapters.ToolCall{
								ID:   getString(m, "id"),
								Name: getString(m, "name"),
							}
							if input, ok := m["input"]; ok {
								if b, err := json.Marshal(input); err == nil {
									tc.Input = string(b)
								}
							}
							msg.ToolCalls = append(msg.ToolCalls, tc)
						case "tool_result":
							tr := adapters.ToolResult{
								ToolUseID: getString(m, "tool_use_id"),
								Content:   getString(m, "content"),
							}
							msg.ToolResults = append(msg.ToolResults, tr)
						}
					}
				}
			}

			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// Internal types for parsing

type record struct {
	Type          string  `json:"type"`
	Summary       string  `json:"summary,omitempty"`
	Message       message `json:"message,omitempty"`
	Cwd           string  `json:"cwd,omitempty"`
	GitBranch     string  `json:"gitBranch,omitempty"`
	Timestamp     string  `json:"timestamp,omitempty"`
	IsMeta        bool    `json:"isMeta,omitempty"`
	ParentSession string  `json:"parentSession,omitempty"` // For branch metadata
}

type message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
	Usage   *usage      `json:"usage,omitempty"`
}

type usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

func (a *Adapter) parseFile(path string) ([]record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []record
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		var r record
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue // Skip malformed lines
		}
		records = append(records, r)
	}

	return records, scanner.Err()
}

// Helper functions

func extractProject(path string) string {
	// Path format: .../projects/{project-name}/{session}.jsonl
	dir := filepath.Dir(path)
	project := filepath.Base(dir)

	// Claude encodes full paths as project names: -Users-julian-code-foo
	// Convert back to readable format: code/foo
	re := regexp.MustCompile(`^-Users-[^-]+-`)
	if re.MatchString(project) {
		// This is a path-encoded project name, convert dashes to slashes
		project = re.ReplaceAllString(project, "")
		project = strings.ReplaceAll(project, "-", "/")
	}
	// Otherwise keep original name (e.g., "my-project" stays as-is)

	return project
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func parseTimestamp(ts string) int64 {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func calculateCost(input, output, cacheRead, cacheWrite int) float64 {
	// Sonnet 3.5 pricing (per 1M tokens)
	inputPrice := 3.0
	outputPrice := 15.0
	cacheReadPrice := 0.30
	cacheWritePrice := 3.75

	cost := float64(input) * inputPrice / 1_000_000
	cost += float64(output) * outputPrice / 1_000_000
	cost += float64(cacheRead) * cacheReadPrice / 1_000_000
	cost += float64(cacheWrite) * cacheWritePrice / 1_000_000

	return cost
}

// BranchSession creates a copy of a session for branching
func (a *Adapter) BranchSession(id string) (string, error) {
	originalPath := a.GetSessionFile(id)
	if originalPath == "" {
		return "", os.ErrNotExist
	}

	// Generate new UUID
	newID := generateUUID()

	// New file in same directory
	dir := filepath.Dir(originalPath)
	newPath := filepath.Join(dir, newID+".jsonl")

	// Read original content
	content, err := os.ReadFile(originalPath)
	if err != nil {
		return "", err
	}

	// Create branch metadata
	branchMeta := map[string]interface{}{
		"type":          "branch",
		"parentSession": id,
		"branchedAt":    time.Now().UTC().Format(time.RFC3339),
	}
	metaJSON, _ := json.Marshal(branchMeta)

	// Write new file with branch metadata prepended
	newContent := append(metaJSON, '\n')
	newContent = append(newContent, content...)

	if err := os.WriteFile(newPath, newContent, 0644); err != nil {
		return "", err
	}

	// Touch the file to ensure proper mtime
	now := time.Now()
	os.Chtimes(newPath, now, now)

	return newID, nil
}

// generateUUID creates a random UUID v4
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
