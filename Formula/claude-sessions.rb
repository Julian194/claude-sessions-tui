class ClaudeSessions < Formula
  desc "TUI for browsing, searching, and exporting Claude Code sessions"
  homepage "https://github.com/yourusername/claude-sessions-tui"
  url "https://github.com/yourusername/claude-sessions-tui/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"
  head "https://github.com/yourusername/claude-sessions-tui.git", branch: "main"

  depends_on "fzf"
  depends_on "python@3.11" => :recommended

  def install
    bin.install "bin/claude-sessions"
    bin.install "bin/claude-sessions-preview"
    bin.install "bin/claude-sessions-rebuild"
    bin.install "bin/claude-sessions-stats"
    bin.install "bin/claude-sessions-export"
  end

  def caveats
    <<~EOS
      claude-sessions requires Claude Code to be installed and have existing sessions.
      Sessions are stored in ~/.claude/projects/

      Usage:
        claude-sessions          # Launch the TUI browser
        claude-sessions-rebuild  # Manually rebuild the session cache

      Keyboard shortcuts in the TUI:
        Enter   - Resume selected session
        Ctrl-O  - Export session as HTML
        Ctrl-R  - Refresh session list
        Esc     - Exit
    EOS
  end

  test do
    assert_match "Usage:", shell_output("#{bin}/claude-sessions-stats 2>&1", 1)
  end
end
