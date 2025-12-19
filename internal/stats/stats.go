package stats

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

// Format formats stats for display
func Format(s *adapters.Stats) string {
	var sb strings.Builder

	sb.WriteString("📊 Session Statistics\n")
	sb.WriteString("─────────────────────\n\n")

	// Messages
	sb.WriteString("💬 Messages\n")
	sb.WriteString(fmt.Sprintf("   User:      %d\n", s.UserMessages))
	sb.WriteString(fmt.Sprintf("   Assistant: %d\n", s.AssistantMessages))
	sb.WriteString(fmt.Sprintf("   Total:     %d\n\n", s.UserMessages+s.AssistantMessages))

	// Tokens
	sb.WriteString("🔤 Tokens\n")
	sb.WriteString(fmt.Sprintf("   Input:       %s\n", formatNumber(s.InputTokens)))
	sb.WriteString(fmt.Sprintf("   Output:      %s\n", formatNumber(s.OutputTokens)))
	sb.WriteString(fmt.Sprintf("   Cache Read:  %s\n", formatNumber(s.CacheRead)))
	sb.WriteString(fmt.Sprintf("   Cache Write: %s\n", formatNumber(s.CacheWrite)))
	sb.WriteString(fmt.Sprintf("   Total:       %s\n\n", formatNumber(s.InputTokens+s.OutputTokens+s.CacheRead+s.CacheWrite)))

	// Cost
	sb.WriteString("💰 Cost\n")
	sb.WriteString(fmt.Sprintf("   Estimated: $%.4f\n\n", s.Cost))

	// Tool calls
	if len(s.ToolCalls) > 0 {
		sb.WriteString("🔧 Tool Calls\n")

		// Sort by count descending
		type toolCount struct {
			name  string
			count int
		}
		var tools []toolCount
		for name, count := range s.ToolCalls {
			tools = append(tools, toolCount{name, count})
		}
		sort.Slice(tools, func(i, j int) bool {
			return tools[i].count > tools[j].count
		})

		for _, tc := range tools {
			sb.WriteString(fmt.Sprintf("   %-12s %d\n", tc.name+":", tc.count))
		}
	}

	return sb.String()
}

// FormatCompact formats stats in a compact single line
func FormatCompact(s *adapters.Stats) string {
	return fmt.Sprintf("%d msgs | %s tokens | $%.4f",
		s.UserMessages+s.AssistantMessages,
		formatNumber(s.InputTokens+s.OutputTokens),
		s.Cost,
	)
}

// FormatTokens formats token counts for display
func FormatTokens(input, output, cacheRead, cacheWrite int) string {
	total := input + output + cacheRead + cacheWrite
	return fmt.Sprintf("%s total (%s in, %s out, %s cache)",
		formatNumber(total),
		formatNumber(input),
		formatNumber(output),
		formatNumber(cacheRead+cacheWrite),
	)
}

// CalculateCost calculates the estimated cost based on token counts
func CalculateCost(input, output, cacheRead, cacheWrite int) float64 {
	// Sonnet 3.5 pricing (per 1M tokens)
	inputPrice := 3.0
	outputPrice := 15.0
	cacheReadPrice := 0.30
	cacheWritePrice := 3.75

	cost := float64(input) * inputPrice / 1_000_000
	cost += float64(output) * outputPrice / 1_000_000
	cost += float64(cacheRead) * cacheReadPrice / 1_000_000
	cost += float64(cacheWrite) * cacheWritePrice / 1_000_000

	return cost
}

// formatNumber formats a number with thousands separators
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	s := fmt.Sprintf("%d", n)
	var result strings.Builder

	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}

	return result.String()
}
