# Refactor Plan: Multi-Agent Session Browser

## Goal

Transform `claude-sessions-tui` into a generic session browser that supports multiple AI coding agents (Claude Code, OpenCode, Aider, Cursor, etc.) via a provider/adapter architecture.

---

## Current Architecture

```
bin/
  claude-sessions           # Main TUI (fzf launcher)
  claude-sessions-rebuild   # Build TSV cache from JSONL
  claude-sessions-preview   # Preview pane content
  claude-sessions-stats     # Python script for token/cost stats
  claude-sessions-export    # HTML export
  claude-sessions-copy-md   # Markdown export to clipboard
  claude-sessions-reset-header  # fzf header reset helper
```

**Data flow:**
1. `rebuild` scans `~/.claude/projects/**/*.jsonl` → writes `sessions-cache.tsv`
2. `sessions` reads cache, launches fzf with preview
3. `preview` parses JSONL for display
4. `export`/`copy-md` parse JSONL for output

---

## Target Architecture

**Language: Go**

Single compiled binary, zero runtime dependencies, fast startup. Perfect for CLI tools distributed via Homebrew.

```
bin/
  sessions                  # Compiled Go binary
  claude-sessions    → sessions   (symlink)
  opencode-sessions  → sessions   (symlink)
  # Future: aider-sessions, cursor-sessions, etc.

cmd/
  sessions/
    main.go                 # Entry point, subcommand routing

internal/
  tui/
    tui.go                  # fzf integration, TUI logic
  cache/
    cache.go                # TSV cache management
  export/
    html.go                 # HTML export
    markdown.go             # Markdown export
  stats/
    stats.go                # Token/cost calculation
  adapters/
    adapter.go              # Interface definition
    claude/
      claude.go             # Claude JSONL parsing
    opencode/
      opencode.go           # OpenCode JSON parsing (Phase 2)

scripts/
  release                   # Release automation
  test-pr                   # PR testing
  upgrade                   # Upgrade helper
```

**Why Go:**
- Single binary - simple Homebrew distribution
- Fast startup (~5ms vs ~50ms for Python)
- Built-in testing (`go test`)
- Excellent JSON handling
- No runtime dependencies for users
- Strong typing catches refactor bugs

---

## Adapter Interface Specification

Each adapter implements the `Adapter` interface:

```go
type Adapter interface {
    // Metadata
    Name() string           // "claude", "opencode"
    DataDir() string        // ~/.claude/projects, ~/.local/share/opencode/storage
    CacheDir() string       // Cache location
    ResumeCmd(id string) string  // Command to resume session

    // Session listing
    ListSessions() ([]string, error)  // Session IDs, newest first
    GetSessionFile(id string) string  // Path to session file

    // Metadata extraction
    ExtractMeta(id string) (*SessionMeta, error)  // For cache building
    GetSessionInfo(id string) (*SessionInfo, error)  // For preview header

    // Content extraction
    GetSummaries(id string) ([]string, error)     // Topic summaries
    GetFilesTouched(id string) ([]string, error)  // Modified files
    GetStats(id string) (*Stats, error)           // Token counts, costs
    GetFirstMessage(id string) (string, error)    // Fallback when no summaries

    // Export
    ExportMessages(id string) ([]Message, error)  // Normalized message format
}

type SessionMeta struct {
    ID      string
    Date    time.Time
    Project string
    Summary string
}

type SessionInfo struct {
    ID        string
    Project   string
    Date      time.Time
    Branch    string
    WorkDir   string
}

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

type Message struct {
    Role        string      `json:"role"`
    Content     string      `json:"content"`
    Timestamp   int64       `json:"timestamp"`
    ToolCalls   []ToolCall  `json:"tool_calls,omitempty"`
    ToolResults []ToolResult `json:"tool_results,omitempty"`
}
```

---

## Subcommand Structure

```bash
claude-sessions              # Default: launch TUI
claude-sessions rebuild      # Rebuild cache
claude-sessions preview <id> # Show preview (used by fzf)
claude-sessions stats <id>   # Show detailed stats
claude-sessions export <id>  # Export to HTML
claude-sessions copy-md <id> # Copy as markdown
```

Adapter detection via binary name (symlink → same binary):

```go
func main() {
    // Detect adapter from how we were invoked
    name := filepath.Base(os.Args[0])
    adapter := getAdapter(name)  // claude-sessions → claude adapter

    // Route subcommand
    cmd := "tui"
    if len(os.Args) > 1 {
        cmd = os.Args[1]
    }

    switch cmd {
    case "tui", "":
        runTUI(adapter)
    case "rebuild":
        runRebuild(adapter)
    case "preview":
        runPreview(adapter, os.Args[2])
    // ...
    }
}
```

---

## Testing Strategy

**Goal:** Write tests first (in Go), verify current bash scripts work, then migrate to Go using tests as safety net.

### Test Structure

```
internal/
  adapters/
    claude/
      claude_test.go        # Unit tests for Claude adapter
      testdata/             # Sample JSONL fixtures
  cache/
    cache_test.go           # Cache format tests
  export/
    export_test.go          # HTML/MD output tests

test/
  integration_test.go       # End-to-end tests (run against binary)
  fixtures/
    claude/                 # Sample Claude JSONL sessions
```

### Test Approach

1. **Fixture-based:** Sample session files with known content
2. **Golden tests:** Compare output against expected results
3. **Integration tests:** Run compiled binary, verify output
4. **`go test`:** Built-in, fast, parallel

```go
func TestPreviewShowsProjectName(t *testing.T) {
    adapter := claude.New(fixtureDir)
    info, err := adapter.GetSessionInfo(testSessionID)
    require.NoError(t, err)
    assert.Equal(t, "test-project", info.Project)
}

func TestExportHTML(t *testing.T) {
    // Golden test - compare against known good output
    got := export.ToHTML(messages)
    golden := readGoldenFile(t, "expected_export.html")
    assert.Equal(t, golden, got)
}
```

### What to Test

| Component | Test cases |
|-----------|------------|
| `claude.ListSessions` | Finds sessions, sorted by date |
| `claude.ExtractMeta` | Parses project name, date, summary |
| `claude.GetStats` | Correct token counts, cost calculation |
| `cache.Write/Read` | TSV format, handles special chars |
| `export.ToHTML` | Valid HTML, correct message order |
| `export.ToMarkdown` | Valid MD, tool calls formatted |
| Integration | Full flow: rebuild → preview → export |

---

## Implementation Tasks

### Phase 1: Go Project Setup & Tests

- [x] **1.1** Initialize Go module (`go mod init`)
- [x] **1.2** Create directory structure (`cmd/`, `internal/`)
- [x] **1.3** Create test fixtures from real Claude sessions (anonymized)
- [x] **1.4** Write Claude adapter tests (parsing, metadata extraction)
- [x] **1.5** Write cache tests (TSV format)
- [x] **1.6** Write export tests (HTML, Markdown) with golden files
- [x] **1.7** Write stats tests (token counting, cost calculation)
- [x] **1.8** All tests pass against fixtures

### Phase 2: Migrate Claude to Go

- [x] **2.1** Implement Claude adapter (`internal/adapters/claude/`)
- [x] **2.2** Implement cache management (`internal/cache/`)
- [x] **2.3** Implement stats calculation (`internal/stats/`)
- [x] **2.4** Implement HTML export (`internal/export/html.go`)
- [x] **2.5** Implement Markdown export (`internal/export/markdown.go`)
- [x] **2.6** Implement TUI/fzf integration (`internal/tui/`)
- [x] **2.7** Implement main entry point with subcommand routing
- [x] **2.8** All tests pass against Go implementation
- [x] **2.9** Manual testing: `go run ./cmd/sessions` works like old scripts
- [ ] **2.10** Update Homebrew formula for Go binary (invalidate cache on upgrade)

### Phase 3: OpenCode Adapter

- [ ] **3.1** Create OpenCode test fixtures
- [ ] **3.2** Write OpenCode adapter tests
- [ ] **3.3** Implement OpenCode adapter (`internal/adapters/opencode/`)
- [ ] **3.4** Create `opencode-sessions` symlink
- [ ] **3.5** Test full workflow: browse, preview, export, resume

### Phase 4: Polish & Release

- [ ] **4.1** Update README for multi-agent support
- [ ] **4.2** Add `--help` with provider-specific info
- [ ] **4.3** Consider renaming repo (e.g., `ai-sessions-tui`)
- [ ] **4.4** Update Homebrew formula with all symlinks
- [ ] **4.5** Tag release

### Phase 5: Future Adapters (optional)

- [ ] **5.1** Research Aider session storage format
- [ ] **5.2** Research Cursor session storage format
- [ ] **5.3** Implement additional adapters as needed

---

## Data Format Reference

### Claude Code

**Location:** `~/.claude/projects/{project-name}/{session-uuid}.jsonl`

**Format:** JSONL (one JSON object per line)

**Key fields:**
```json
{"type": "summary", "summary": "..."}
{"role": "user", "content": "..."}
{"role": "assistant", "content": [...], "message": {...}}
{"cwd": "/path/to/project", "gitBranch": "main"}
```

### OpenCode

**Location:** `~/.local/share/opencode/storage/`

**Format:** Hierarchical JSON files

**Structure:**
```
storage/
├── project/{projectID}.json
│   { "id": "...", "worktree": "/path", "vcs": "git" }
│
├── session/{projectID}/{sessionID}.json
│   { "id": "ses_...", "title": "...", "time": {...}, "summary": {...} }
│
├── message/{sessionID}/{messageID}.json
│   { "id": "msg_...", "role": "user|assistant", "tokens": {...}, "cost": ... }
│
├── part/{messageID}/{partID}.json
│   { "id": "prt_...", "type": "text|tool", "text": "...", "state": {...} }
│
└── session_diff/{sessionID}.json
    [{ "file": "...", "additions": N, "deletions": N }]
```

---

## Environment Variables

```bash
# Override data directory (existing, keep for Claude)
CLAUDE_DIR="$HOME/.claude"

# New: OpenCode data directory override
OPENCODE_DIR="$HOME/.local/share/opencode"

# New: Generic cache location override  
SESSIONS_CACHE_DIR="$HOME/.cache/sessions-tui"
```

---

## Open Questions

1. **Repo rename?** Consider `ai-sessions-tui` or `coding-sessions-tui` for clarity
2. **Unified cache?** Single cache file vs per-adapter cache files
3. **Cross-adapter search?** Future feature to search across all agents at once
