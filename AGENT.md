# Agent Instructions

## Development Workflow

When making changes to this project, **always** complete these steps before considering the task done:

### 1. Run Tests
```bash
go test ./internal/... -v
```

### 2. Rebuild Binaries
```bash
go build -o claude-sessions ./cmd/sessions
```

### 3. Install Dev Binaries (to ~/.local/bin/)
```bash
cp claude-sessions ~/.local/bin/claude-sessions-dev
cp claude-sessions ~/.local/bin/opencode-sessions-dev
```

**DO NOT** overwrite Homebrew binaries in `/opt/homebrew/bin/`. Keep dev and production separate.

### 4. Rebuild Cache (if data structures changed)
```bash
claude-sessions-dev rebuild
opencode-sessions-dev rebuild
```

### One-liner
```bash
go test ./internal/... && go build -o claude-sessions ./cmd/sessions && cp claude-sessions ~/.local/bin/claude-sessions-dev && cp claude-sessions ~/.local/bin/opencode-sessions-dev
```

## Dev vs Production

| Version | Location | Usage |
|---------|----------|-------|
| Dev | `~/.local/bin/*-dev` | Testing local changes |
| Production | `/opt/homebrew/bin/` | Installed via Homebrew |

User can set up shell aliases to switch between them.

## Project Structure

- `cmd/sessions/` - CLI entry point (binary name determines adapter)
- `internal/adapters/claude/` - Claude Code adapter
- `internal/adapters/opencode/` - OpenCode adapter  
- `internal/tui/` - fzf TUI integration
- `internal/cache/` - Session cache management
- `internal/export/` - HTML/Markdown export
- `internal/stats/` - Token counting and cost calculation

## Binary Naming

The same binary serves both adapters - the binary name determines which adapter is used:
- `claude-sessions` or `claude` → Claude Code adapter
- `opencode-sessions` or `opencode` → OpenCode adapter

## Feature Implementation Rule

**When adding new features, ALWAYS check feasibility for ALL supported adapters and implement for all unless explicitly told otherwise.**

Both adapters (Claude Code and OpenCode) should have feature parity where possible. Check data availability in both before implementing.

