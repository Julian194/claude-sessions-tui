package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
	"github.com/Julian194/claude-sessions-tui/internal/cache"
)

// Model is the main Bubble Tea model
type Model struct {
	// Components
	list    list.Model
	preview viewport.Model

	// State
	sessions     []cache.Entry
	pins         *Pins
	adapter      adapters.Adapter
	cacheDir     string
	activePane   string // "list" or "preview"
	showActivity bool

	// Layout
	width, height int

	// Status
	loading bool
	message string

	// Keys
	keys KeyMap

	// Result (set when user selects an action)
	result *Result
	done   bool
}

// SessionItem implements list.Item for cache.Entry
type SessionItem struct {
	entry    cache.Entry
	isPinned bool
}

func (s SessionItem) Title() string {
	prefix := ""
	if s.isPinned {
		prefix = "* "
	}
	if s.entry.ParentSID != "" && s.entry.ParentSID != "-" {
		prefix += "  "
	}
	return prefix + s.entry.Date.Format("15:04") + " " + s.entry.Project
}

func (s SessionItem) Description() string {
	return s.entry.Summary
}

func (s SessionItem) FilterValue() string {
	return s.entry.Project + " " + s.entry.Summary + " " + s.entry.SessionID
}

// NewModel creates a new TUI model
func NewModel(adapter adapters.Adapter, cacheDir string) Model {
	// Load pins
	pins := NewPins(cacheDir)
	pins.Load()

	// Create list with default delegate
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle
	delegate.Styles.NormalTitle = normalItemStyle
	delegate.Styles.NormalDesc = dimItemStyle

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Sessions"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()

	// Create preview viewport
	vp := viewport.New(0, 0)
	vp.SetContent("Select a session to preview")

	return Model{
		list:       l,
		preview:    vp,
		pins:       pins,
		adapter:    adapter,
		cacheDir:   cacheDir,
		activePane: "list",
		keys:       DefaultKeyMap(),
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return m.loadSessions()
}

// loadSessionsMsg is sent when sessions are loaded
type loadSessionsMsg struct {
	entries []cache.Entry
	err     error
}

func (m Model) loadSessions() tea.Cmd {
	return func() tea.Msg {
		cacheFile := filepath.Join(m.cacheDir, "sessions-cache.tsv")
		entries, err := cache.Read(cacheFile)
		if err != nil {
			// Try building cache
			entries, err = cache.BuildIncremental(m.adapter, cacheFile, nil)
			if err == nil {
				cache.Write(cacheFile, entries)
			}
		}
		return loadSessionsMsg{entries: entries, err: err}
	}
}

// refreshCacheMsg is sent when cache is refreshed
type refreshCacheMsg struct {
	entries []cache.Entry
	err     error
}

func (m Model) refreshCache() tea.Cmd {
	return func() tea.Msg {
		cacheFile := filepath.Join(m.cacheDir, "sessions-cache.tsv")
		existing, _ := cache.Read(cacheFile)
		entries, err := cache.BuildIncremental(m.adapter, cacheFile, existing)
		if err == nil {
			cache.Write(cacheFile, entries)
		}
		return refreshCacheMsg{entries: entries, err: err}
	}
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case loadSessionsMsg:
		if msg.err != nil {
			m.message = "Error loading sessions: " + msg.err.Error()
		} else {
			m.sessions = msg.entries
			m.updateListItems()
			m.message = fmt.Sprintf("%d sessions", len(msg.entries))
		}
		m.loading = false

	case refreshCacheMsg:
		if msg.err != nil {
			m.message = "Error refreshing: " + msg.err.Error()
		} else {
			m.sessions = msg.entries
			m.updateListItems()
			m.message = fmt.Sprintf("Refreshed: %d sessions", len(msg.entries))
		}
		m.loading = false

	case tea.KeyMsg:
		// Handle quit first
		if key.Matches(msg, m.keys.Quit) {
			m.done = true
			return m, tea.Quit
		}

		// Don't process other keys while filtering
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Tab):
			if m.activePane == "list" {
				m.activePane = "preview"
			} else {
				m.activePane = "list"
			}

		case key.Matches(msg, m.keys.Select):
			if item, ok := m.list.SelectedItem().(SessionItem); ok {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionResume,
				}
				// Get workdir
				if info, err := m.adapter.GetSessionInfo(item.entry.SessionID); err == nil {
					m.result.WorkDir = info.WorkDir
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Export):
			if item, ok := m.list.SelectedItem().(SessionItem); ok {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionExport,
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.CopyMD):
			if item, ok := m.list.SelectedItem().(SessionItem); ok {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionCopyMD,
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Branch):
			if item, ok := m.list.SelectedItem().(SessionItem); ok {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionBranch,
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Pin):
			if item, ok := m.list.SelectedItem().(SessionItem); ok {
				pinned := m.pins.Toggle(item.entry.SessionID)
				m.pins.Save()
				if pinned {
					m.message = "Pinned session"
				} else {
					m.message = "Unpinned session"
				}
				m.updateListItems()
			}

		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.message = "Refreshing..."
			return m, m.refreshCache()

		case key.Matches(msg, m.keys.ToggleActivity):
			m.showActivity = !m.showActivity
			m.updatePreview()
		}
	}

	// Route updates to active pane
	if m.activePane == "list" {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Update preview when selection changes
		if _, ok := msg.(tea.KeyMsg); ok {
			m.updatePreview()
		}
	} else {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate pane widths
	listWidth := m.width / 2
	previewWidth := m.width - listWidth - 1

	// Apply borders based on active pane
	var listPane, previewPane string

	listContent := m.list.View()
	previewContent := m.preview.View()

	if m.activePane == "list" {
		listPane = activeBorderStyle.
			Width(listWidth - 2).
			Height(m.height - 4).
			Render(listContent)
		previewPane = inactiveBorderStyle.
			Width(previewWidth - 2).
			Height(m.height - 4).
			Render(previewContent)
	} else {
		listPane = inactiveBorderStyle.
			Width(listWidth - 2).
			Height(m.height - 4).
			Render(listContent)
		previewPane = activeBorderStyle.
			Width(previewWidth - 2).
			Height(m.height - 4).
			Render(previewContent)
	}

	// Join panes horizontally
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)

	// Status bar
	help := helpStyle.Render("enter:resume  ctrl+o:export  ctrl+y:copy  ctrl+b:branch  p:pin  ctrl+r:refresh  tab:switch  q:quit")
	status := statusBarStyle.Render(m.message)
	statusBar := lipgloss.JoinHorizontal(lipgloss.Left, status, "  ", help)

	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

// updateLayout recalculates component sizes
func (m *Model) updateLayout() {
	listWidth := m.width / 2
	previewWidth := m.width - listWidth - 3

	m.list.SetSize(listWidth-4, m.height-6)
	m.preview.Width = previewWidth - 2
	m.preview.Height = m.height - 6
}

// updateListItems refreshes the list with sorted sessions
func (m *Model) updateListItems() {
	// Sort: pinned first, then by date
	sorted := make([]cache.Entry, len(m.sessions))
	copy(sorted, m.sessions)

	sort.SliceStable(sorted, func(i, j int) bool {
		iPinned := m.pins.IsPinned(sorted[i].SessionID)
		jPinned := m.pins.IsPinned(sorted[j].SessionID)
		if iPinned != jPinned {
			return iPinned
		}
		return sorted[i].Date.After(sorted[j].Date)
	})

	items := make([]list.Item, len(sorted))
	for i, entry := range sorted {
		items[i] = SessionItem{
			entry:    entry,
			isPinned: m.pins.IsPinned(entry.SessionID),
		}
	}
	m.list.SetItems(items)
}

// updatePreview updates the preview pane content
func (m *Model) updatePreview() {
	item, ok := m.list.SelectedItem().(SessionItem)
	if !ok {
		m.preview.SetContent("No session selected")
		return
	}

	if m.showActivity {
		m.preview.SetContent("Activity heatmap (TODO)")
		return
	}

	// Generate preview content
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("ID: %s\n", item.entry.SessionID))
	b.WriteString(fmt.Sprintf("Project: %s\n", item.entry.Project))
	b.WriteString(fmt.Sprintf("Date: %s\n", item.entry.Date.Format("2006-01-02 15:04")))

	// Get additional info from adapter
	if info, err := m.adapter.GetSessionInfo(item.entry.SessionID); err == nil {
		if info.Branch != "" {
			b.WriteString(fmt.Sprintf("Branch: %s\n", info.Branch))
		}
	}

	// Models
	if models, err := m.adapter.GetModels(item.entry.SessionID); err == nil && len(models) > 0 {
		b.WriteString(fmt.Sprintf("Models: %s\n", strings.Join(models, ", ")))
	}

	b.WriteString("\n")

	// Summaries
	if summaries, err := m.adapter.GetSummaries(item.entry.SessionID); err == nil && len(summaries) > 0 {
		b.WriteString("--- Topics ---\n")
		for _, s := range summaries {
			b.WriteString(fmt.Sprintf("* %s\n", s))
		}
		b.WriteString("\n")
	}

	// Slash commands
	if cmds, err := m.adapter.GetSlashCommands(item.entry.SessionID); err == nil && len(cmds) > 0 {
		b.WriteString("--- Slash Commands ---\n")
		for _, cmd := range cmds {
			b.WriteString(fmt.Sprintf("  %s\n", cmd))
		}
		b.WriteString("\n")
	}

	// Files
	if files, err := m.adapter.GetFilesTouched(item.entry.SessionID); err == nil && len(files) > 0 {
		b.WriteString("--- Files ---\n")
		shown := files
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, f := range shown {
			b.WriteString(fmt.Sprintf("* %s\n", f))
		}
		if len(files) > 10 {
			b.WriteString(fmt.Sprintf("  ... and %d more\n", len(files)-10))
		}
		b.WriteString("\n")
	}

	// Stats
	if stats, err := m.adapter.GetStats(item.entry.SessionID); err == nil {
		b.WriteString("--- Stats ---\n")
		b.WriteString(fmt.Sprintf("Messages: %d user, %d assistant\n", stats.UserMessages, stats.AssistantMessages))
		b.WriteString(fmt.Sprintf("Tokens: %d in, %d out", stats.InputTokens, stats.OutputTokens))
		if stats.CacheRead > 0 || stats.CacheWrite > 0 {
			b.WriteString(fmt.Sprintf(", %d cache", stats.CacheRead+stats.CacheWrite))
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Cost: $%.4f\n", stats.Cost))
		b.WriteString("\n")
	}

	// First message (if no summaries)
	if summaries, _ := m.adapter.GetSummaries(item.entry.SessionID); len(summaries) == 0 {
		if msg, err := m.adapter.GetFirstMessage(item.entry.SessionID); err == nil && msg != "" {
			b.WriteString("--- First Message ---\n")
			b.WriteString(msg)
			b.WriteString("\n")
		}
	}

	m.preview.SetContent(b.String())
	m.preview.GotoTop()
}

// Result returns the user's selection (nil if cancelled)
func (m Model) Result() *Result {
	if m.done && m.result != nil {
		return m.result
	}
	return nil
}
