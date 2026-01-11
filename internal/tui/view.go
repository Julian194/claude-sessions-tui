package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate pane dimensions (border takes 2 chars each side)
	listWidth := m.width / 2
	previewWidth := m.width - listWidth

	// Content dimensions (subtract border)
	listContentWidth := listWidth - 2
	previewContentWidth := previewWidth - 2
	contentHeight := m.height - 4

	// Apply borders based on active pane
	var listPane, previewPane string

	listContent := m.list.View()
	previewContent := m.preview.View()

	if m.activePane == "list" {
		listPane = activeBorderStyle.
			Width(listContentWidth).
			Height(contentHeight).
			Render(listContent)
		previewPane = inactiveBorderStyle.
			Width(previewContentWidth).
			Height(contentHeight).
			Render(previewContent)
	} else {
		listPane = inactiveBorderStyle.
			Width(listContentWidth).
			Height(contentHeight).
			Render(listContent)
		previewPane = activeBorderStyle.
			Width(previewContentWidth).
			Height(contentHeight).
			Render(previewContent)
	}

	// Join panes horizontally
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)

	// Status bar with filter indicator
	filterIndicator := ""
	switch m.filterMode {
	case FilterToday:
		filterIndicator = " [Today]"
	case FilterWeek:
		filterIndicator = " [Week]"
	case FilterHighCost:
		filterIndicator = " [$$$]"
	case FilterProject:
		filterIndicator = " [" + m.filterValue + "]"
	}

	helpText := "?:help  enter:resume  space:expand  p:pin  1-4:filter  ctrl+o:export  ctrl+y:copy  q:quit"
	if m.filterMode != FilterNone {
		helpText = "esc:clear  " + helpText
	}
	help := helpStyle.Render(helpText)
	status := statusBarStyle.Render(m.message + filterIndicator)
	statusBar := lipgloss.JoinHorizontal(lipgloss.Left, status, "  ", help)

	view := lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)

	// Help overlay
	if m.showHelp {
		return renderHelpOverlay(m.width, m.height)
	}

	return view
}

// updateLayout recalculates component sizes
func (m *Model) updateLayout() {
	listWidth := m.width / 2
	previewWidth := m.width - listWidth

	// List content area (inside border: -2 for border, -2 for padding)
	m.list.SetSize(listWidth-4, m.height-6)

	// Preview viewport (inside border)
	m.preview.Width = previewWidth - 4
	m.preview.Height = m.height - 6
}

// updatePreview updates the preview pane content
func (m *Model) updatePreview() {
	item := m.getSelectedSession()
	if item == nil {
		m.preview.SetContent("No session selected")
		return
	}

	if m.showActivity {
		m.preview.SetContent("Activity heatmap (TODO)")
		return
	}

	// Generate preview content
	var b strings.Builder

	// Header with styling
	b.WriteString(fmt.Sprintf("ID: %s\n", item.entry.SessionID))
	b.WriteString(fmt.Sprintf("Project: %s\n", item.entry.Project))
	b.WriteString(fmt.Sprintf("Date: %s\n", item.entry.Date.Format("2006-01-02 15:04")))

	if item.isAgent {
		b.WriteString(fmt.Sprintf("Parent: %s\n", item.entry.ParentSID))
	}

	// Get additional info from adapter
	if info, err := m.adapter.GetSessionInfo(item.entry.SessionID); err == nil {
		if info.Branch != "" {
			b.WriteString(fmt.Sprintf("Branch: %s\n", info.Branch))
		}
	}

	// Models
	if models, err := m.adapter.GetModels(item.entry.SessionID); err == nil && len(models) > 0 {
		b.WriteString(fmt.Sprintf("Models: %s\n", strings.Join(models, ", ")))
	}

	b.WriteString("\n")

	// Summaries
	if summaries, err := m.adapter.GetSummaries(item.entry.SessionID); err == nil && len(summaries) > 0 {
		b.WriteString("━━━ Topics ━━━\n")
		for _, s := range summaries {
			b.WriteString(fmt.Sprintf("• %s\n", s))
		}
		b.WriteString("\n")
	}

	// Slash commands
	if cmds, err := m.adapter.GetSlashCommands(item.entry.SessionID); err == nil && len(cmds) > 0 {
		b.WriteString("━━━ Slash Commands ━━━\n")
		for _, cmd := range cmds {
			b.WriteString(fmt.Sprintf("  %s\n", cmd))
		}
		b.WriteString("\n")
	}

	// Files
	if files, err := m.adapter.GetFilesTouched(item.entry.SessionID); err == nil && len(files) > 0 {
		b.WriteString("━━━ Files ━━━\n")
		shown := files
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, f := range shown {
			b.WriteString(fmt.Sprintf("• %s\n", f))
		}
		if len(files) > 10 {
			b.WriteString(fmt.Sprintf("  ... and %d more\n", len(files)-10))
		}
		b.WriteString("\n")
	}

	// Stats
	if stats, err := m.adapter.GetStats(item.entry.SessionID); err == nil {
		b.WriteString("━━━ Stats ━━━\n")
		b.WriteString(fmt.Sprintf("Messages: %d user, %d assistant\n", stats.UserMessages, stats.AssistantMessages))
		b.WriteString(fmt.Sprintf("Tokens: %d in, %d out", stats.InputTokens, stats.OutputTokens))
		if stats.CacheRead > 0 || stats.CacheWrite > 0 {
			b.WriteString(fmt.Sprintf(", %d cache", stats.CacheRead+stats.CacheWrite))
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Cost: $%.4f\n", stats.Cost))
		b.WriteString("\n")
	}

	// First message (if no summaries)
	if summaries, _ := m.adapter.GetSummaries(item.entry.SessionID); len(summaries) == 0 {
		if msg, err := m.adapter.GetFirstMessage(item.entry.SessionID); err == nil && msg != "" {
			b.WriteString("━━━ First Message ━━━\n")
			b.WriteString(msg)
			b.WriteString("\n")
		}
	}

	m.preview.SetContent(b.String())
	m.preview.GotoTop()
}
