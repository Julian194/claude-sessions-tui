package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

// loadSessionsMsg is sent when sessions are loaded
type loadSessionsMsg struct {
	entries []cache.Entry
	err     error
}

// refreshCacheMsg is sent when cache is refreshed
type refreshCacheMsg struct {
	entries []cache.Entry
	err     error
}

// clearMessageMsg clears the status message
type clearMessageMsg struct{}

func clearMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

// previewUpdateMsg triggers a debounced preview update
type previewUpdateMsg struct {
	index int // the index that requested this update
}

func requestPreviewUpdate(index int) tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
		return previewUpdateMsg{index: index}
	})
}
