class ClaudeSessions < Formula
  desc "TUI for browsing, searching, and exporting Claude Code sessions"
  homepage "https://github.com/Julian194/claude-sessions-tui"
  url "https://github.com/Julian194/claude-sessions-tui.git", tag: "v0.2.6"
  version "0.2.6"
  license "MIT"
  head "https://github.com/Julian194/claude-sessions-tui.git", branch: "main"

  depends_on "go" => :build
  depends_on "fzf"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin / "claude-sessions", "./cmd/sessions"
    bin.install_symlink "claude-sessions" => "opencode-sessions"
  end

  def caveats
    <<~EOS
      This package provides session browsers for AI coding assistants:
        claude-sessions   - Browse Claude Code sessions (~/.claude/projects/)
        opencode-sessions - Browse OpenCode sessions (~/.local/share/opencode/storage/)

      Usage:
        claude-sessions          # Launch Claude TUI browser
        opencode-sessions        # Launch OpenCode TUI browser
        <cmd> rebuild            # Manually rebuild the session cache

      Keyboard shortcuts in the TUI:
        Enter   - Resume selected session
        Ctrl-O  - Export session as HTML
        Ctrl-Y  - Copy session as markdown
        Ctrl-B  - Branch session
        Ctrl-R  - Refresh session list
        Esc     - Exit
    EOS
  end

  test do
    assert_match "Usage:", shell_output("#{bin}/claude-sessions help")
  end
end
