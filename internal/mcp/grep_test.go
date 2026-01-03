package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spetr/mcp-codewizard/internal/config"
)

func TestMatchGlobPath(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"**/*.go", "main.go", true},
		{"**/*.go", "internal/mcp/server.go", true},
		{"**/*.go", "main.ts", false},
		{"*.go", "main.go", true},
		{"*.go", "internal/main.go", true}, // matches basename
		{"src/**/*.ts", "src/app/main.ts", true},
		{"src/**/*.ts", "lib/main.ts", false},
		{"**/vendor/**", "vendor/lib/file.go", true},
		{"**/node_modules/**", "node_modules/pkg/index.js", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := matchGlobPath(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchGlobPath(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

func TestSearchFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func helper() {
	fmt.Println("helper function")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		projectDir: tmpDir,
		config:     config.DefaultConfig(),
	}

	// Test basic search
	result, err := s.grepWithGoRegexp(context.Background(), "fmt\\.Println", "", 1, 10, false, false, false)
	if err != nil {
		t.Fatalf("grepWithGoRegexp failed: %v", err)
	}

	if len(result.Matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(result.Matches))
	}

	if result.Backend != "go-regexp" {
		t.Errorf("expected backend 'go-regexp', got %q", result.Backend)
	}

	// Verify first match
	if len(result.Matches) > 0 {
		m := result.Matches[0]
		if m.Line != 6 {
			t.Errorf("expected line 6, got %d", m.Line)
		}
		if len(m.ContextBefore) != 1 {
			t.Errorf("expected 1 context line before, got %d", len(m.ContextBefore))
		}
		if len(m.ContextAfter) != 1 {
			t.Errorf("expected 1 context line after, got %d", len(m.ContextAfter))
		}
	}
}

func TestSearchFileCaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func TestFunction() {
	// test comment
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		projectDir: tmpDir,
		config:     config.DefaultConfig(),
	}

	// Case insensitive search for "test"
	result, err := s.grepWithGoRegexp(context.Background(), "test", "", 0, 10, false, false, false)
	if err != nil {
		t.Fatalf("grepWithGoRegexp failed: %v", err)
	}

	// Should match both "TestFunction" and "test comment"
	if len(result.Matches) != 2 {
		t.Errorf("expected 2 case-insensitive matches, got %d", len(result.Matches))
	}

	// Case sensitive search
	result, err = s.grepWithGoRegexp(context.Background(), "test", "", 0, 10, true, false, false)
	if err != nil {
		t.Fatalf("grepWithGoRegexp failed: %v", err)
	}

	// Should only match "test comment"
	if len(result.Matches) != 1 {
		t.Errorf("expected 1 case-sensitive match, got %d", len(result.Matches))
	}
}

func TestSearchFileLiteral(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

// Pattern: [a-z]+
func main() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		projectDir: tmpDir,
		config:     config.DefaultConfig(),
	}

	// Literal search for regex characters
	result, err := s.grepWithGoRegexp(context.Background(), "[a-z]+", "", 0, 10, false, false, true)
	if err != nil {
		t.Fatalf("grepWithGoRegexp failed: %v", err)
	}

	if len(result.Matches) != 1 {
		t.Errorf("expected 1 literal match, got %d", len(result.Matches))
	}
}

func TestSearchFileMaxResults(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file with many lines
	var content string
	for i := 0; i < 100; i++ {
		content += "test line\n"
	}
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		projectDir: tmpDir,
		config:     config.DefaultConfig(),
	}

	// Limit to 5 results
	result, err := s.grepWithGoRegexp(context.Background(), "test", "", 0, 5, false, false, false)
	if err != nil {
		t.Fatalf("grepWithGoRegexp failed: %v", err)
	}

	if len(result.Matches) != 5 {
		t.Errorf("expected 5 matches (limited), got %d", len(result.Matches))
	}

	if !result.Truncated {
		t.Error("expected Truncated to be true")
	}
}
