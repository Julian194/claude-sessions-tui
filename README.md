# claude-sessions-tui

A terminal UI for browsing, searching, and exporting [Claude Code](https://claude.ai/code) sessions.

![Demo](demo.gif)

## Features

- **Browse sessions** with built-in fuzzy search
- **Preview pane** showing topics, files touched, and stats
- **Quick filters** - Today (1), This week (2), Project (3), High cost (4)
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

### From source

Requires Go 1.21+:

```bash
git clone https://github.com/Julian194/claude-sessions-tui.git
cd claude-sessions-tui
go build -o claude-sessions ./cmd/sessions
sudo mv claude-sessions /usr/local/bin/
```

### Windows (WSL)

Claude Code runs on Windows via WSL. The TUI works inside WSL and reads sessions from the Windows host.

**Prerequisites:** WSL2 with Ubuntu (or similar), Claude Code installed on Windows.

1. **Install inside WSL** (Homebrew for Linux or from source):
   ```bash
   # Option A: Homebrew
   brew install julian194/tap/claude-sessions

   # Option B: From source
   git clone https://github.com/Julian194/claude-sessions-tui.git
   cd claude-sessions-tui
   go build -o claude-sessions ./cmd/sessions
   sudo mv claude-sessions /usr/local/bin/
   ```

2. **Point to Windows sessions** - add to your `~/.bashrc` or `~/.zshrc`:
   ```bash
   export CLAUDE_DIR="/mnt/c/Users/<your-windows-username>/.claude"
   ```

3. **Clipboard** works out of the box - the TUI uses `clip.exe` which is available in WSL by default.

4. **Browser / HTML export** - install `wslu` for `wslview` to open exports in your Windows browser:
   ```bash
   sudo apt install wslu
   ```

5. **Optional alias:**
   ```bash
   alias cs='claude-sessions'
   ```

### Dependencies

- [Claude Code](https://claude.ai/code) - the AI coding assistant

## Usage

### Launch the TUI

```bash
claude-sessions
```

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `↑/k`, `↓/j` | Navigate sessions |
| `Tab` | Switch pane (list/preview) |
| `/` | Search/filter sessions |
| `Enter` | Resume selected session |
| `Ctrl+B` | Branch session (create copy and resume) |
| `Ctrl+E` | Open session file in VS Code |
| `Ctrl+O` | Export session as HTML |
| `Ctrl+Y` | Copy session as Markdown |
| `Ctrl+R` | Refresh session cache |
| `Ctrl+A` | Toggle activity heatmap |
| `1-4` | Quick filters (Today/Week/Project/Cost) |
| `Esc` | Clear active filter |
| `Space` | Toggle expand/collapse parent session |
| `→/l`, `←/h` | Expand/collapse children |
| `p` | Pin/unpin session |
| `?` | Show help overlay |
| `q`, `Ctrl+C` | Quit |

### Subcommands

```bash
# Launch the TUI (default)
claude-sessions

# Rebuild the session cache manually
claude-sessions rebuild

# View statistics for a specific session
claude-sessions stats <session-id>

# Export a session to HTML
claude-sessions export <session-id>

# Copy session as Markdown to clipboard
claude-sessions copy-md <session-id>

# Show help
claude-sessions help
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

### Environment variables

```bash
# Override Claude data directory
export CLAUDE_DIR="$HOME/.claude"

# Override cache directory
export SESSIONS_CACHE_DIR="$HOME/.cache/sessions-tui"
```

## How it works

Claude Code stores sessions as JSONL files in `~/.claude/projects/`. Each project has its own directory with session files named by UUID.

The TUI:
1. Scans all session files and extracts metadata
2. Caches results for fast subsequent launches
3. Provides interactive filtering, pinning, and quick filters
4. Can resume sessions, branch them, or export them

## Architecture

Built in Go with a provider/adapter architecture to support multiple AI coding assistants:

```
cmd/sessions/        # CLI entry point
internal/
  adapters/          # Provider implementations
    claude/          # Claude Code adapter
    opencode/        # OpenCode adapter
  cache/             # Session cache management
  export/            # HTML/Markdown export
  stats/             # Token counting and cost calculation
  tui/               # Bubble Tea TUI
```

## Development

### Building from source

```bash
go test ./internal/...
go build -o claude-sessions ./cmd/sessions
```

### Installing dev binaries

Install to `~/.local/bin/` with a `-dev` suffix to keep them separate from production (Homebrew) installs:

```bash
cp claude-sessions ~/.local/bin/claude-sessions-dev
cp claude-sessions ~/.local/bin/opencode-sessions-dev
```

### Shell aliases

Add these to your shell config (e.g., `~/.config/fish/config.fish` or `~/.zshrc`):

```bash
# Production (Homebrew)
alias cs='/opt/homebrew/bin/claude-sessions'

# Dev (local builds)
alias csd='~/.local/bin/claude-sessions-dev'
alias osd='~/.local/bin/opencode-sessions-dev'
```

### Binary naming

The same binary serves both Claude Code and OpenCode - the adapter is selected based on the binary name:

| Binary name contains | Adapter |
|---------------------|---------|
| `opencode` | OpenCode |
| anything else | Claude Code |

## Troubleshooting

### Cache directory errors

If you see an error like `failed to create cache: open ...sessions-cache.tsv: no such file or directory`, create the cache directories manually:

```bash
# For Claude Code
mkdir -p ~/.claude/.cache

# For OpenCode
mkdir -p ~/.local/share/opencode/.cache
```

## License

MIT
