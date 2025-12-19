package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
	"github.com/Julian194/claude-sessions-tui/internal/adapters/claude"
	"github.com/Julian194/claude-sessions-tui/internal/adapters/opencode"
	"github.com/Julian194/claude-sessions-tui/internal/export"
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
	// Get path to self for subcommands
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
	case tui.ActionExport:
		return runExport(adapter, result.SessionID)
	case tui.ActionCopyMD:
		return runCopyMD(adapter, result.SessionID)
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

	// Write to file
	filename := fmt.Sprintf("session-%s.html", sid[:8])
	if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
		return err
	}

	fmt.Printf("Exported to %s\n", filename)

	// Try to open in browser
	exec.Command("open", filename).Start()

	return nil
}

func runCopyMD(adapter adapters.Adapter, sid string) error {
	messages, err := adapter.ExportMessages(sid)
	if err != nil {
		return err
	}

	info, _ := adapter.GetSessionInfo(sid)
	models, _ := adapter.GetModels(sid)
	md := export.ToMarkdown(messages, info, models)

	// Copy to clipboard using pbcopy (macOS)
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(md)
	if err := cmd.Run(); err != nil {
		// Fallback: print to stdout
		fmt.Print(md)
		return nil
	}

	fmt.Println("Copied to clipboard!")
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
  help          Show this help message

Keyboard shortcuts in TUI:
  Enter     Resume selected session
  Ctrl-O    Export session to HTML
  Ctrl-Y    Copy session as markdown
  Ctrl-B    Branch session
  Ctrl-R    Refresh cache

Environment:
  SESSIONS_CACHE_DIR   Override cache directory
  CLAUDE_DIR           Override Claude data directory

`, binaryName, adapter.Name(), adapter.DataDir(), adapter.CacheDir(), binaryName)
}
