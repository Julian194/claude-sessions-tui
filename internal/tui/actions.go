package tui

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Julian194/claude-sessions-tui/internal/export"
)

// exportMsg is sent when export completes
type exportMsg struct {
	filename string
	err      error
}

// copyMsg is sent when copy completes
type copyMsg struct {
	err error
}

// exportHTML exports a session to HTML and opens it in browser
func (m Model) exportHTML(sid string) tea.Cmd {
	return func() tea.Msg {
		messages, err := m.adapter.ExportMessages(sid)
		if err != nil {
			return exportMsg{err: err}
		}

		info, err := m.adapter.GetSessionInfo(sid)
		if err != nil {
			return exportMsg{err: err}
		}

		models, _ := m.adapter.GetModels(sid)
		html := export.ToHTML(messages, info, models)

		shortID := sid
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		filename := fmt.Sprintf("/tmp/session-%s.html", shortID)
		if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
			return exportMsg{err: err}
		}

		// Open in browser
		if cmd, err := exec.LookPath("xdg-open"); err == nil {
			exec.Command(cmd, filename).Start()
		}

		return exportMsg{filename: filename}
	}
}

// copyMarkdown copies a session as markdown to clipboard
func (m Model) copyMarkdown(sid string) tea.Cmd {
	return func() tea.Msg {
		messages, err := m.adapter.ExportMessages(sid)
		if err != nil {
			return copyMsg{err: err}
		}

		info, _ := m.adapter.GetSessionInfo(sid)
		models, _ := m.adapter.GetModels(sid)
		md := export.ToMarkdown(messages, info, models)

		// Write to temp file first (more reliable on Wayland)
		// Note: don't delete - wl-copy forks and needs the file
		tmpFile := "/tmp/claude-session-clipboard.md"
		if err := os.WriteFile(tmpFile, []byte(md), 0644); err != nil {
			return copyMsg{err: err}
		}

		// Find clipboard tool and copy from file
		var clipCmd *exec.Cmd
		if _, err := exec.LookPath("wl-copy"); err == nil {
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("wl-copy < %s", tmpFile))
		} else if _, err := exec.LookPath("xclip"); err == nil {
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("xclip -selection clipboard < %s", tmpFile))
		} else if _, err := exec.LookPath("pbcopy"); err == nil {
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("pbcopy < %s", tmpFile))
		} else {
			return copyMsg{err: fmt.Errorf("no clipboard tool found")}
		}

		clipCmd.Env = os.Environ()
		if err := clipCmd.Run(); err != nil {
			return copyMsg{err: err}
		}

		return copyMsg{}
	}
}
