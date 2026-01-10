package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

// Action represents the user's selected action
type Action int

const (
	ActionResume Action = iota
	ActionBranch
	ActionExport
	ActionCopyMD
	ActionCancel
)

// Result contains the selected session and action
type Result struct {
	SessionID string
	Action    Action
	WorkDir   string
}

// Config holds TUI configuration
type Config struct {
	Adapter  adapters.Adapter
	CacheDir string
	BinPath  string // Kept for backwards compatibility, not used in Charm TUI
}

// Run launches the Bubble Tea TUI and returns the user's selection
func Run(cfg Config) (*Result, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Create and run the Bubble Tea program
	m := NewModel(cfg.Adapter, cfg.CacheDir)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	// Extract result from final model
	if model, ok := finalModel.(Model); ok {
		if result := model.Result(); result != nil {
			return result, nil
		}
	}

	return &Result{Action: ActionCancel}, nil
}

// Rebuild rebuilds the cache and outputs formatted data
// Kept for backwards compatibility with CLI subcommands
func Rebuild(cfg Config, mainOnly bool) error {
	cacheFile := filepath.Join(cfg.CacheDir, "sessions-cache.tsv")

	// Read existing cache for incremental build
	existing, _ := cache.Read(cacheFile)

	// Use incremental build
	entries, err := cache.BuildIncremental(cfg.Adapter, cacheFile, existing)
	if err != nil {
		return err
	}

	if err := cache.Write(cacheFile, entries); err != nil {
		return err
	}

	if mainOnly {
		var filtered []cache.Entry
		for _, e := range entries {
			if e.ParentSID == "" || e.ParentSID == "-" {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// Output count for CLI feedback
	fmt.Printf("Rebuilt cache: %d sessions\n", len(entries))
	return nil
}

// Preview outputs the preview pane content for a session
// Kept for backwards compatibility with CLI subcommands
func Preview(adapter adapters.Adapter, sid string) error {
	info, err := adapter.GetSessionInfo(sid)
	if err != nil {
		return err
	}

	// Header
	fmt.Printf("ID: %s\n", info.ID)
	fmt.Printf("Project: %s\n", info.Project)
	fmt.Printf("Date: %s\n", info.Date.Format("2006-01-02 15:04"))
	if info.Branch != "" {
		fmt.Printf("Branch: %s\n", info.Branch)
	}
	models, _ := adapter.GetModels(sid)
	if len(models) > 0 {
		fmt.Printf("Models: %s\n", joinStrings(models, ", "))
	}
	fmt.Println()

	// Summaries
	summaries, _ := adapter.GetSummaries(sid)
	if len(summaries) > 0 {
		fmt.Println("--- Topics ---")
		for _, s := range summaries {
			fmt.Printf("* %s\n", s)
		}
		fmt.Println()
	}

	// Slash commands
	cmds, _ := adapter.GetSlashCommands(sid)
	if len(cmds) > 0 {
		fmt.Println("--- Slash Commands ---")
		for _, cmd := range cmds {
			fmt.Printf("  %s\n", cmd)
		}
		fmt.Println()
	}

	// Files
	files, _ := adapter.GetFilesTouched(sid)
	if len(files) > 0 {
		fmt.Println("--- Files ---")
		shown := files
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, f := range shown {
			rel := f
			if info.WorkDir != "" {
				if r, err := filepath.Rel(info.WorkDir, f); err == nil && len(r) < len(f) && r[0] != '.' {
					rel = r
				}
			}
			fmt.Printf("* %s\n", rel)
		}
		if len(files) > 10 {
			fmt.Printf("  ... and %d more\n", len(files)-10)
		}
		fmt.Println()
	}

	// Stats
	stats, err := adapter.GetStats(sid)
	if err == nil {
		fmt.Println("--- Stats ---")
		fmt.Printf("Messages: %d user, %d assistant\n", stats.UserMessages, stats.AssistantMessages)
		fmt.Printf("Tokens: %d in, %d out", stats.InputTokens, stats.OutputTokens)
		if stats.CacheRead > 0 || stats.CacheWrite > 0 {
			fmt.Printf(", %d cache", stats.CacheRead+stats.CacheWrite)
		}
		fmt.Println()
		fmt.Printf("Cost: $%.4f\n", stats.Cost)
		fmt.Println()
	}

	// First message (if no summaries)
	if len(summaries) == 0 {
		msg, _ := adapter.GetFirstMessage(sid)
		if msg != "" {
			fmt.Println("--- First Message ---")
			fmt.Println(msg)
		}
	}

	return nil
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}
