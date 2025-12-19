package cache

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
)

// Entry represents a single cache entry
type Entry struct {
	SessionID string
	Date      time.Time
	Project   string
	Summary   string
	ParentSID string // Parent session ID for branches
}

// Deprecated: ID is deprecated, use SessionID instead
func (e Entry) ID() string {
	return e.SessionID
}

// Cache manages the session cache file
type Cache struct {
	path string
}

// New creates a new cache manager
func New(path string) *Cache {
	return &Cache{path: path}
}

// Path returns the cache file path
func (c *Cache) Path() string {
	return c.path
}

// Write writes entries to the cache file in TSV format
func (c *Cache) Write(entries []Entry) error {
	return Write(c.path, entries)
}

// Write writes entries to a cache file in TSV format (standalone function)
func Write(path string, entries []Entry) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range entries {
		// Escape special characters in fields
		summary := escapeTSV(e.Summary)
		project := escapeTSV(e.Project)

		// TSV format: sid, date, project, summary, mtime, parent_sid, full_date
		parentSID := e.ParentSID
		if parentSID == "" {
			parentSID = "-"
		}

		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			e.SessionID,
			e.Date.Format("15:04"),
			project,
			summary,
			e.Date.Unix(),
			parentSID,
			e.Date.Format("2006-01-02"),
		)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// Read reads entries from the cache file
func (c *Cache) Read() ([]Entry, error) {
	return Read(c.path)
}

// Read reads entries from a cache file (standalone function)
func Read(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue // Skip malformed lines
		}

		// Parse mtime if available (column 5)
		var date time.Time
		if len(parts) >= 5 {
			var mtime int64
			fmt.Sscanf(parts[4], "%d", &mtime)
			date = time.Unix(mtime, 0)
		} else {
			// Fallback to parsing time from column 2
			date, _ = time.Parse("15:04", parts[1])
		}

		// Parse parent_sid if available (column 6)
		parentSID := ""
		if len(parts) >= 6 && parts[5] != "-" {
			parentSID = parts[5]
		}

		entries = append(entries, Entry{
			SessionID: parts[0],
			Date:      date,
			Project:   unescapeTSV(parts[2]),
			Summary:   unescapeTSV(parts[3]),
			ParentSID: parentSID,
		})
	}

	return entries, scanner.Err()
}

// Exists checks if the cache file exists
func (c *Cache) Exists() bool {
	_, err := os.Stat(c.path)
	return err == nil
}

// ModTime returns the cache file modification time
func (c *Cache) ModTime() (time.Time, error) {
	info, err := os.Stat(c.path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Clear removes the cache file
func (c *Cache) Clear() error {
	if !c.Exists() {
		return nil
	}
	return os.Remove(c.path)
}

// BuildFrom builds the cache from an adapter
func (c *Cache) BuildFrom(adapter adapters.Adapter) error {
	entries, err := BuildFrom(adapter)
	if err != nil {
		return err
	}
	return c.Write(entries)
}

// BuildFrom builds cache entries from an adapter (standalone function)
func BuildFrom(adapter adapters.Adapter) ([]Entry, error) {
	return BuildIncremental(adapter, "", nil)
}

// BuildIncremental builds cache entries incrementally, only processing files newer than cache
func BuildIncremental(adapter adapters.Adapter, cachePath string, existing []Entry) ([]Entry, error) {
	// Get cache mtime for incremental check
	var cacheMtime time.Time
	if cachePath != "" {
		if info, err := os.Stat(cachePath); err == nil {
			cacheMtime = info.ModTime()
		}
	}

	// Build lookup of existing entries
	existingMap := make(map[string]Entry)
	for _, e := range existing {
		existingMap[e.SessionID] = e
	}

	sessions, err := adapter.ListSessions()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, id := range sessions {
		// Check if file is newer than cache
		sessionPath := adapter.GetSessionFile(id)
		if sessionPath == "" {
			continue
		}

		info, err := os.Stat(sessionPath)
		if err != nil {
			continue
		}

		// If cache exists and file is older, use existing entry
		if !cacheMtime.IsZero() && info.ModTime().Before(cacheMtime) {
			if existing, ok := existingMap[id]; ok {
				entries = append(entries, existing)
				continue
			}
		}

		// Extract fresh metadata
		meta, err := adapter.ExtractMeta(id)
		if err != nil {
			continue
		}
		entries = append(entries, Entry{
			SessionID: meta.ID,
			Date:      meta.Date,
			Project:   meta.Project,
			Summary:   meta.Summary,
			ParentSID: meta.ParentSID,
		})
	}

	return entries, nil
}

// Helper functions

func escapeTSV(s string) string {
	// Replace tabs and newlines with spaces
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

func unescapeTSV(s string) string {
	// Currently no escaping needed for reading
	return s
}
