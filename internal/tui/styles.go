package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	cyan      = lipgloss.Color("#00FFFF")
	dimWhite  = lipgloss.Color("#808080")
	white     = lipgloss.Color("#FFFFFF")
	highlight = lipgloss.Color("#0066FF")
	green     = lipgloss.Color("#00FF00")
	purple    = lipgloss.Color("#BD93F9")
	orange    = lipgloss.Color("#FFB86C")
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

	// Date header in list
	dateHeaderStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(dimWhite).
			Padding(0, 1)

	// Preview section headers
	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(cyan).
				Bold(true)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(dimWhite)

	// Preview pane styles
	labelStyle = lipgloss.NewStyle().
			Foreground(dimWhite)

	valueStyle = lipgloss.NewStyle().
			Foreground(white)

	projectStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	idStyle = lipgloss.NewStyle().
		Foreground(dimWhite)

	branchStyle = lipgloss.NewStyle().
			Foreground(purple)

	modelStyle = lipgloss.NewStyle().
			Foreground(orange)

	// Cost
	costStyle = lipgloss.NewStyle().
		Foreground(green)

	// Stats
	statValueStyle = lipgloss.NewStyle().
			Foreground(white).
			Bold(true)

	// Bullet points
	bulletStyle = lipgloss.NewStyle().
			Foreground(cyan)

	// File path
	filePathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD"))

	// Slash command
	slashCmdStyle = lipgloss.NewStyle().
			Foreground(purple)
)
