package tui

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	BinPath  string // Path to the sessions binary for subcommands
}

// Run launches the fzf TUI and returns the user's selection
func Run(cfg Config) (*Result, error) {
	cacheFile := filepath.Join(cfg.CacheDir, "sessions-cache.tsv")

	// Ensure cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		f, err := os.Create(cacheFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
		f.Close()
	}

	// Generate random port for fzf listen
	rand.Seed(time.Now().UnixNano())
	port := 10000 + rand.Intn(50000)

	header := "enter=resume | ctrl-o=export | ctrl-y=copy-md | ctrl-b=branch | ctrl-r=refresh"
	loadingHeader := "[Loading...] " + header

	// Build fzf command
	previewCmd := fmt.Sprintf("%s preview {1}", cfg.BinPath)
	rebuildCmd := fmt.Sprintf("%s rebuild", cfg.BinPath)

	args := []string{
		"--delimiter=\t",
		"--with-nth=2,3,4",
		"--ansi",
		fmt.Sprintf("--preview=%s", previewCmd),
		"--preview-window=right:50%:wrap",
		fmt.Sprintf("--header=%s", loadingHeader),
		fmt.Sprintf("--listen=localhost:%d", port),
		fmt.Sprintf("--bind=ctrl-r:reload(%s)+change-header(%s)", rebuildCmd, header),
		"--expect=enter,ctrl-b,ctrl-o,ctrl-y",
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stderr = os.Stderr

	// Read and format existing cache for immediate display
	entries, _ := cache.Read(cacheFile)
	formatted := formatForDisplay(entries)
	cmd.Stdin = strings.NewReader(strings.Join(formatted, "\n"))

	// Check if cache is fresh (less than 1 hour old)
	cacheInfo, _ := os.Stat(cacheFile)
	cacheFresh := cacheInfo != nil && time.Since(cacheInfo.ModTime()) < time.Hour

	// Background: rebuild cache and reload fzf (only if stale)
	go func() {
		time.Sleep(100 * time.Millisecond) // Brief delay for fzf to start listening

		if !cacheFresh || len(entries) == 0 {
			// Rebuild cache
			newEntries, err := cache.BuildFrom(cfg.Adapter)
			if err == nil {
				cache.Write(cacheFile, newEntries)
			}

			// Reload fzf with new data
			reloadURL := fmt.Sprintf("http://localhost:%d", port)
			body := fmt.Sprintf("reload(%s)+change-header(%s)", rebuildCmd, header)
			http.Post(reloadURL, "text/plain", strings.NewReader(body))
		} else {
			// Just update header (cache is fresh)
			reloadURL := fmt.Sprintf("http://localhost:%d", port)
			http.Post(reloadURL, "text/plain", strings.NewReader(fmt.Sprintf("change-header(%s)", header)))
		}
	}()

	// Run fzf
	output, err := cmd.Output()
	if err != nil {
		// fzf exits with 130 on Ctrl-C, 1 on no match
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 || exitErr.ExitCode() == 1 {
				return &Result{Action: ActionCancel}, nil
			}
		}
		return nil, fmt.Errorf("fzf failed: %w", err)
	}

	return parseResult(output, cfg.Adapter)
}

// parseResult extracts the action and session from fzf output
func parseResult(output []byte, adapter adapters.Adapter) (*Result, error) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return &Result{Action: ActionCancel}, nil
	}

	key := lines[0]
	line := strings.Join(lines[1:], "\n")

	// Parse TSV line to get session ID
	fields := strings.Split(line, "\t")
	if len(fields) == 0 || fields[0] == "" {
		return &Result{Action: ActionCancel}, nil
	}

	sid := fields[0]

	// Skip date headers
	if sid == "---HEADER---" {
		return &Result{Action: ActionCancel}, nil
	}

	// Get workdir for the session
	workDir := ""
	info, err := adapter.GetSessionInfo(sid)
	if err == nil {
		workDir = info.WorkDir
	}

	result := &Result{
		SessionID: sid,
		WorkDir:   workDir,
	}

	switch key {
	case "ctrl-b":
		result.Action = ActionBranch
	case "ctrl-o":
		result.Action = ActionExport
	case "ctrl-y":
		result.Action = ActionCopyMD
	default:
		result.Action = ActionResume
	}

	return result, nil
}

// Rebuild rebuilds the cache and outputs formatted data for fzf reload
func Rebuild(cfg Config) error {
	cacheFile := filepath.Join(cfg.CacheDir, "sessions-cache.tsv")

	entries, err := cache.BuildFrom(cfg.Adapter)
	if err != nil {
		return err
	}

	if err := cache.Write(cacheFile, entries); err != nil {
		return err
	}

	// Output formatted entries for fzf display
	formatted := formatForDisplay(entries)
	for _, line := range formatted {
		fmt.Println(line)
	}

	return nil
}

// formatForDisplay formats cache entries with date headers and branch grouping
func formatForDisplay(entries []cache.Entry) []string {
	if len(entries) == 0 {
		return nil
	}

	// ANSI colors
	cyan := "\033[0;36m"
	yellow := "\033[0;33m"
	nc := "\033[0m"

	// Separate roots and branches
	var roots, branches []cache.Entry
	for _, e := range entries {
		if e.ParentSID == "" || e.ParentSID == "-" {
			roots = append(roots, e)
		} else {
			branches = append(branches, e)
		}
	}

	// Build parent ID set
	rootIDs := make(map[string]bool)
	for _, r := range roots {
		rootIDs[r.SessionID] = true
	}

	var result []string
	currentDate := ""

	for _, root := range roots {
		// Check if we need a new date header
		rootDate := root.Date.Format("2006-01-02")
		if rootDate != currentDate {
			// Output date header (after the sessions that belong to it)
			if currentDate != "" {
				formatted := formatDateHeader(currentDate)
				header := fmt.Sprintf("---HEADER---\t%s%s ─────────────────────────%s\t-\t-\t0\t-\t-",
					cyan, formatted, nc)
				result = append(result, header)
			}
			currentDate = rootDate
		}

		// Output root session
		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%s",
			root.SessionID,
			root.Date.Format("15:04"),
			root.Project,
			root.Summary,
			root.Date.Unix(),
			root.ParentSID,
			rootDate,
		)
		result = append(result, line)

		// Output branches for this root
		for _, b := range branches {
			if b.ParentSID == root.SessionID {
				branchLine := fmt.Sprintf("%s\t%s  └─ %s%s\t%s\t%s\t%d\t%s\t%s",
					b.SessionID,
					yellow, b.Date.Format("15:04"), nc,
					b.Project,
					b.Summary,
					b.Date.Unix(),
					b.ParentSID,
					b.Date.Format("2006-01-02"),
				)
				result = append(result, branchLine)
			}
		}
	}

	// Final date header
	if currentDate != "" {
		formatted := formatDateHeader(currentDate)
		header := fmt.Sprintf("---HEADER---\t%s%s ─────────────────────────%s\t-\t-\t0\t-\t-",
			cyan, formatted, nc)
		result = append(result, header)
	}

	// Output orphaned branches
	for _, b := range branches {
		if !rootIDs[b.ParentSID] {
			branchLine := fmt.Sprintf("%s\t%s  └─ %s%s\t%s\t%s\t%d\t%s\t%s",
				b.SessionID,
				yellow, b.Date.Format("15:04"), nc,
				b.Project,
				b.Summary,
				b.Date.Unix(),
				b.ParentSID,
				b.Date.Format("2006-01-02"),
			)
			result = append(result, branchLine)
		}
	}

	return result
}

func formatDateHeader(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("Monday, January 02, 2006")
}

// Preview outputs the preview pane content for a session
func Preview(adapter adapters.Adapter, sid string) error {
	info, err := adapter.GetSessionInfo(sid)
	if err != nil {
		return err
	}

	// Header
	fmt.Printf("🔑 %s\n", info.ID)
	fmt.Printf("📁 %s\n", info.Project)
	fmt.Printf("📅 %s\n", info.Date.Format("2006-01-02 15:04"))
	if info.Branch != "" {
		fmt.Printf("🌿 %s\n", info.Branch)
	}
	fmt.Println()

	// Summaries
	summaries, _ := adapter.GetSummaries(sid)
	if len(summaries) > 0 {
		fmt.Println("━━━ Topics ━━━")
		for _, s := range summaries {
			fmt.Printf("• %s\n", s)
		}
		fmt.Println()
	}

	// Slash commands
	cmds, _ := adapter.GetSlashCommands(sid)
	if len(cmds) > 0 {
		fmt.Println("━━━ Slash Commands ━━━")
		for _, cmd := range cmds {
			fmt.Printf("  %s\n", cmd)
		}
		fmt.Println()
	}

	// Files
	files, _ := adapter.GetFilesTouched(sid)
	if len(files) > 0 {
		fmt.Println("━━━ Files ━━━")
		shown := files
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, f := range shown {
			rel := f
			if info.WorkDir != "" {
				if r, err := filepath.Rel(info.WorkDir, f); err == nil && !strings.HasPrefix(r, "..") {
					rel = r
				}
			}
			fmt.Printf("• %s\n", rel)
		}
		if len(files) > 10 {
			fmt.Printf("  ... and %d more\n", len(files)-10)
		}
		fmt.Println()
	}

	// Stats (use claude-sessions-stats style output)
	stats, err := adapter.GetStats(sid)
	if err == nil {
		fmt.Println("━━━ Stats ━━━")
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
			fmt.Println("━━━ First Message ━━━")
			fmt.Println(msg)
		}
	}

	return nil
}

// ReadCache reads the cache file
func ReadCache(cacheFile string) ([]cache.Entry, error) {
	f, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []cache.Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Parse TSV line
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 5 {
			continue
		}
		// Skip header lines
		if fields[0] == "---HEADER---" {
			continue
		}
		entries = append(entries, cache.Entry{
			SessionID: fields[0],
			// Date parsing would go here
			Project: fields[2],
			Summary: fields[3],
		})
	}
	return entries, scanner.Err()
}
