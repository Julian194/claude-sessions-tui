package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
	"github.com/Julian194/claude-sessions-tui/internal/adapters/claude"
	"github.com/Julian194/claude-sessions-tui/internal/adapters/opencode"
	"github.com/Julian194/claude-sessions-tui/internal/cache"
	"github.com/Julian194/claude-sessions-tui/internal/export"
	"github.com/Julian194/claude-sessions-tui/internal/heatmap"
	"github.com/Julian194/claude-sessions-tui/internal/stats"
	"github.com/Julian194/claude-sessions-tui/internal/tui"
)

func main() {
	// Detect adapter from binary name
	binaryName := filepath.Base(os.Args[0])
	adapter := getAdapter(binaryName)

	// Get cache directory
	cacheDir := getCacheDir(adapter)

	// Route subcommand
	cmd := ""
	args := os.Args[1:]
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	var err error
	switch cmd {
	case "", "tui":
		err = runTUI(adapter, cacheDir)
	case "rebuild":
		mainOnly := len(args) > 0 && args[0] == "--main-only"
		err = runRebuild(adapter, cacheDir, mainOnly)
	case "preview":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: sessions preview <session-id>")
			os.Exit(1)
		}
		err = runPreview(adapter, args[0])
	case "stats":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: sessions stats <session-id>")
			os.Exit(1)
		}
		err = runStats(adapter, args[0])
	case "export":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: sessions export <session-id>")
			os.Exit(1)
		}
		err = runExport(adapter, args[0])
	case "copy-md":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: sessions copy-md <session-id>")
			os.Exit(1)
		}
		err = runCopyMD(adapter, args[0])
	case "open":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: sessions open <session-id>")
			os.Exit(1)
		}
		err = runOpen(adapter, args[0])
	case "activity":
		err = runActivity(adapter, cacheDir)
	case "activity-preview":
		err = runActivityPreview(adapter, cacheDir)
	case "reset-header":
		if len(args) < 2 {
			os.Exit(1)
		}
		runResetHeader(args[0], args[1])
	case "help", "--help", "-h":
		printUsage(binaryName, adapter)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage(binaryName, adapter)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getAdapter(binaryName string) adapters.Adapter {
	if strings.Contains(binaryName, "opencode") {
		return opencode.New("")
	}
	return claude.New("")
}

func getCacheDir(adapter adapters.Adapter) string {
	// Check environment variable override
	if dir := os.Getenv("SESSIONS_CACHE_DIR"); dir != "" {
		return dir
	}

	// Use adapter's cache dir
	return adapter.CacheDir()
}

func runTUI(adapter adapters.Adapter, cacheDir string) error {
	binPath, err := os.Executable()
	if err != nil {
		binPath = os.Args[0]
	}

	cfg := tui.Config{
		Adapter:  adapter,
		CacheDir: cacheDir,
		BinPath:  binPath,
	}

	result, err := tui.Run(cfg)
	if err != nil {
		return err
	}

	if result == nil || result.Action == tui.ActionCancel {
		return nil
	}

	switch result.Action {
	case tui.ActionResume:
		return resumeSession(adapter, result.SessionID, result.WorkDir)
	case tui.ActionBranch:
		return branchSession(adapter, result.SessionID, result.WorkDir)
	case tui.ActionOpen:
		return runOpen(adapter, result.SessionID)
	}

	return nil
}

func runRebuild(adapter adapters.Adapter, cacheDir string, mainOnly bool) error {
	cfg := tui.Config{
		Adapter:  adapter,
		CacheDir: cacheDir,
	}
	return tui.Rebuild(cfg, mainOnly)
}

func runPreview(adapter adapters.Adapter, sid string) error {
	return tui.Preview(adapter, sid)
}

func runStats(adapter adapters.Adapter, sid string) error {
	s, err := adapter.GetStats(sid)
	if err != nil {
		return err
	}
	fmt.Print(stats.Format(s))
	return nil
}

func runActivity(adapter adapters.Adapter, cacheDir string) error {
	cacheFile := filepath.Join(cacheDir, "sessions-cache.tsv")

	entries, err := cache.Read(cacheFile)
	if err != nil {
		entries, err = cache.BuildFrom(adapter)
		if err != nil {
			return err
		}
	}

	fmt.Println(heatmap.RenderFromCache(entries, 0))
	return nil
}

func runActivityPreview(adapter adapters.Adapter, cacheDir string) error {
	cacheFile := filepath.Join(cacheDir, "sessions-cache.tsv")

	entries, err := cache.Read(cacheFile)
	if err != nil {
		entries, err = cache.BuildFrom(adapter)
		if err != nil {
			return err
		}
	}

	fmt.Println("\n📊 Activity Heatmap")
	fmt.Println(heatmap.RenderFromCache(entries, 0))
	return nil
}

func runExport(adapter adapters.Adapter, sid string) error {
	messages, err := adapter.ExportMessages(sid)
	if err != nil {
		return err
	}

	info, err := adapter.GetSessionInfo(sid)
	if err != nil {
		return err
	}

	models, _ := adapter.GetModels(sid)
	html := export.ToHTML(messages, info, models)

	// Write to /tmp for reliable access
	shortID := sid
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	filename := fmt.Sprintf("/tmp/session-%s.html", shortID)
	if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
		return err
	}

	fmt.Printf("Exported to %s\n", filename)

	// Open in browser (cross-platform)
	switch {
	case fileExists("/usr/bin/open"): // macOS
		exec.Command("open", filename).Start()
	case commandExists("xdg-open"): // Linux
		exec.Command("xdg-open", filename).Start()
	case commandExists("wslview"): // WSL
		exec.Command("wslview", filename).Start()
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func runResetHeader(port, header string) {
	time.Sleep(1 * time.Second)
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:%s", port),
		"text/plain",
		strings.NewReader(fmt.Sprintf("change-header(%s)", header)),
	)
	if err == nil {
		resp.Body.Close()
	}
}

func runCopyMD(adapter adapters.Adapter, sid string) error {
	messages, err := adapter.ExportMessages(sid)
	if err != nil {
		return err
	}

	info, _ := adapter.GetSessionInfo(sid)
	models, _ := adapter.GetModels(sid)
	md := export.ToMarkdown(messages, info, models)

	var clipboardCmd []string
	switch {
	case commandExists("pbcopy"):
		clipboardCmd = []string{"pbcopy"}
	case commandExists("wl-copy"):
		clipboardCmd = []string{"wl-copy"}
	case commandExists("xclip"):
		clipboardCmd = []string{"xclip", "-selection", "clipboard"}
	case commandExists("xsel"):
		clipboardCmd = []string{"xsel", "--clipboard", "--input"}
	default:
		fmt.Println("No clipboard tool found (need pbcopy, wl-copy, xclip, or xsel)")
		fmt.Print(md)
		return nil
	}

	cmd := exec.Command(clipboardCmd[0], clipboardCmd[1:]...)
	cmd.Stdin = strings.NewReader(md)
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Clipboard copy failed:", err)
		fmt.Print(md)
		return nil
	}

	if commandExists("notify-send") {
		notifCmd := exec.Command("notify-send", "Claude Sessions", "Copied to clipboard!")
		notifCmd.Run()
	}
	fmt.Fprintln(os.Stderr, "Copied to clipboard!")
	return nil
}

func resumeSession(adapter adapters.Adapter, sid string, workDir string) error {
	resumeCmd := adapter.ResumeCmd(sid)
	parts := strings.Fields(resumeCmd)

	if adapter.Name() == "claude" {
		parts = append(parts, "--dangerously-skip-permissions")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if workDir != "" {
		if _, err := os.Stat(workDir); err == nil {
			cmd.Dir = workDir
		}
	}

	return cmd.Run()
}

func branchSession(adapter adapters.Adapter, sid string, workDir string) error {
	newSID, err := adapter.BranchSession(sid)
	if err != nil {
		return fmt.Errorf("branch failed: %w", err)
	}

	fmt.Printf("Branched session: %s\n", newSID)
	return resumeSession(adapter, newSID, workDir)
}

func runOpen(adapter adapters.Adapter, sid string) error {
	sessionPath := adapter.GetSessionFile(sid)
	if sessionPath == "" {
		return fmt.Errorf("session file not found: %s", sid)
	}

	if _, err := os.Stat(sessionPath); err != nil {
		return fmt.Errorf("cannot access session file: %w", err)
	}

	shortID := sid
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	var formattedPath string
	var err error

	if adapter.Name() == "claude" {
		formattedPath, err = formatJSONL(sessionPath, shortID)
	} else {
		formattedPath, err = formatJSON(sessionPath, shortID)
	}

	if err != nil {
		return fmt.Errorf("failed to format session file: %w", err)
	}

	cmd := exec.Command("code", formattedPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open in VS Code: %w", err)
	}

	fmt.Printf("Opening formatted session in VS Code: %s\n", formattedPath)
	return nil
}

func formatJSONL(inputPath string, shortID string) (string, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	var formatted []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			formatted = append(formatted, line)
			continue
		}

		pretty, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			formatted = append(formatted, line)
			continue
		}
		formatted = append(formatted, string(pretty))
	}

	outputPath := fmt.Sprintf("/tmp/session-%s-formatted.jsonl", shortID)
	if err := os.WriteFile(outputPath, []byte(strings.Join(formatted, "\n\n")), 0644); err != nil {
		return "", err
	}

	return outputPath, nil
}

func formatJSON(inputPath string, shortID string) (string, error) {
	sessionBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	var session map[string]interface{}
	if err := json.Unmarshal(sessionBytes, &session); err != nil {
		return "", err
	}

	sessionID, _ := session["id"].(string)
	if sessionID == "" {
		return "", fmt.Errorf("invalid session file: missing id")
	}

	sessionDir := filepath.Dir(inputPath)
	dataDir := filepath.Dir(filepath.Dir(sessionDir))

	messageDir := filepath.Join(dataDir, "message", sessionID)
	fmt.Fprintf(os.Stderr, "DEBUG: inputPath=%s\n", inputPath)
	fmt.Fprintf(os.Stderr, "DEBUG: sessionID=%s\n", sessionID)
	fmt.Fprintf(os.Stderr, "DEBUG: sessionDir=%s\n", sessionDir)
	fmt.Fprintf(os.Stderr, "DEBUG: dataDir=%s\n", dataDir)
	fmt.Fprintf(os.Stderr, "DEBUG: messageDir=%s\n", messageDir)

	var messages []interface{}
	if entries, err := os.ReadDir(messageDir); err == nil {
		fmt.Fprintf(os.Stderr, "DEBUG: found %d message files\n", len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			msgBytes, err := os.ReadFile(filepath.Join(messageDir, entry.Name()))
			if err != nil {
				continue
			}
			var msg map[string]interface{}
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}

			msgID, _ := msg["id"].(string)
			if msgID == "" {
				continue
			}

			partDir := filepath.Join(dataDir, "part", msgID)
			var parts []interface{}
			if pEntries, err := os.ReadDir(partDir); err == nil {
				for _, pEntry := range pEntries {
					if pEntry.IsDir() || !strings.HasSuffix(pEntry.Name(), ".json") {
						continue
					}
					partBytes, err := os.ReadFile(filepath.Join(partDir, pEntry.Name()))
					if err != nil {
						continue
					}
					var part map[string]interface{}
					if err := json.Unmarshal(partBytes, &part); err != nil {
						continue
					}
					parts = append(parts, part)
				}
			}
			msg["parts"] = parts
			messages = append(messages, msg)
		}
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG: error reading message dir: %v\n", err)
	}

	session["messages"] = messages

	output, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return "", err
	}

	outputPath := fmt.Sprintf("/tmp/session-%s-formatted.json", shortID)
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return "", err
	}

	return outputPath, nil
}

func printUsage(binaryName string, adapter adapters.Adapter) {
	fmt.Printf(`%s - browse and export AI coding sessions (v2.0.0)

Provider: %s
Data:     %s
Cache:    %s

Usage: %s [command] [arguments]

Commands:
  (default)     Launch interactive TUI
  rebuild       Rebuild the session cache
  preview <id>  Show preview for a session
  stats <id>    Show statistics for a session
  export <id>   Export session to HTML
  copy-md <id>  Copy session as markdown to clipboard
  open <id>     Open original session file in VS Code
  help          Show this help message

Keyboard shortcuts in TUI:
  Enter     Resume selected session
  Ctrl-O    Export session to HTML
  Ctrl-Y    Copy session as markdown
  Ctrl-E    Open original session file in VS Code
  Ctrl-B    Branch session
  Ctrl-R    Refresh cache

Environment:
  SESSIONS_CACHE_DIR   Override cache directory
  CLAUDE_DIR           Override Claude data directory

`, binaryName, adapter.Name(), adapter.DataDir(), adapter.CacheDir(), binaryName)
}
