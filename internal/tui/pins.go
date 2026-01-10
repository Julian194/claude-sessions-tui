package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Pins manages pinned session IDs
type Pins struct {
	ids  map[string]bool
	path string
}

// NewPins creates a new Pins manager
func NewPins(cacheDir string) *Pins {
	return &Pins{
		ids:  make(map[string]bool),
		path: filepath.Join(cacheDir, "pinned-sessions.txt"),
	}
}

// Load reads pinned sessions from disk
func (p *Pins) Load() error {
	f, err := os.Open(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No pins file yet
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		id := strings.TrimSpace(scanner.Text())
		if id != "" {
			p.ids[id] = true
		}
	}
	return scanner.Err()
}

// Save writes pinned sessions to disk
func (p *Pins) Save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(p.path), 0755); err != nil {
		return err
	}

	f, err := os.Create(p.path)
	if err != nil {
		return err
	}
	defer f.Close()

	for id := range p.ids {
		if _, err := f.WriteString(id + "\n"); err != nil {
			return err
		}
	}
	return nil
}

// Toggle adds or removes a session ID from pins
func (p *Pins) Toggle(id string) bool {
	if p.ids[id] {
		delete(p.ids, id)
		return false
	}
	p.ids[id] = true
	return true
}

// IsPinned checks if a session is pinned
func (p *Pins) IsPinned(id string) bool {
	return p.ids[id]
}

// Count returns the number of pinned sessions
func (p *Pins) Count() int {
	return len(p.ids)
}
