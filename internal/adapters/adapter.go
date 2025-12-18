package adapters

import "time"

// Adapter defines the interface for session providers
type Adapter interface {
	// Metadata
	Name() string
	DataDir() string
	CacheDir() string
	ResumeCmd(id string) string

	// Session listing
	ListSessions() ([]string, error)
	GetSessionFile(id string) string

	// Metadata extraction
	ExtractMeta(id string) (*SessionMeta, error)
	GetSessionInfo(id string) (*SessionInfo, error)

	// Content extraction
	GetSummaries(id string) ([]string, error)
	GetFilesTouched(id string) ([]string, error)
	GetSlashCommands(id string) ([]string, error)
	GetStats(id string) (*Stats, error)
	GetFirstMessage(id string) (string, error)

	// Export
	ExportMessages(id string) ([]Message, error)

	// Session operations
	BranchSession(id string) (string, error) // Returns new session ID
}

// SessionMeta contains basic session metadata for cache building
type SessionMeta struct {
	ID      string
	Date    time.Time
	Project string
	Summary string
}

// SessionInfo contains detailed session information for preview
type SessionInfo struct {
	ID      string
	Project string
	Date    time.Time
	Branch  string
	WorkDir string
}

// Stats contains session statistics
type Stats struct {
	UserMessages      int
	AssistantMessages int
	InputTokens       int
	OutputTokens      int
	CacheRead         int
	CacheWrite        int
	Cost              float64
	ToolCalls         map[string]int
}

// Message represents a normalized message for export
type Message struct {
	Role        string       `json:"role"`
	Content     string       `json:"content"`
	Timestamp   int64        `json:"timestamp"`
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

// ToolResult represents a tool result
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	Success   bool   `json:"success"`
}
