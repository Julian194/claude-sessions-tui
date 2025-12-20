package heatmap

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/cache"
	"golang.org/x/term"
)

// True color (24-bit RGB) matching GitHub's contribution graph
var levelColors = []string{
	"\033[38;2;22;27;34m",  // level 0: #161b22 (empty/background)
	"\033[38;2;14;68;41m",  // level 1: #0e4429 (dark green)
	"\033[38;2;0;109;50m",  // level 2: #006d32 (medium-dark)
	"\033[38;2;38;166;65m", // level 3: #26a641 (medium)
	"\033[38;2;57;211;83m", // level 4: #39d353 (bright green)
}

const reset = "\033[0m"
const dim = "\033[2m"
const block = "■"

func RenderFromCache(entries []cache.Entry, weeks int) string {
	activity := make(map[string]int)
	for _, e := range entries {
		dateKey := e.Date.Format("2006-01-02")
		activity[dateKey]++
	}

	if weeks <= 0 {
		weeks = calculateMaxWeeks()
	}

	return Render(activity, weeks)
}

func calculateMaxWeeks() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		width = 80
	}

	maxWeeks := (width - 6) / 2
	if maxWeeks > 52 {
		maxWeeks = 52
	}
	if maxWeeks < 12 {
		maxWeeks = 12
	}
	return maxWeeks
}

func Render(activity map[string]int, weeks int) string {
	var sb strings.Builder

	today := time.Now()
	year := today.Year()
	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, today.Location())
	dec31 := time.Date(year, 12, 31, 0, 0, 0, 0, today.Location())

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

	weeks = int(end.Sub(start).Hours()/24/7) + 1

	sb.WriteString(renderMonthHeader(start, weeks, year))

	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	maxCount := findMax(activity)

	for d := 0; d < 7; d++ {
		if d == 0 || d == 2 || d == 4 {
			sb.WriteString(fmt.Sprintf("%s ", days[d]))
		} else {
			sb.WriteString("    ")
		}

		for w := 0; w < weeks; w++ {
			date := start.AddDate(0, 0, 7*w+d)

			if date.Year() != year {
				sb.WriteString("  ")
				continue
			}

			if date.After(today) {
				sb.WriteString("  ")
				continue
			}

			dateKey := date.Format("2006-01-02")
			count := activity[dateKey]
			level := countToLevel(count, maxCount)

			sb.WriteString(levelColors[level])
			sb.WriteString(block + " ")
			sb.WriteString(reset)
		}
		sb.WriteString("\n")
	}

	total := 0
	for _, count := range activity {
		total += count
	}
	sb.WriteString(fmt.Sprintf("\n%s%d sessions in %d%s\n", dim, total, year, reset))

	sb.WriteString(fmt.Sprintf("\n%sLess%s ", dim, reset))
	for _, c := range levelColors {
		sb.WriteString(c + block + " " + reset)
	}
	sb.WriteString(fmt.Sprintf("%sMore%s", dim, reset))

	return sb.String()
}

func renderMonthHeader(start time.Time, weeks int, year int) string {
	header := make([]byte, 5+weeks*2)
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
			pos := 5 + w*2
			if pos+3 <= len(header) {
				copy(header[pos:pos+3], weekStart.Format("Jan"))
			}
			currentMonth = month
		}
	}

	return string(header) + "\n"
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
