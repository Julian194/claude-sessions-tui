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
	ActionOpen
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
	BinPath  string
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

	// Read cache early to get session count for header
	entries, _ := cache.Read(cacheFile)

	// Generate random port for fzf listen
	rand.Seed(time.Now().UnixNano())
	port := 10000 + rand.Intn(50000)

	keybinds := "enter=resume  ctrl-o=export  ctrl-y=copy-md  ctrl-e=open  ctrl-b=branch  ctrl-r=refresh  ctrl-a=activity"
	sessionCount := len(entries)
	header := fmt.Sprintf("[%d sessions] %s", sessionCount, keybinds)
	loadingHeader := fmt.Sprintf("[Loading...] %s", keybinds)
	exportedHeader := fmt.Sprintf("[Exported!] %s", keybinds)
	copiedHeader := fmt.Sprintf("[Copied to clipboard!] %s", keybinds)
	openedHeader := fmt.Sprintf("[Opened in VS Code!] %s", keybinds)

	previewCmd := fmt.Sprintf("%s preview {1}", cfg.BinPath)
	activityCmd := fmt.Sprintf("%s activity-preview", cfg.BinPath)

	activityToggle := fmt.Sprintf(
		`sh -c 'if [ "$FZF_PREVIEW_LABEL" = " Activity " ]; then printf "change-preview(%s)+change-preview-label()"; else printf "change-preview(%s)+change-preview-label( Activity )"; fi'`,
		previewCmd, activityCmd,
	)
	rebuildCmd := fmt.Sprintf("%s rebuild", cfg.BinPath)

	rebuildWithCount := fmt.Sprintf(
		`sh -c '%s > /tmp/fzf_rebuild_$$ && count=$(grep -cv "^---HEADER---" /tmp/fzf_rebuild_$$); cat /tmp/fzf_rebuild_$$; rm -f /tmp/fzf_rebuild_$$; curl -s "http://localhost:%d" -d "change-header([${count} sessions] %s)"'`,
		rebuildCmd, port, keybinds,
	)

	resetCmd := fmt.Sprintf("%s reset-header %d '%s'", cfg.BinPath, port, header)
	exportCmd := fmt.Sprintf("%s export {1} && %s &", cfg.BinPath, resetCmd)
	copyMDCmd := fmt.Sprintf("%s copy-md {1} && %s", cfg.BinPath, resetCmd)
	openCmd := fmt.Sprintf("%s open {1} && %s &", cfg.BinPath, resetCmd)

	args := []string{
		"--delimiter=\t",
		"--with-nth=2,3,4",
		"--ansi",
		"--no-sort",
		"--no-separator",
		"--no-scrollbar",
		"--info=inline-right",
		"--prompt=> ",
		"--border=rounded",
		fmt.Sprintf("--preview=%s", previewCmd),
		"--preview-window=right:50%:wrap:border-left",
		fmt.Sprintf("--header=%s", loadingHeader),
		fmt.Sprintf("--listen=localhost:%d", port),
		fmt.Sprintf("--bind=ctrl-r:reload(%s)", rebuildWithCount),
		fmt.Sprintf("--bind=ctrl-o:execute-silent(%s)+change-header(%s)", exportCmd, exportedHeader),
		fmt.Sprintf("--bind=ctrl-y:execute-silent(%s)+change-header(%s)", copyMDCmd, copiedHeader),
		fmt.Sprintf("--bind=ctrl-e:execute-silent(%s)+change-header(%s)", openCmd, openedHeader),
		fmt.Sprintf("--bind=ctrl-a:transform:%s", activityToggle),
		"--expect=enter,ctrl-b,ctrl-e",
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stderr = os.Stderr

	formatted := formatForDisplay(entries)
	cmd.Stdin = strings.NewReader(strings.Join(formatted, "\n"))

	// Background: incremental rebuild and reload fzf
	go func() {
		time.Sleep(100 * time.Millisecond) // Brief delay for fzf to start listening

		reloadURL := fmt.Sprintf("http://localhost:%d", port)

		// Always do incremental rebuild (fast - only processes new/modified files)
		newEntries, err := cache.BuildIncremental(cfg.Adapter, cacheFile, entries)
		if err == nil {
			newHeader := fmt.Sprintf("[%d sessions] %s", len(newEntries), keybinds)

			// Check if anything changed
			changed := len(newEntries) != len(entries)
			if !changed {
				// Quick check: compare first few entries
				for i := 0; i < len(newEntries) && i < 5; i++ {
					if newEntries[i].SessionID != entries[i].SessionID {
						changed = true
						break
					}
				}
			}

			if changed {
				cache.Write(cacheFile, newEntries)
				body := fmt.Sprintf("reload(%s)+change-header(%s)", rebuildCmd, newHeader)
				http.Post(reloadURL, "text/plain", strings.NewReader(body))
			} else {
				http.Post(reloadURL, "text/plain", strings.NewReader(fmt.Sprintf("change-header(%s)", newHeader)))
			}
		} else {
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
	case "ctrl-e":
		result.Action = ActionOpen
	default:
		result.Action = ActionResume
	}

	return result, nil
}

// Rebuild rebuilds the cache and outputs formatted data for fzf reload
func Rebuild(cfg Config, mainOnly bool) error {
	cacheFile := filepath.Join(cfg.CacheDir, "sessions-cache.tsv")

	// Read existing cache for incremental build
	existing, _ := cache.Read(cacheFile)

	// Use incremental build instead of full rebuild
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

	// Output formatted entries for fzf display
	formatted := formatForDisplay(entries)
	for _, line := range formatted {
		fmt.Println(line)
	}

	return nil
}

// formatForDisplay formats cache entries with date headers and child indicators
func formatForDisplay(entries []cache.Entry) []string {
	if len(entries) == 0 {
		return nil
	}

	cyan := "\033[0;36m"
	dim := "\033[2m"
	nc := "\033[0m"

	var result []string
	currentDate := ""

	for _, e := range entries {
		entryDate := e.Date.Format("2006-01-02")

		if entryDate != currentDate {
			if currentDate != "" {
				formatted := formatDateHeader(currentDate)
				header := fmt.Sprintf("---HEADER---\t%s%s ─────────────────────────%s\t\t\t0\t-\t-",
					cyan, formatted, nc)
				result = append(result, header)
			}
			currentDate = entryDate
		}

		isChild := e.ParentSID != "" && e.ParentSID != "-"

		var line string
		if isChild {
			line = fmt.Sprintf("%s\t%s↳ %s%s\t%s\t%s\t%d\t%s\t%s",
				e.SessionID,
				dim, e.Date.Format("15:04"), nc,
				e.Project,
				e.Summary,
				e.Date.Unix(),
				e.ParentSID,
				entryDate,
			)
		} else {
			line = fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%s",
				e.SessionID,
				e.Date.Format("15:04"),
				e.Project,
				e.Summary,
				e.Date.Unix(),
				e.ParentSID,
				entryDate,
			)
		}
		result = append(result, line)
	}

	if currentDate != "" {
		formatted := formatDateHeader(currentDate)
		header := fmt.Sprintf("---HEADER---\t%s%s ─────────────────────────%s\t\t\t0\t-\t-",
			cyan, formatted, nc)
		result = append(result, header)
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
	models, _ := adapter.GetModels(sid)
	if len(models) > 0 {
		fmt.Printf("🤖 %s\n", strings.Join(models, ", "))
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
