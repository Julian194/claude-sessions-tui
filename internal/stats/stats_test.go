package stats

import (
	"strings"
	"testing"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

func sampleStats() *adapters.Stats {
	return &adapters.Stats{
		UserMessages:      5,
		AssistantMessages: 10,
		InputTokens:       15000,
		OutputTokens:      3000,
		CacheRead:         5000,
		CacheWrite:        2000,
		Cost:              0.0825,
		ToolCalls: map[string]int{
			"Read":  15,
			"Edit":  8,
			"Bash":  3,
			"Write": 2,
		},
	}
}

func TestFormat(t *testing.T) {
	s := sampleStats()
	output := Format(s)

	// Check headers
	if !strings.Contains(output, "Session Statistics") {
		t.Error("Format should contain title")
	}
	if !strings.Contains(output, "Messages") {
		t.Error("Format should contain Messages section")
	}
	if !strings.Contains(output, "Tokens") {
		t.Error("Format should contain Tokens section")
	}
	if !strings.Contains(output, "Cost") {
		t.Error("Format should contain Cost section")
	}
	if !strings.Contains(output, "Tool Calls") {
		t.Error("Format should contain Tool Calls section")
	}
}

func TestFormat_MessageCounts(t *testing.T) {
	s := sampleStats()
	output := Format(s)

	if !strings.Contains(output, "User:      5") {
		t.Error("Format should show user message count")
	}
	if !strings.Contains(output, "Assistant: 10") {
		t.Error("Format should show assistant message count")
	}
	if !strings.Contains(output, "Total:     15") {
		t.Error("Format should show total message count")
	}
}

func TestFormat_TokenCounts(t *testing.T) {
	s := sampleStats()
	output := Format(s)

	if !strings.Contains(output, "Input:       15,000") {
		t.Error("Format should show input tokens with formatting")
	}
	if !strings.Contains(output, "Output:      3,000") {
		t.Error("Format should show output tokens with formatting")
	}
}

func TestFormat_Cost(t *testing.T) {
	s := sampleStats()
	output := Format(s)

	if !strings.Contains(output, "$0.0825") {
		t.Error("Format should show cost")
	}
}

func TestFormat_ToolCalls(t *testing.T) {
	s := sampleStats()
	output := Format(s)

	// Should be sorted by count descending
	readIdx := strings.Index(output, "Read:")
	editIdx := strings.Index(output, "Edit:")
	bashIdx := strings.Index(output, "Bash:")

	if readIdx == -1 || editIdx == -1 || bashIdx == -1 {
		t.Error("Format should contain all tool names")
	}

	if readIdx > editIdx {
		t.Error("Read should appear before Edit (sorted by count)")
	}
	if editIdx > bashIdx {
		t.Error("Edit should appear before Bash (sorted by count)")
	}
}

func TestFormat_NoToolCalls(t *testing.T) {
	s := &adapters.Stats{
		UserMessages:      1,
		AssistantMessages: 1,
		ToolCalls:         map[string]int{},
	}
	output := Format(s)

	if strings.Contains(output, "Tool Calls") {
		t.Error("Format should not show Tool Calls section when empty")
	}
}

func TestFormatCompact(t *testing.T) {
	s := sampleStats()
	output := FormatCompact(s)

	if !strings.Contains(output, "15 msgs") {
		t.Error("FormatCompact should show message count")
	}
	if !strings.Contains(output, "18,000 tokens") {
		t.Error("FormatCompact should show token count")
	}
	if !strings.Contains(output, "$0.0825") {
		t.Error("FormatCompact should show cost")
	}
}

func TestFormatTokens(t *testing.T) {
	output := FormatTokens(15000, 3000, 5000, 2000)

	if !strings.Contains(output, "25,000 total") {
		t.Error("FormatTokens should show total")
	}
	if !strings.Contains(output, "15,000 in") {
		t.Error("FormatTokens should show input")
	}
	if !strings.Contains(output, "3,000 out") {
		t.Error("FormatTokens should show output")
	}
	if !strings.Contains(output, "7,000 cache") {
		t.Error("FormatTokens should show cache")
	}
}

func TestCalculateCost(t *testing.T) {
	// Test with known values
	// Input: 1M tokens * $3/1M = $3
	// Output: 1M tokens * $15/1M = $15
	// Cache read: 1M tokens * $0.30/1M = $0.30
	// Cache write: 1M tokens * $3.75/1M = $3.75
	// Total = $22.05

	cost := CalculateCost(1_000_000, 1_000_000, 1_000_000, 1_000_000)
	expected := 3.0 + 15.0 + 0.30 + 3.75

	if cost != expected {
		t.Errorf("CalculateCost() = %f, want %f", cost, expected)
	}
}

func TestCalculateCost_Small(t *testing.T) {
	// Test with smaller values
	// Input: 10000 tokens * $3/1M = $0.03
	// Output: 5000 tokens * $15/1M = $0.075
	// Cache read: 2000 * $0.30/1M = $0.0006
	// Cache write: 1000 * $3.75/1M = $0.00375

	cost := CalculateCost(10000, 5000, 2000, 1000)
	expected := 0.03 + 0.075 + 0.0006 + 0.00375

	// Use approximate comparison due to floating point
	diff := cost - expected
	if diff < -0.0001 || diff > 0.0001 {
		t.Errorf("CalculateCost() = %f, want approximately %f", cost, expected)
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{100, "100"},
		{999, "999"},
		{1000, "1,000"},
		{10000, "10,000"},
		{100000, "100,000"},
		{1000000, "1,000,000"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		got := formatNumber(tt.input)
		if got != tt.want {
			t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
