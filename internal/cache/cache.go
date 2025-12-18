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
	ID      string
	Date    time.Time
	Project string
	Summary string
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
	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range entries {
		// Escape special characters in fields
		summary := escapeTSV(e.Summary)
		project := escapeTSV(e.Project)

		line := fmt.Sprintf("%s\t%s\t%s\t%s\n",
			e.ID,
			e.Date.Format("2006-01-02 15:04"),
			project,
			summary,
		)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// Read reads entries from the cache file
func (c *Cache) Read() ([]Entry, error) {
	f, err := os.Open(c.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue // Skip malformed lines
		}

		date, _ := time.Parse("2006-01-02 15:04", parts[1])

		entries = append(entries, Entry{
			ID:      parts[0],
			Date:    date,
			Project: unescapeTSV(parts[2]),
			Summary: unescapeTSV(parts[3]),
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
	sessions, err := adapter.ListSessions()
	if err != nil {
		return err
	}

	var entries []Entry
	for _, id := range sessions {
		meta, err := adapter.ExtractMeta(id)
		if err != nil {
			continue // Skip sessions that fail to parse
		}
		entries = append(entries, Entry{
			ID:      meta.ID,
			Date:    meta.Date,
			Project: meta.Project,
			Summary: meta.Summary,
		})
	}

	return c.Write(entries)
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
