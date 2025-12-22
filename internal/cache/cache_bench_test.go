package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Julian194/claude-sessions-tui/internal/adapters"
	"github.com/Julian194/claude-sessions-tui/internal/adapters/claude"
)

// BenchmarkCacheRead measures the performance of reading the cache file
func BenchmarkCacheRead(b *testing.B) {
	tmpDir := b.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	// Create a cache with 100 entries
	entries := generateEntries(100)
	Write(cachePath, entries)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Read(cachePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCacheRead_Large measures reading a larger cache
func BenchmarkCacheRead_Large(b *testing.B) {
	tmpDir := b.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	// Create a cache with 1000 entries
	entries := generateEntries(1000)
	Write(cachePath, entries)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Read(cachePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCacheWrite measures the performance of writing the cache file
func BenchmarkCacheWrite(b *testing.B) {
	tmpDir := b.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	entries := generateEntries(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Write(cachePath, entries)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBuildIncremental_Mock measures BuildIncremental with mock data
func BenchmarkBuildIncremental_Mock(b *testing.B) {
	tmpDir := b.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.tsv")

	// Create mock session files
	sessionFiles := make(map[string]string)
	metas := make(map[string]*adapters.SessionMeta)
	sessions := make([]string, 50)

	for i := 0; i < 50; i++ {
		id := generateSessionID(i)
		sessions[i] = id

		sessionFile := filepath.Join(tmpDir, id+".jsonl")
		os.WriteFile(sessionFile, []byte(`{"type":"test"}`), 0644)
		sessionFiles[id] = sessionFile

		metas[id] = &adapters.SessionMeta{
			ID:      id,
			Date:    time.Now().Add(-time.Duration(i) * time.Hour),
			Project: "test-project",
			Summary: "Test session summary that is reasonably long to simulate real data",
		}
	}

	mock := &mockAdapter{
		sessions:    sessions,
		sessionFile: sessionFiles,
		metas:       metas,
	}

	// Initial cache write
	Write(cachePath, []Entry{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := BuildIncremental(mock, cachePath, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRealWorld_BuildIncremental uses real Claude data
func BenchmarkRealWorld_BuildIncremental(b *testing.B) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".claude", "projects")
	cacheDir := filepath.Join(home, ".claude", ".cache")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real Claude data found")
	}

	adapter := claude.New(dataDir)
	cachePath := filepath.Join(cacheDir, "sessions-cache.tsv")

	// Read existing cache for incremental build
	existing, _ := Read(cachePath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entries, err := BuildIncremental(adapter, cachePath, existing)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(entries)), "entries")
	}
}

// BenchmarkRealWorld_CacheRead reads the real cache file
func BenchmarkRealWorld_CacheRead(b *testing.B) {
	home, _ := os.UserHomeDir()
	cachePath := filepath.Join(home, ".claude", ".cache", "sessions-cache.tsv")

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		b.Skip("No real cache file found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entries, err := Read(cachePath)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(entries)), "entries")
	}
}

// Helper functions

func generateEntries(n int) []Entry {
	entries := make([]Entry, n)
	for i := 0; i < n; i++ {
		entries[i] = Entry{
			SessionID: generateSessionID(i),
			Date:      time.Now().Add(-time.Duration(i) * time.Hour),
			Project:   "test-project",
			Summary:   "This is a test summary that has some reasonable length to it",
			ParentSID: "",
		}
	}
	return entries
}

func generateSessionID(i int) string {
	return "session-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}

// Note: mockAdapter is defined in cache_test.go and shared with benchmarks
