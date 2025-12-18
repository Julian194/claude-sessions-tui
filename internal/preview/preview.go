package preview

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
	"github.com/Julian194/claude-sessions-tui/internal/stats"
)

// Format generates the preview pane content for a session
func Format(adapter adapters.Adapter, id string) (string, error) {
	info, err := adapter.GetSessionInfo(id)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	// Header info
	sb.WriteString(fmt.Sprintf("🔑 %s\n", info.ID))
	sb.WriteString(fmt.Sprintf("📁 %s\n", info.Project))
	sb.WriteString(fmt.Sprintf("📅 %s\n", info.Date.Format("2006-01-02 15:04")))
	if info.Branch != "" {
		sb.WriteString(fmt.Sprintf("🌿 %s\n", info.Branch))
	}
	sb.WriteString("\n")

	// Summaries (topics)
	summaries, _ := adapter.GetSummaries(id)
	if len(summaries) > 0 {
		sb.WriteString("━━━ Topics ━━━\n")
		for _, s := range summaries {
			sb.WriteString(fmt.Sprintf("• %s\n", s))
		}
		sb.WriteString("\n")
	}

	// Slash commands
	cmds, _ := adapter.GetSlashCommands(id)
	if len(cmds) > 0 {
		sb.WriteString("━━━ Slash Commands ━━━\n")
		for _, cmd := range cmds {
			sb.WriteString(fmt.Sprintf("  %s\n", cmd))
		}
		sb.WriteString("\n")
	}

	// Files touched (relative to cwd)
	files, _ := adapter.GetFilesTouched(id)
	if len(files) > 0 {
		sb.WriteString("━━━ Files ━━━\n")
		// Limit to 10 files
		shown := files
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, f := range shown {
			// Make relative to workdir if possible
			rel := f
			if info.WorkDir != "" {
				if r, err := filepath.Rel(info.WorkDir, f); err == nil && !strings.HasPrefix(r, "..") {
					rel = r
				}
			}
			sb.WriteString(fmt.Sprintf("• %s\n", rel))
		}
		if len(files) > 10 {
			sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(files)-10))
		}
		sb.WriteString("\n")
	}

	// Stats
	s, err := adapter.GetStats(id)
	if err == nil {
		sb.WriteString(stats.Format(s))
		sb.WriteString("\n")
	}

	// First message (fallback if no summaries)
	if len(summaries) == 0 {
		msg, _ := adapter.GetFirstMessage(id)
		if msg != "" {
			sb.WriteString("━━━ First Message ━━━\n")
			sb.WriteString(msg)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
