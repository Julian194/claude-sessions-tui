package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatJSONL(t *testing.T) {
	// Create temp input file with JSONL content
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jsonl")

	input := `{"type":"message","content":"hello"}
{"type":"response","content":"world"}
`
	if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	outputPath, err := formatJSONL(inputPath, "testid12")
	if err != nil {
		t.Fatalf("formatJSONL failed: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("output file was not created: %s", outputPath)
	}

	// Read and verify content is pretty-printed
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Check that output contains indentation (pretty-printed)
	if !strings.Contains(string(content), "  ") {
		t.Error("output should be pretty-printed with indentation")
	}

	// Check that both objects are present
	if !strings.Contains(string(content), `"type": "message"`) {
		t.Error("output should contain first JSON object")
	}
	if !strings.Contains(string(content), `"type": "response"`) {
		t.Error("output should contain second JSON object")
	}

	// Cleanup
	os.Remove(outputPath)
}

func TestFormatJSONL_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jsonl")

	// Input with empty lines that should be skipped
	input := `{"type":"message"}

{"type":"response"}
`
	if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	outputPath, err := formatJSONL(inputPath, "testid34")
	if err != nil {
		t.Fatalf("formatJSONL failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Should have 2 JSON objects separated by double newline
	parts := strings.Split(strings.TrimSpace(string(content)), "\n\n")
	if len(parts) != 2 {
		t.Errorf("expected 2 JSON objects, got %d", len(parts))
	}

	os.Remove(outputPath)
}

func TestFormatJSONL_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jsonl")

	// Input with invalid JSON line - should be preserved as-is
	input := `{"type":"valid"}
not valid json
{"type":"alsovalid"}
`
	if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	outputPath, err := formatJSONL(inputPath, "testid56")
	if err != nil {
		t.Fatalf("formatJSONL failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Invalid JSON line should be preserved
	if !strings.Contains(string(content), "not valid json") {
		t.Error("invalid JSON line should be preserved as-is")
	}

	os.Remove(outputPath)
}

func TestFormatJSONL_FileNotFound(t *testing.T) {
	_, err := formatJSONL("/nonexistent/path/file.jsonl", "testid78")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
