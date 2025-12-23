package opencode

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
func BenchmarkGetSessionFile(b *testing.B) {
	a := New(benchDataDir(b))

	// Get a session ID from the test data
	sessions, _ := a.ListSessions()
	if len(sessions) == 0 {
		b.Skip("No test sessions found")
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

// BenchmarkExtractMeta measures the performance of extracting metadata
func BenchmarkExtractMeta(b *testing.B) {
	a := New(benchDataDir(b))

	sessions, _ := a.ListSessions()
	if len(sessions) == 0 {
		b.Skip("No test sessions found")
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

func benchDataDir(b *testing.B) string {
	b.Helper()
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	return filepath.Join(wd, "testdata", "storage")
}

// Real-world benchmarks (skip if no data)
func BenchmarkRealWorld_ListSessions(b *testing.B) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".local", "share", "opencode", "storage")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real OpenCode data found")
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
	dataDir := filepath.Join(home, ".local", "share", "opencode", "storage")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real OpenCode data found")
	}

	a := New(dataDir)

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
	dataDir := filepath.Join(home, ".local", "share", "opencode", "storage")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		b.Skip("No real OpenCode data found")
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
