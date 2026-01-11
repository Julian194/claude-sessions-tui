package tui

import "github.com/charmbracelet/lipgloss"

const helpContent = `
  Keyboard Shortcuts
  ──────────────────────────────────────

  Navigation
    ↑/k, ↓/j     Move up/down
    Tab          Switch pane (list/preview)
    /            Search/filter sessions

  Session Actions
    Enter        Resume selected session
    Ctrl+B       Branch session (copy & resume)
    Ctrl+E       Open session file in VS Code
    Ctrl+O       Export to HTML
    Ctrl+Y       Copy as markdown

  Quick Filters
    1            Today's sessions only
    2            This week's sessions
    3            Current project only
    Esc          Clear active filter

  Parent/Child Sessions
    Space        Toggle expand/collapse
    →/l          Expand children
    ←/h          Collapse children

  Other
    p            Pin/unpin session
    Ctrl+R       Refresh cache
    Ctrl+A       Toggle activity heatmap
    ?            Toggle this help
    q, Ctrl+C    Quit

  Press ? or Esc to close
`

// renderHelpOverlay renders the centered help popup
func renderHelpOverlay(width, height int) string {
	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(1, 2).
		Render(helpContent)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		helpBox)
}
