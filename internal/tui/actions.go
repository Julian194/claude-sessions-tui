package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		filename := filepath.Join(os.TempDir(), fmt.Sprintf("session-%s.html", shortID))
		if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
			return exportMsg{err: err}
		}

		// Open in browser (cross-platform)
		switch {
		case commandExists("open"): // macOS
			exec.Command("open", filename).Start()
		case commandExists("xdg-open"): // Linux
			exec.Command("xdg-open", filename).Start()
		case commandExists("wslview"): // WSL
			// Convert Unix path to Windows path for WSL
			winPath, err := exec.Command("wslpath", "-w", filename).Output()
			if err == nil {
				exec.Command("wslview", strings.TrimSpace(string(winPath))).Start()
			}
		}

		return exportMsg{filename: filename}
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
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
		tmpFile := filepath.Join(os.TempDir(), "claude-session-clipboard.md")
		if err := os.WriteFile(tmpFile, []byte(md), 0644); err != nil {
			return copyMsg{err: err}
		}

		// Find clipboard tool and copy from file
		var clipCmd *exec.Cmd
		switch {
		case commandExists("pbcopy"): // macOS
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("pbcopy < %s", tmpFile))
		case commandExists("wl-copy"): // Wayland
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("wl-copy < %s", tmpFile))
		case commandExists("xclip"): // X11
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("xclip -selection clipboard < %s", tmpFile))
		case commandExists("xsel"): // X11 alternative
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("xsel --clipboard --input < %s", tmpFile))
		case commandExists("clip.exe"): // WSL
			clipCmd = exec.Command("sh", "-c", fmt.Sprintf("clip.exe < %s", tmpFile))
		default:
			return copyMsg{err: fmt.Errorf("no clipboard tool found")}
		}

		clipCmd.Env = os.Environ()
		if err := clipCmd.Run(); err != nil {
			return copyMsg{err: err}
		}

		return copyMsg{}
	}
}
