package tui

import (
	"fmt"

	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

// DateHeaderItem is a non-selectable date separator
type DateHeaderItem struct {
	date string
}

func (d DateHeaderItem) Title() string {
	return "─── " + d.date + " ───"
}

func (d DateHeaderItem) Description() string {
	return "" // Empty but still takes up space for consistent height
}

func (d DateHeaderItem) FilterValue() string {
	return "" // Not searchable
}

// SessionItem implements list.Item for cache.Entry
type SessionItem struct {
	entry      cache.Entry
	isPinned   bool
	isAgent    bool
	isParent   bool // Has children
	isExpanded bool // If parent, is it expanded?
	childCount int  // Number of children (for parents)
}

func (s SessionItem) Title() string {
	prefix := ""

	// Expand/collapse indicator for parents
	if s.isParent {
		if s.isExpanded {
			prefix = "▼ "
		} else {
			prefix = "▶ "
		}
	}

	if s.isPinned {
		prefix += "★ "
	}
	if s.isAgent {
		prefix = "  " // Indent children
	}

	title := prefix + s.entry.Date.Format("15:04") + " " + s.entry.Project

	// Show child count for collapsed parents
	if s.isParent && !s.isExpanded && s.childCount > 0 {
		title += fmt.Sprintf(" (%d)", s.childCount)
	}

	return title
}

func (s SessionItem) Description() string {
	return s.entry.Summary
}

func (s SessionItem) FilterValue() string {
	return s.entry.Project + " " + s.entry.Summary + " " + s.entry.SessionID
}
