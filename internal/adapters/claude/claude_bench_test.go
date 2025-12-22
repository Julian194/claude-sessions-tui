package claude

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkListSessions measures the performance of listing all sessions
func BenchmarkListSessions(b *testing.B) {
	a := New(benchDataDir(b))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetSessionFile measures the performance of finding a session file
// This is critical because it's called for EVERY session in BuildIncremental
func BenchmarkGetSessionFile(b *testing.B) {
	a := New(benchDataDir(b))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := a.GetSessionFile("test-session")
		if path == "" {
			b.Fatal("session not found")
		}
	}
}

// BenchmarkExtractMeta measures the performance of extracting metadata from a session
func BenchmarkExtractMeta(b *testing.B) {
	a := New(benchDataDir(b))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.ExtractMeta("test-session")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseFile measures the performance of parsing a JSONL session file
func BenchmarkParseFile(b *testing.B) {
	a := New(benchDataDir(b))
	path := a.GetSessionFile("test-session")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.parseFile(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchDataDir returns the testdata directory path for benchmarks
func benchDataDir(b *testing.B) string {
	b.Helper()
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	return filepath.Join(wd, "testdata")
}

// BenchmarkRealWorld runs benchmarks against real Claude data if available
// Skip if no real data exists
func BenchmarkRealWorld_ListSessions(b *testing.B) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".claude", "projects")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real Claude data found")
	}

	a := New(dataDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessions, err := a.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(sessions)), "sessions")
	}
}

func BenchmarkRealWorld_GetSessionFile(b *testing.B) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".claude", "projects")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real Claude data found")
	}

	a := New(dataDir)

	// Get a real session ID
	sessions, err := a.ListSessions()
	if err != nil || len(sessions) == 0 {
		b.Skip("No sessions found")
	}
	sessionID := sessions[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := a.GetSessionFile(sessionID)
		if path == "" {
			b.Fatal("session not found")
		}
	}
}

func BenchmarkRealWorld_ExtractMeta(b *testing.B) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".claude", "projects")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real Claude data found")
	}

	a := New(dataDir)

	sessions, err := a.ListSessions()
	if err != nil || len(sessions) == 0 {
		b.Skip("No sessions found")
	}
	sessionID := sessions[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.ExtractMeta(sessionID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRealWorld_FullLoad simulates a full cache rebuild
func BenchmarkRealWorld_FullLoad(b *testing.B) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".claude", "projects")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real Claude data found")
	}

	a := New(dataDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessions, err := a.ListSessions()
		if err != nil {
			b.Fatal(err)
		}

		// Simulate what BuildIncremental does
		for _, id := range sessions {
			path := a.GetSessionFile(id)
			if path == "" {
				continue
			}
			_, _ = a.ExtractMeta(id)
		}

		b.ReportMetric(float64(len(sessions)), "sessions")
	}
}
