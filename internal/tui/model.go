package tui

import (
	"fmt"
	"io"
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

// FilterMode represents the current filter type
type FilterMode int

const (
	FilterNone FilterMode = iota
	FilterProject
	FilterToday
	FilterWeek
	FilterHighCost
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
	filterMode   FilterMode
	filterValue  string // For project filter

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

// ListItem is the interface for items in our list (date headers or sessions)
type ListItem interface {
	list.Item
	IsHeader() bool
	IsAgent() bool
}

// DateHeader represents a date separator in the list
type DateHeader struct {
	date  string
	count int
}

func (d DateHeader) Title() string       { return d.date }
func (d DateHeader) Description() string { return fmt.Sprintf("%d sessions", d.count) }
func (d DateHeader) FilterValue() string { return "" } // Don't filter headers
func (d DateHeader) IsHeader() bool      { return true }
func (d DateHeader) IsAgent() bool       { return false }

// SessionItem implements list.Item for cache.Entry
type SessionItem struct {
	entry    cache.Entry
	isPinned bool
	isAgent  bool
	depth    int // 0 = root, 1 = agent child
}

func (s SessionItem) Title() string {
	prefix := ""
	if s.isPinned {
		prefix = "★ "
	}
	if s.depth > 0 {
		prefix += "  ↳ "
	}
	return prefix + s.entry.Date.Format("15:04") + " " + s.entry.Project
}

func (s SessionItem) Description() string {
	return s.entry.Summary
}

func (s SessionItem) FilterValue() string {
	return s.entry.Project + " " + s.entry.Summary + " " + s.entry.SessionID
}

func (s SessionItem) IsHeader() bool { return false }
func (s SessionItem) IsAgent() bool  { return s.isAgent }

// CustomDelegate handles rendering of both headers and session items
type CustomDelegate struct {
	list.DefaultDelegate
}

func NewCustomDelegate() CustomDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = selectedItemStyle
	d.Styles.SelectedDesc = selectedItemStyle
	d.Styles.NormalTitle = normalItemStyle
	d.Styles.NormalDesc = dimItemStyle
	return CustomDelegate{DefaultDelegate: d}
}

func (d CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if listItem, ok := item.(ListItem); ok && listItem.IsHeader() {
		// Render date header
		header, _ := item.(DateHeader)
		str := dateHeaderStyle.Render(fmt.Sprintf("── %s ──", header.date))
		fmt.Fprint(w, str)
		return
	}

	// Check if this is an agent session
	if listItem, ok := item.(ListItem); ok && listItem.IsAgent() {
		session := item.(SessionItem)
		title := session.Title()
		desc := session.Description()

		if index == m.Index() {
			title = selectedItemStyle.Render(title)
			desc = selectedItemStyle.Render(desc)
		} else {
			title = agentItemStyle.Render(title)
			desc = dimItemStyle.Render(desc)
		}
		fmt.Fprintf(w, "%s\n%s", title, desc)
		return
	}

	// Default rendering for regular sessions
	d.DefaultDelegate.Render(w, m, index, item)
}

func (d CustomDelegate) Height() int { return 2 }

func (d CustomDelegate) Spacing() int { return 0 }

// NewModel creates a new TUI model
func NewModel(adapter adapters.Adapter, cacheDir string) Model {
	// Load pins
	pins := NewPins(cacheDir)
	pins.Load()

	// Create list with custom delegate
	delegate := NewCustomDelegate()

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
		filterMode: FilterNone,
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
			item := m.getSelectedSession()
			if item != nil {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionResume,
				}
				if info, err := m.adapter.GetSessionInfo(item.entry.SessionID); err == nil {
					m.result.WorkDir = info.WorkDir
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Export):
			item := m.getSelectedSession()
			if item != nil {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionExport,
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.CopyMD):
			item := m.getSelectedSession()
			if item != nil {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionCopyMD,
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Branch):
			item := m.getSelectedSession()
			if item != nil {
				m.result = &Result{
					SessionID: item.entry.SessionID,
					Action:    ActionBranch,
				}
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Pin):
			item := m.getSelectedSession()
			if item != nil {
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

		// Quick filters
		case key.Matches(msg, m.keys.FilterToday):
			if m.filterMode == FilterToday {
				m.filterMode = FilterNone
				m.message = "Filter: All"
			} else {
				m.filterMode = FilterToday
				m.message = "Filter: Today"
			}
			m.updateListItems()

		case key.Matches(msg, m.keys.FilterWeek):
			if m.filterMode == FilterWeek {
				m.filterMode = FilterNone
				m.message = "Filter: All"
			} else {
				m.filterMode = FilterWeek
				m.message = "Filter: This week"
			}
			m.updateListItems()

		case key.Matches(msg, m.keys.FilterCost):
			if m.filterMode == FilterHighCost {
				m.filterMode = FilterNone
				m.message = "Filter: All"
			} else {
				m.filterMode = FilterHighCost
				m.message = "Filter: High cost (>$0.10)"
			}
			m.updateListItems()

		case key.Matches(msg, m.keys.FilterProject):
			item := m.getSelectedSession()
			if item != nil {
				if m.filterMode == FilterProject && m.filterValue == item.entry.Project {
					m.filterMode = FilterNone
					m.message = "Filter: All"
				} else {
					m.filterMode = FilterProject
					m.filterValue = item.entry.Project
					m.message = "Filter: " + item.entry.Project
				}
				m.updateListItems()
			}
		}
	}

	// Route updates to active pane
	if m.activePane == "list" {
		var cmd tea.Cmd
		oldIndex := m.list.Index()
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Update preview when selection changes
		if m.list.Index() != oldIndex {
			// Skip date headers
			m.skipHeaders()
			m.updatePreview()
		}
	} else {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// skipHeaders moves selection past date headers
func (m *Model) skipHeaders() {
	items := m.list.Items()
	idx := m.list.Index()
	if idx >= 0 && idx < len(items) {
		if item, ok := items[idx].(ListItem); ok && item.IsHeader() {
			// Move to next non-header item
			for i := idx + 1; i < len(items); i++ {
				if nextItem, ok := items[i].(ListItem); ok && !nextItem.IsHeader() {
					m.list.Select(i)
					return
				}
			}
			// If no next item, try previous
			for i := idx - 1; i >= 0; i-- {
				if prevItem, ok := items[i].(ListItem); ok && !prevItem.IsHeader() {
					m.list.Select(i)
					return
				}
			}
		}
	}
}

// getSelectedSession returns the selected session item (skipping headers)
func (m *Model) getSelectedSession() *SessionItem {
	selected := m.list.SelectedItem()
	if selected == nil {
		return nil
	}
	if item, ok := selected.(SessionItem); ok {
		return &item
	}
	return nil
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

	// Status bar with filter indicator
	filterIndicator := ""
	switch m.filterMode {
	case FilterToday:
		filterIndicator = " [Today]"
	case FilterWeek:
		filterIndicator = " [Week]"
	case FilterHighCost:
		filterIndicator = " [$$$]"
	case FilterProject:
		filterIndicator = " [" + m.filterValue + "]"
	}

	help := helpStyle.Render("enter:resume  p:pin  1:today  2:week  3:project  4:cost  ctrl+r:refresh  q:quit")
	status := statusBarStyle.Render(m.message + filterIndicator)
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

// updateListItems refreshes the list with grouped and sorted sessions
func (m *Model) updateListItems() {
	// Apply filters first
	filtered := m.applyFilters(m.sessions)

	// Separate main sessions and agent sessions
	mainSessions := make([]cache.Entry, 0)
	agentsByParent := make(map[string][]cache.Entry)

	for _, entry := range filtered {
		if entry.ParentSID != "" && entry.ParentSID != "-" {
			agentsByParent[entry.ParentSID] = append(agentsByParent[entry.ParentSID], entry)
		} else {
			mainSessions = append(mainSessions, entry)
		}
	}

	// Sort main sessions: pinned first, then by date
	sort.SliceStable(mainSessions, func(i, j int) bool {
		iPinned := m.pins.IsPinned(mainSessions[i].SessionID)
		jPinned := m.pins.IsPinned(mainSessions[j].SessionID)
		if iPinned != jPinned {
			return iPinned
		}
		return mainSessions[i].Date.After(mainSessions[j].Date)
	})

	// Sort agents by date within each parent
	for parentID := range agentsByParent {
		agents := agentsByParent[parentID]
		sort.SliceStable(agents, func(i, j int) bool {
			return agents[i].Date.After(agents[j].Date)
		})
		agentsByParent[parentID] = agents
	}

	// Build list with date headers and nested agents
	var items []list.Item
	currentDate := ""
	dateCount := 0

	for _, entry := range mainSessions {
		entryDate := entry.Date.Format("Monday, January 2, 2006")

		// Add date header if date changed
		if entryDate != currentDate {
			if currentDate != "" && dateCount > 0 {
				// Insert header for previous date at correct position
			}
			items = append(items, DateHeader{date: entryDate, count: 0})
			currentDate = entryDate
			dateCount = 0
		}
		dateCount++

		// Add main session
		items = append(items, SessionItem{
			entry:    entry,
			isPinned: m.pins.IsPinned(entry.SessionID),
			isAgent:  false,
			depth:    0,
		})

		// Add nested agent sessions
		if agents, ok := agentsByParent[entry.SessionID]; ok {
			for _, agent := range agents {
				items = append(items, SessionItem{
					entry:    agent,
					isPinned: m.pins.IsPinned(agent.SessionID),
					isAgent:  true,
					depth:    1,
				})
			}
		}
	}

	m.list.SetItems(items)

	// Skip to first non-header item
	m.skipHeaders()
}

// applyFilters filters sessions based on current filter mode
func (m *Model) applyFilters(sessions []cache.Entry) []cache.Entry {
	if m.filterMode == FilterNone {
		return sessions
	}

	var filtered []cache.Entry
	now := sessions[0].Date // Use most recent session date as reference

	for _, entry := range sessions {
		switch m.filterMode {
		case FilterToday:
			if entry.Date.Format("2006-01-02") == now.Format("2006-01-02") {
				filtered = append(filtered, entry)
			}
		case FilterWeek:
			weekAgo := now.AddDate(0, 0, -7)
			if entry.Date.After(weekAgo) {
				filtered = append(filtered, entry)
			}
		case FilterProject:
			if entry.Project == m.filterValue {
				filtered = append(filtered, entry)
			}
		case FilterHighCost:
			// Include all for now, filter on stats (would need adapter call)
			// For simplicity, include sessions - could be enhanced
			filtered = append(filtered, entry)
		default:
			filtered = append(filtered, entry)
		}
	}

	// If filter results empty, show all
	if len(filtered) == 0 {
		return sessions
	}
	return filtered
}

// updatePreview updates the preview pane content
func (m *Model) updatePreview() {
	item := m.getSelectedSession()
	if item == nil {
		m.preview.SetContent("No session selected")
		return
	}

	if m.showActivity {
		m.preview.SetContent("Activity heatmap (TODO)")
		return
	}

	// Generate preview content
	var b strings.Builder

	// Header with styling
	b.WriteString(fmt.Sprintf("ID: %s\n", item.entry.SessionID))
	b.WriteString(fmt.Sprintf("Project: %s\n", item.entry.Project))
	b.WriteString(fmt.Sprintf("Date: %s\n", item.entry.Date.Format("2006-01-02 15:04")))

	if item.isAgent {
		b.WriteString(fmt.Sprintf("Parent: %s\n", item.entry.ParentSID))
	}

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
		b.WriteString("━━━ Topics ━━━\n")
		for _, s := range summaries {
			b.WriteString(fmt.Sprintf("• %s\n", s))
		}
		b.WriteString("\n")
	}

	// Slash commands
	if cmds, err := m.adapter.GetSlashCommands(item.entry.SessionID); err == nil && len(cmds) > 0 {
		b.WriteString("━━━ Slash Commands ━━━\n")
		for _, cmd := range cmds {
			b.WriteString(fmt.Sprintf("  %s\n", cmd))
		}
		b.WriteString("\n")
	}

	// Files
	if files, err := m.adapter.GetFilesTouched(item.entry.SessionID); err == nil && len(files) > 0 {
		b.WriteString("━━━ Files ━━━\n")
		shown := files
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, f := range shown {
			b.WriteString(fmt.Sprintf("• %s\n", f))
		}
		if len(files) > 10 {
			b.WriteString(fmt.Sprintf("  ... and %d more\n", len(files)-10))
		}
		b.WriteString("\n")
	}

	// Stats
	if stats, err := m.adapter.GetStats(item.entry.SessionID); err == nil {
		b.WriteString("━━━ Stats ━━━\n")
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
			b.WriteString("━━━ First Message ━━━\n")
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
