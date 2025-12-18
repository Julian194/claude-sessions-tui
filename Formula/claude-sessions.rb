class ClaudeSessions < Formula
  desc "TUI for browsing, searching, and exporting Claude Code sessions"
  homepage "https://github.com/Julian194/claude-sessions-tui"
  url "https://github.com/Julian194/claude-sessions-tui.git", tag: "v0.3.0"
  version "0.3.0"
  license "MIT"
  head "https://github.com/Julian194/claude-sessions-tui.git", branch: "main"

  depends_on "go" => :build
  depends_on "fzf"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"claude-sessions", "./cmd/sessions"
  end

  def caveats
    <<~EOS
      claude-sessions requires Claude Code to be installed and have existing sessions.
      Sessions are stored in ~/.claude/projects/

      Usage:
        claude-sessions          # Launch the TUI browser
        claude-sessions rebuild  # Manually rebuild the session cache

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
