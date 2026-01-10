package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	cyan      = lipgloss.Color("#00FFFF")
	dimWhite  = lipgloss.Color("#808080")
	white     = lipgloss.Color("#FFFFFF")
	highlight = lipgloss.Color("#0066FF")
	green     = lipgloss.Color("#00FF00")
	yellow    = lipgloss.Color("#FFFF00")
)

// Styles
var (
	// Pane borders
	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(cyan)

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(dimWhite)

	// List item styles
	normalItemStyle = lipgloss.NewStyle().
			Foreground(white)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(white).
				Background(highlight).
				Bold(true)

	dimItemStyle = lipgloss.NewStyle().
			Foreground(dimWhite)

	// Date header style
	dateHeaderStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	// Pin indicator
	pinStyle = lipgloss.NewStyle().
			Foreground(yellow)

	// Child indicator (branched/agent sessions)
	childIndicatorStyle = lipgloss.NewStyle().
				Foreground(dimWhite)

	// Agent session items (slightly dimmed)
	agentItemStyle = lipgloss.NewStyle().
			Foreground(dimWhite).
			Italic(true)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(dimWhite).
			Padding(0, 1)

	// Preview header
	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true)

	// Preview section headers
	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(cyan).
				Bold(true)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(dimWhite)
)
