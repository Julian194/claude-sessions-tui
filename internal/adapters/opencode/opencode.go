package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

type Adapter struct {
	dataDir  string
	cacheDir string
}

func New(dataDir string) *Adapter {
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share", "opencode", "storage")
	}
	return &Adapter{
		dataDir:  dataDir,
		cacheDir: filepath.Join(dataDir, "..", ".cache"),
	}
}

func (a *Adapter) Name() string {
	return "opencode"
}

func (a *Adapter) DataDir() string {
	return a.dataDir
}

func (a *Adapter) CacheDir() string {
	return a.cacheDir
}

func (a *Adapter) ResumeCmd(id string) string {
	return "opencode --session " + id
}

func (a *Adapter) ListSessions() ([]string, error) {
	var sessions []sessionFile

	sessionDir := filepath.Join(a.dataDir, "session")
	err := filepath.Walk(sessionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".json") && strings.HasPrefix(filepath.Base(path), "ses_") {
			sessions = append(sessions, sessionFile{
				id:    strings.TrimSuffix(filepath.Base(path), ".json"),
				mtime: info.ModTime(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

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

func (a *Adapter) GetSessionFile(id string) string {
	var found string
	sessionDir := filepath.Join(a.dataDir, "session")
	filepath.Walk(sessionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Base(path) == id+".json" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func (a *Adapter) ExtractMeta(id string) (*adapters.SessionMeta, error) {
	session, err := a.loadSession(id)
	if err != nil {
		return nil, err
	}

	project := extractProject(session.Directory)

	return &adapters.SessionMeta{
		ID:      id,
		Date:    time.UnixMilli(session.Time.Updated),
		Project: project,
		Summary: session.Title,
	}, nil
}

func (a *Adapter) GetSessionInfo(id string) (*adapters.SessionInfo, error) {
	session, err := a.loadSession(id)
	if err != nil {
		return nil, err
	}

	project := extractProject(session.Directory)

	return &adapters.SessionInfo{
		ID:      id,
		Project: project,
		Date:    time.UnixMilli(session.Time.Updated),
		WorkDir: session.Directory,
	}, nil
}

func (a *Adapter) GetSummaries(id string) ([]string, error) {
	session, err := a.loadSession(id)
	if err != nil {
		return nil, err
	}

	if session.Title != "" {
		return []string{session.Title}, nil
	}
	return nil, nil
}

func (a *Adapter) GetFilesTouched(id string) ([]string, error) {
	parts, err := a.loadParts(id)
	if err != nil {
		return nil, err
	}

	fileSet := make(map[string]bool)
	for _, part := range parts {
		if part.Type == "tool" && (part.Tool == "edit" || part.Tool == "write") {
			if part.State.Input.FilePath != "" {
				fileSet[part.State.Input.FilePath] = true
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

func (a *Adapter) GetSlashCommands(id string) ([]string, error) {
	return nil, nil
}

func (a *Adapter) GetStats(id string) (*adapters.Stats, error) {
	messages, err := a.loadMessages(id)
	if err != nil {
		return nil, err
	}

	parts, err := a.loadParts(id)
	if err != nil {
		return nil, err
	}

	stats := &adapters.Stats{
		ToolCalls: make(map[string]int),
	}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			stats.UserMessages++
		case "assistant":
			stats.AssistantMessages++
			stats.InputTokens += msg.Tokens.Input
			stats.OutputTokens += msg.Tokens.Output
			stats.CacheRead += msg.Tokens.Cache.Read
			stats.CacheWrite += msg.Tokens.Cache.Write
			stats.Cost += msg.Cost
		}
	}

	for _, part := range parts {
		if part.Type == "tool" && part.Tool != "" {
			stats.ToolCalls[part.Tool]++
		}
	}

	return stats, nil
}

func (a *Adapter) GetFirstMessage(id string) (string, error) {
	parts, err := a.loadParts(id)
	if err != nil {
		return "", err
	}

	messages, err := a.loadMessages(id)
	if err != nil {
		return "", err
	}

	msgOrder := make(map[string]int64)
	for _, msg := range messages {
		if msg.Role == "user" {
			msgOrder[msg.ID] = msg.Time.Created
		}
	}

	var firstMsgID string
	var firstTime int64 = 1<<62 - 1
	for msgID, t := range msgOrder {
		if t < firstTime {
			firstTime = t
			firstMsgID = msgID
		}
	}

	for _, part := range parts {
		if part.MessageID == firstMsgID && part.Type == "text" {
			return truncate(part.Text, 200), nil
		}
	}

	return "", nil
}

func (a *Adapter) ExportMessages(id string) ([]adapters.Message, error) {
	messages, err := a.loadMessages(id)
	if err != nil {
		return nil, err
	}

	parts, err := a.loadParts(id)
	if err != nil {
		return nil, err
	}

	partsByMsg := make(map[string][]part)
	for _, p := range parts {
		partsByMsg[p.MessageID] = append(partsByMsg[p.MessageID], p)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Time.Created < messages[j].Time.Created
	})

	var result []adapters.Message
	for _, msg := range messages {
		m := adapters.Message{
			Role:      msg.Role,
			Timestamp: msg.Time.Created / 1000,
		}

		msgParts := partsByMsg[msg.ID]
		for _, p := range msgParts {
			if p.Type == "text" {
				m.Content += p.Text
			} else if p.Type == "tool" {
				tc := adapters.ToolCall{
					ID:   p.CallID,
					Name: p.Tool,
				}
				if input, err := json.Marshal(p.State.Input); err == nil {
					tc.Input = string(input)
				}
				m.ToolCalls = append(m.ToolCalls, tc)
			}
		}

		result = append(result, m)
	}

	return result, nil
}

func (a *Adapter) BranchSession(id string) (string, error) {
	return "", nil
}

type sessionData struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectID"`
	Directory string `json:"directory"`
	Title     string `json:"title"`
	Time      struct {
		Created int64 `json:"created"`
		Updated int64 `json:"updated"`
	} `json:"time"`
}

type messageData struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Time      struct {
		Created   int64 `json:"created"`
		Completed int64 `json:"completed"`
	} `json:"time"`
	ModelID string  `json:"modelID"`
	Cost    float64 `json:"cost"`
	Tokens  struct {
		Input     int `json:"input"`
		Output    int `json:"output"`
		Reasoning int `json:"reasoning"`
		Cache     struct {
			Read  int `json:"read"`
			Write int `json:"write"`
		} `json:"cache"`
	} `json:"tokens"`
}

type part struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	CallID    string `json:"callID,omitempty"`
	Tool      string `json:"tool,omitempty"`
	State     struct {
		Status string `json:"status"`
		Input  struct {
			FilePath  string `json:"filePath,omitempty"`
			OldString string `json:"oldString,omitempty"`
			NewString string `json:"newString,omitempty"`
		} `json:"input"`
		Output string `json:"output"`
	} `json:"state,omitempty"`
}

func (a *Adapter) loadSession(id string) (*sessionData, error) {
	path := a.GetSessionFile(id)
	if path == "" {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session sessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (a *Adapter) loadMessages(sessionID string) ([]messageData, error) {
	msgDir := filepath.Join(a.dataDir, "message", sessionID)
	entries, err := os.ReadDir(msgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var messages []messageData
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(msgDir, entry.Name()))
		if err != nil {
			continue
		}

		var msg messageData
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (a *Adapter) loadParts(sessionID string) ([]part, error) {
	messages, err := a.loadMessages(sessionID)
	if err != nil {
		return nil, err
	}

	var parts []part
	for _, msg := range messages {
		partDir := filepath.Join(a.dataDir, "part", msg.ID)
		entries, err := os.ReadDir(partDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(partDir, entry.Name()))
			if err != nil {
				continue
			}

			var p part
			if err := json.Unmarshal(data, &p); err != nil {
				continue
			}

			parts = append(parts, p)
		}
	}

	return parts, nil
}

func extractProject(dir string) string {
	if dir == "" {
		return ""
	}
	return filepath.Base(dir)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
