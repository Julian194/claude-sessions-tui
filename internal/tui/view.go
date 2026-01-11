package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Julian194/claude-sessions-tui/internal/heatmap"
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
	listBorder := inactiveBorderStyle
	previewBorder := inactiveBorderStyle
	if m.activePane == "list" {
		listBorder = activeBorderStyle
	} else {
		previewBorder = activeBorderStyle
	}

	listPane := listBorder.Width(listContentWidth).Height(contentHeight).Render(m.list.View())
	previewPane := previewBorder.Width(previewContentWidth).Height(contentHeight).Render(m.preview.View())

	// Join panes horizontally
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)

	// Status bar with filter indicator
	filterIndicator := ""
	switch m.filterMode {
	case FilterToday:
		filterIndicator = " [Today]"
	case FilterWeek:
		filterIndicator = " [Week]"
	case FilterProject:
		filterIndicator = " [" + m.filterValue + "]"
	}
	if m.showActivity {
		filterIndicator += " [Activity]"
	}

	helpText := "?:help  enter:resume  p:pin  1-3:filter  ctrl+a:activity  ctrl+o:export  ctrl+y:copy  q:quit"
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
	// Activity heatmap doesn't need a selected session
	if m.showActivity {
		m.preview.SetContent(heatmap.RenderFromCache(m.sessions, m.preview.Width, m.activityYear))
		return
	}

	item := m.getSelectedSession()
	if item == nil {
		m.preview.SetContent(dimItemStyle.Render("No session selected"))
		return
	}

	// Generate preview content
	var b strings.Builder

	// Project name as title
	b.WriteString(projectStyle.Render(item.entry.Project))
	b.WriteString("\n")

	// Date prominently displayed
	b.WriteString(valueStyle.Render(item.entry.Date.Format("Mon, Jan 2 2006 at 15:04")))
	b.WriteString("\n\n")

	// Metadata section
	b.WriteString(labelStyle.Render("ID "))
	b.WriteString(idStyle.Render(item.entry.SessionID))
	b.WriteString("\n")

	if item.isAgent {
		b.WriteString(labelStyle.Render("Parent "))
		b.WriteString(idStyle.Render(item.entry.ParentSID))
		b.WriteString("\n")
	}

	// Get additional info from adapter
	if info, err := m.adapter.GetSessionInfo(item.entry.SessionID); err == nil {
		if info.Branch != "" {
			b.WriteString(labelStyle.Render("Branch "))
			b.WriteString(branchStyle.Render(info.Branch))
			b.WriteString("\n")
		}
	}

	// Models
	if models, err := m.adapter.GetModels(item.entry.SessionID); err == nil && len(models) > 0 {
		b.WriteString(labelStyle.Render("Model "))
		b.WriteString(modelStyle.Render(strings.Join(models, ", ")))
		b.WriteString("\n")
	}

	// Stats (placed early for visibility)
	if stats, err := m.adapter.GetStats(item.entry.SessionID); err == nil {
		b.WriteString("\n")
		b.WriteString(renderSectionHeader("Stats"))
		b.WriteString("\n")

		// Cost
		b.WriteString(labelStyle.Render("Cost "))
		b.WriteString(costStyle.Render(fmt.Sprintf("$%.4f", stats.Cost)))
		b.WriteString("\n")

		// Messages
		b.WriteString(labelStyle.Render("Messages "))
		b.WriteString(statValueStyle.Render(fmt.Sprintf("%d", stats.UserMessages)))
		b.WriteString(dimItemStyle.Render(" user, "))
		b.WriteString(statValueStyle.Render(fmt.Sprintf("%d", stats.AssistantMessages)))
		b.WriteString(dimItemStyle.Render(" assistant"))
		b.WriteString("\n")

		// Tokens
		b.WriteString(labelStyle.Render("Tokens "))
		b.WriteString(statValueStyle.Render(formatTokens(stats.InputTokens)))
		b.WriteString(dimItemStyle.Render(" in, "))
		b.WriteString(statValueStyle.Render(formatTokens(stats.OutputTokens)))
		b.WriteString(dimItemStyle.Render(" out"))
		if stats.CacheRead > 0 || stats.CacheWrite > 0 {
			b.WriteString(dimItemStyle.Render(", "))
			b.WriteString(statValueStyle.Render(formatTokens(stats.CacheRead + stats.CacheWrite)))
			b.WriteString(dimItemStyle.Render(" cache"))
		}
		b.WriteString("\n")
	}

	// Summaries/Topics
	if summaries, err := m.adapter.GetSummaries(item.entry.SessionID); err == nil && len(summaries) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSectionHeader("Topics"))
		b.WriteString("\n")
		for _, s := range summaries {
			b.WriteString(bulletStyle.Render("▸ "))
			b.WriteString(valueStyle.Render(s))
			b.WriteString("\n")
		}
	}

	// Slash commands
	if cmds, err := m.adapter.GetSlashCommands(item.entry.SessionID); err == nil && len(cmds) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSectionHeader("Commands"))
		b.WriteString("\n")
		for _, cmd := range cmds {
			b.WriteString("  ")
			b.WriteString(slashCmdStyle.Render(cmd))
			b.WriteString("\n")
		}
	}

	// Files
	if files, err := m.adapter.GetFilesTouched(item.entry.SessionID); err == nil && len(files) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSectionHeader(fmt.Sprintf("Files (%d)", len(files))))
		b.WriteString("\n")
		shown := files
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, f := range shown {
			b.WriteString(bulletStyle.Render("▸ "))
			b.WriteString(filePathStyle.Render(f))
			b.WriteString("\n")
		}
		if len(files) > 10 {
			b.WriteString(dimItemStyle.Render(fmt.Sprintf("  + %d more files\n", len(files)-10)))
		}
	}

	// First message (if no summaries)
	if summaries, _ := m.adapter.GetSummaries(item.entry.SessionID); len(summaries) == 0 {
		if msg, err := m.adapter.GetFirstMessage(item.entry.SessionID); err == nil && msg != "" {
			b.WriteString("\n")
			b.WriteString(renderSectionHeader("First Message"))
			b.WriteString("\n")
			b.WriteString(dimItemStyle.Render(msg))
			b.WriteString("\n")
		}
	}

	m.preview.SetContent(b.String())
	m.preview.GotoTop()
}

// renderSectionHeader creates a styled section header
func renderSectionHeader(title string) string {
	return sectionHeaderStyle.Render("─── " + title + " ───")
}

// formatTokens formats token counts with K suffix for readability
func formatTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
