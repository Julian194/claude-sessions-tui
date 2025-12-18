# claude-sessions-tui

A terminal UI for browsing, searching, and exporting [Claude Code](https://claude.ai/code) sessions.

![Demo](demo.gif)

## Features

- **Browse sessions** with fuzzy search via fzf
- **Preview pane** showing topics, files touched, and stats
- **Resume sessions** directly from the TUI
- **Branch sessions** - create a copy of any session to explore alternative paths
- **Export to HTML** with dark/light themes, search, and syntax highlighting
- **Copy as Markdown** - LLM-optimized format copied to clipboard
- **Session statistics** including token usage, tool calls, and cost estimates
- **Cross-platform** support for macOS and Linux

## Installation

### Homebrew (recommended)

```bash
brew install julian194/tap/claude-sessions
```

### Manual installation

```bash
git clone https://github.com/Julian194/claude-sessions-tui.git
cd claude-sessions-tui
chmod +x bin/*

# Add to PATH (add to your .bashrc, .zshrc, or config.fish)
export PATH="$PATH:$(pwd)/bin"
```

### Dependencies

- [fzf](https://github.com/junegunn/fzf) - fuzzy finder
- Python 3 (for statistics)
- [Claude Code](https://claude.ai/code) - the AI coding assistant

## Usage

### Launch the TUI

```bash
claude-sessions
```

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Resume selected session |
| `Ctrl-B` | Branch session (create copy and resume) |
| `Ctrl-O` | Export session as HTML |
| `Ctrl-Y` | Copy session as LLM-optimized Markdown |
| `Ctrl-R` | Refresh session list |
| `↑/↓` | Navigate sessions |
| Type | Filter sessions |
| `Esc` | Exit |

### Individual commands

```bash
# Rebuild the session cache manually
claude-sessions-rebuild

# View statistics for a specific session
claude-sessions-stats <session-id>
claude-sessions-stats <session-id> --full

# Export a session to HTML
claude-sessions-export <session-id>

# Copy session as Markdown to clipboard
claude-sessions-copy-md <session-id>

# Branch a session (create a copy)
claude-sessions-branch <session-id>

# Preview a session (used internally by fzf)
claude-sessions-preview <session-id>
```

## Preview pane

The preview shows:
- Session ID, project name, and date
- Git branch (if available)
- **Topics** - AI-generated summaries of conversation segments
- **Files** - Files that were modified during the session
- **Stats** - Message count, tool calls, token usage, and estimated cost

## HTML Export

The exported HTML includes:
- Beautiful dark/light theme toggle
- Full-text search with highlighting
- Syntax highlighting for code blocks
- Session metadata (date, branch, stats)
- Responsive design for mobile

## Configuration

Set `CLAUDE_DIR` environment variable to use a custom Claude Code directory:

```bash
export CLAUDE_DIR="$HOME/.claude"  # default
```

## How it works

Claude Code stores sessions as JSONL files in `~/.claude/projects/`. Each project has its own directory with session files named by UUID.

The TUI:
1. Scans all session files and extracts metadata
2. Caches results for fast subsequent launches (auto-refreshes after 1 hour)
3. Uses fzf for interactive filtering
4. Can resume sessions or export them

## License

MIT
