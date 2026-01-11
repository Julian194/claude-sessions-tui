package heatmap

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

// GitHub-style contribution colors
var levelStyles = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#161b22")), // level 0: empty
	lipgloss.NewStyle().Foreground(lipgloss.Color("#0e4429")), // level 1: low
	lipgloss.NewStyle().Foreground(lipgloss.Color("#006d32")), // level 2: medium-low
	lipgloss.NewStyle().Foreground(lipgloss.Color("#26a641")), // level 3: medium
	lipgloss.NewStyle().Foreground(lipgloss.Color("#39d353")), // level 4: high
}

var (
	dimStyle   = lipgloss.NewStyle().Faint(true)
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#39d353"))
)

const block = "■"

// RenderFromCache renders a heatmap from cache entries for a specific year
func RenderFromCache(entries []cache.Entry, width int, year int) string {
	activity := make(map[string]int)
	for _, e := range entries {
		if e.Date.Year() == year {
			dateKey := e.Date.Format("2006-01-02")
			activity[dateKey]++
		}
	}

	weeks := calculateWeeks(width)
	return Render(activity, weeks, year)
}

func calculateWeeks(width int) int {
	if width <= 0 {
		width = 60
	}
	// Account for day labels (4 chars) and spacing (each week = 2 chars)
	weeks := (width - 5) / 2
	return max(8, min(52, weeks))
}

// Render creates a GitHub-style activity heatmap
func Render(activity map[string]int, maxWeeks int, year int) string {
	var sb strings.Builder

	today := time.Now()

	// Calculate grid boundaries for the year
	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, today.Location())
	dec31 := time.Date(year, 12, 31, 0, 0, 0, 0, today.Location())

	// Align to week boundaries (Monday start)
	startOffset := int(jan1.Weekday())
	if startOffset == 0 {
		startOffset = 7
	}
	start := jan1.AddDate(0, 0, -(startOffset - 1))

	endOffset := int(dec31.Weekday())
	if endOffset == 0 {
		endOffset = 7
	}
	end := dec31.AddDate(0, 0, 7-endOffset)

	totalWeeks := int(end.Sub(start).Hours()/24/7) + 1

	// Limit to maxWeeks
	weeks := totalWeeks
	if weeks > maxWeeks {
		// For past years: show END of year (where data is)
		// For current year: show START of year (up to today)
		if year < today.Year() {
			startWeekOffset := weeks - maxWeeks
			start = start.AddDate(0, 0, startWeekOffset*7)
		}
		weeks = maxWeeks
	}

	// Title
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Activity %d", year)))
	sb.WriteString("\n\n")

	// Month header
	sb.WriteString(renderMonthHeader(start, weeks, year))

	// Grid
	days := []string{"Mon", "   ", "Wed", "   ", "Fri", "   ", "   "}
	maxCount := findMax(activity)

	for d := 0; d < 7; d++ {
		sb.WriteString(dimStyle.Render(days[d]) + " ")

		for w := 0; w < weeks; w++ {
			date := start.AddDate(0, 0, 7*w+d)

			// Skip dates outside current year
			if date.Year() != year {
				sb.WriteString("  ")
				continue
			}

			// Future dates show as empty (level 0)
			if date.After(today) {
				sb.WriteString(levelStyles[0].Render(block) + " ")
				continue
			}

			dateKey := date.Format("2006-01-02")
			count := activity[dateKey]
			level := countToLevel(count, maxCount)

			sb.WriteString(levelStyles[level].Render(block) + " ")
		}
		sb.WriteString("\n")
	}

	// Stats
	total := 0
	for _, count := range activity {
		total += count
	}
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render(fmt.Sprintf("%d sessions in %d", total, year)))
	sb.WriteString("\n\n")

	// Legend
	sb.WriteString(dimStyle.Render("Less "))
	for _, style := range levelStyles {
		sb.WriteString(style.Render(block) + " ")
	}
	sb.WriteString(dimStyle.Render("More"))
	sb.WriteString("\n\n")

	// Navigation hint
	sb.WriteString(dimStyle.Render("← prev year    → next year    Ctrl+A close"))

	return sb.String()
}

func renderMonthHeader(start time.Time, weeks int, year int) string {
	header := make([]byte, 4+weeks*2)
	for i := range header {
		header[i] = ' '
	}

	currentMonth := -1
	for w := 0; w < weeks; w++ {
		weekStart := start.AddDate(0, 0, 7*w)

		if weekStart.Year() != year {
			continue
		}

		month := int(weekStart.Month())
		if month != currentMonth {
			pos := 4 + w*2
			if pos+3 <= len(header) {
				copy(header[pos:pos+3], weekStart.Format("Jan"))
			}
			currentMonth = month
		}
	}

	return dimStyle.Render(string(header)) + "\n"
}

func findMax(activity map[string]int) int {
	maxCount := 1
	for _, count := range activity {
		if count > maxCount {
			maxCount = count
		}
	}
	return maxCount
}

func countToLevel(count, maxCount int) int {
	if count == 0 {
		return 0
	}
	ratio := float64(count) / float64(maxCount)
	switch {
	case ratio <= 0.25:
		return 1
	case ratio <= 0.5:
		return 2
	case ratio <= 0.75:
		return 3
	default:
		return 4
	}
}
