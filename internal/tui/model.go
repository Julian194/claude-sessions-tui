package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

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
	showHelp     bool
	filterMode   FilterMode
	filterValue  string // For project filter

	// Parent-child session grouping
	expanded   map[string]bool          // Which parent sessions are expanded
	children   map[string][]cache.Entry // Parent ID -> child sessions
	childCount map[string]int           // Parent ID -> child count

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
	l.SetShowStatusBar(false) // We have our own status bar
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
		expanded:   make(map[string]bool),
		children:   make(map[string][]cache.Entry),
		childCount: make(map[string]int),
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return m.loadSessions()
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

	case exportMsg:
		if msg.err != nil {
			m.message = "Export failed: " + msg.err.Error()
		} else {
			m.message = "Exported: " + msg.filename
		}
		return m, clearMessageAfter(3 * time.Second)

	case copyMsg:
		if msg.err != nil {
			m.message = "Copy failed: " + msg.err.Error()
		} else {
			m.message = "Copied to clipboard!"
		}
		return m, clearMessageAfter(3 * time.Second)

	case clearMessageMsg:
		m.message = ""

	case previewUpdateMsg:
		// Only update if we're still on the same item (debounce)
		if m.list.Index() == msg.index {
			m.updatePreview()
		}

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	// Navigation
	case key.Matches(msg, m.keys.Tab):
		if m.activePane == "list" {
			m.activePane = "preview"
		} else {
			m.activePane = "list"
		}

	// Session actions
	case key.Matches(msg, m.keys.Select):
		return m.handleSelect()

	case key.Matches(msg, m.keys.Export):
		return m.handleExport()

	case key.Matches(msg, m.keys.CopyMD):
		return m.handleCopyMD()

	case key.Matches(msg, m.keys.Open):
		return m.handleOpen()

	case key.Matches(msg, m.keys.Branch):
		return m.handleBranch()

	case key.Matches(msg, m.keys.Pin):
		m.handlePin()

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.message = "Refreshing..."
		return m, m.refreshCache()

	case key.Matches(msg, m.keys.ToggleActivity):
		m.showActivity = !m.showActivity
		m.updatePreview()

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	// Expand/collapse parent sessions
	case key.Matches(msg, m.keys.ToggleExpand):
		item := m.getSelectedSession()
		if item != nil && item.isParent {
			m.expanded[item.entry.SessionID] = !m.expanded[item.entry.SessionID]
			m.updateListItems()
		}
		return m, nil

	case key.Matches(msg, m.keys.Expand):
		item := m.getSelectedSession()
		if item != nil && item.isParent && !m.expanded[item.entry.SessionID] {
			m.expanded[item.entry.SessionID] = true
			m.updateListItems()
		}
		return m, nil

	case key.Matches(msg, m.keys.Collapse):
		item := m.getSelectedSession()
		if item != nil && item.isParent && m.expanded[item.entry.SessionID] {
			m.expanded[item.entry.SessionID] = false
			m.updateListItems()
		}
		return m, nil

	// Quick filters
	case key.Matches(msg, m.keys.FilterToday):
		m.toggleFilter(FilterToday, "Today")

	case key.Matches(msg, m.keys.FilterWeek):
		m.toggleFilter(FilterWeek, "This week")

	case key.Matches(msg, m.keys.FilterCost):
		m.toggleFilter(FilterHighCost, "High cost (>$0.10)")

	case key.Matches(msg, m.keys.FilterProject):
		if item := m.getSelectedSession(); item != nil {
			if m.filterMode == FilterProject && m.filterValue == item.entry.Project {
				m.toggleFilter(FilterNone, "All")
			} else {
				m.filterMode = FilterProject
				m.filterValue = item.entry.Project
				m.message = "Filter: " + item.entry.Project
				m.updateListItems()
			}
		}

	case key.Matches(msg, m.keys.FilterClear):
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.filterMode != FilterNone {
			m.filterMode = FilterNone
			m.message = "Filter cleared"
			m.updateListItems()
		}
	}

	// Pass unhandled keys to the active pane
	var cmds []tea.Cmd
	if m.activePane == "list" {
		var cmd tea.Cmd
		oldIndex := m.list.Index()
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Skip date headers (not selectable)
		newIndex := m.list.Index()
		if _, isHeader := m.list.SelectedItem().(DateHeaderItem); isHeader {
			m.skipDateHeader(oldIndex, newIndex)
		}

		// Debounced preview update when selection changes
		if m.list.Index() != oldIndex {
			cmds = append(cmds, requestPreviewUpdate(m.list.Index()))
		}
	} else {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// Session action handlers

func (m Model) handleSelect() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleExport() (tea.Model, tea.Cmd) {
	item := m.getSelectedSession()
	if item != nil {
		m.message = "Exporting..."
		return m, m.exportHTML(item.entry.SessionID)
	}
	return m, nil
}

func (m Model) handleCopyMD() (tea.Model, tea.Cmd) {
	item := m.getSelectedSession()
	if item != nil {
		m.message = "Copying..."
		return m, m.copyMarkdown(item.entry.SessionID)
	}
	return m, nil
}

func (m Model) handleOpen() (tea.Model, tea.Cmd) {
	item := m.getSelectedSession()
	if item != nil {
		m.result = &Result{
			SessionID: item.entry.SessionID,
			Action:    ActionOpen,
		}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleBranch() (tea.Model, tea.Cmd) {
	item := m.getSelectedSession()
	if item != nil {
		m.result = &Result{
			SessionID: item.entry.SessionID,
			Action:    ActionBranch,
		}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) handlePin() {
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
}

// Helper methods

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

func (m *Model) toggleFilter(mode FilterMode, label string) {
	if m.filterMode == mode {
		m.filterMode = FilterNone
		m.message = "Filter: All"
	} else {
		m.filterMode = mode
		m.message = "Filter: " + label
	}
	m.updateListItems()
}

func (m *Model) skipDateHeader(oldIndex, newIndex int) {
	if newIndex > oldIndex {
		// Moving down - find next session
		for i := newIndex + 1; i < len(m.list.Items()); i++ {
			if _, ok := m.list.Items()[i].(SessionItem); ok {
				m.list.Select(i)
				break
			}
		}
	} else {
		// Moving up - find previous session
		for i := newIndex - 1; i >= 0; i-- {
			if _, ok := m.list.Items()[i].(SessionItem); ok {
				m.list.Select(i)
				break
			}
		}
	}
}

// List and filtering

func (m *Model) updateListItems() {
	// Apply filters first - make a copy to avoid modifying original
	filtered := make([]cache.Entry, len(m.sessions))
	copy(filtered, m.sessions)
	filtered = m.applyFilters(filtered)

	// Build parent-child relationships
	m.children = make(map[string][]cache.Entry)
	m.childCount = make(map[string]int)

	// Separate parent sessions and child sessions
	var parentSessions []cache.Entry
	childSessions := make(map[string][]cache.Entry)

	for _, entry := range filtered {
		isAgent := entry.ParentSID != "" && entry.ParentSID != "-"
		if isAgent {
			childSessions[entry.ParentSID] = append(childSessions[entry.ParentSID], entry)
		} else {
			parentSessions = append(parentSessions, entry)
		}
	}

	// Sort parent sessions: pinned first, then by date (newest first)
	sort.Slice(parentSessions, func(i, j int) bool {
		iPinned := m.pins.IsPinned(parentSessions[i].SessionID)
		jPinned := m.pins.IsPinned(parentSessions[j].SessionID)
		if iPinned != jPinned {
			return iPinned
		}
		return parentSessions[i].Date.After(parentSessions[j].Date)
	})

	// Sort children by date within each parent
	for parentID, children := range childSessions {
		sort.Slice(children, func(i, j int) bool {
			return children[i].Date.After(children[j].Date)
		})
		m.children[parentID] = children
		m.childCount[parentID] = len(children)
	}

	// Build list items with parent/child structure and date headers
	var items []list.Item
	var lastDate string
	for _, entry := range parentSessions {
		hasChildren := m.childCount[entry.SessionID] > 0
		isExpanded := m.expanded[entry.SessionID]

		// Insert date header before first session of each day
		currentDate := entry.Date.Format("2006-01-02")
		if currentDate != lastDate {
			items = append(items, DateHeaderItem{date: formatDateHeader(entry.Date)})
			lastDate = currentDate
		}

		items = append(items, SessionItem{
			entry:      entry,
			isPinned:   m.pins.IsPinned(entry.SessionID),
			isAgent:    false,
			isParent:   hasChildren,
			isExpanded: isExpanded,
			childCount: m.childCount[entry.SessionID],
		})

		// Add children if parent is expanded
		if hasChildren && isExpanded {
			for _, child := range m.children[entry.SessionID] {
				items = append(items, SessionItem{
					entry:    child,
					isPinned: m.pins.IsPinned(child.SessionID),
					isAgent:  true,
				})
			}
		}
	}

	// Preserve current selection if possible
	var currentID string
	if selected := m.getSelectedSession(); selected != nil {
		currentID = selected.entry.SessionID
	}

	m.list.SetItems(items)

	// Try to re-select the previously selected session
	selectedIndex := -1
	if currentID != "" {
		for i, item := range items {
			if si, ok := item.(SessionItem); ok && si.entry.SessionID == currentID {
				selectedIndex = i
				break
			}
		}
	}

	// Fall back to first session if previous selection not found
	if selectedIndex < 0 {
		for i, item := range items {
			if _, ok := item.(SessionItem); ok {
				selectedIndex = i
				break
			}
		}
	}

	if selectedIndex >= 0 {
		m.list.Select(selectedIndex)
	}
	m.updatePreview()
}

func (m *Model) applyFilters(sessions []cache.Entry) []cache.Entry {
	if m.filterMode == FilterNone || m.filterMode == FilterHighCost || len(sessions) == 0 {
		return sessions
	}

	now := sessions[0].Date
	var filtered []cache.Entry

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
		}
	}

	if len(filtered) == 0 {
		return sessions
	}
	return filtered
}

// Result returns the user's selection (nil if cancelled)
func (m Model) Result() *Result {
	if m.done && m.result != nil {
		return m.result
	}
	return nil
}

// formatDateHeader returns a friendly date string
func formatDateHeader(t time.Time) string {
	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	date := t.Format("2006-01-02")

	switch date {
	case today:
		return "Today, " + t.Format("Jan 2")
	case yesterday:
		return "Yesterday, " + t.Format("Jan 2")
	default:
		return t.Format("Mon, Jan 2")
	}
}
