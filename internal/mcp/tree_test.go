package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spetr/mcp-codewizard/internal/config"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"app.ts", "typescript"},
		{"component.tsx", "tsx"},
		{"App.jsx", "jsx"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"program.c", "c"},
		{"program.cpp", "cpp"},
		{"header.h", "c-header"},
		{"script.rb", "ruby"},
		{"index.php", "php"},
		{"Program.cs", "csharp"},
		{"app.swift", "swift"},
		{"Main.kt", "kotlin"},
		{"app.scala", "scala"},
		{"script.lua", "lua"},
		{"query.sql", "sql"},
		{"script.sh", "shell"},
		{"index.html", "html"},
		{"style.css", "css"},
		{"App.vue", "vue"},
		{"Component.svelte", "svelte"},
		{"config.yaml", "yaml"},
		{"package.json", "json"},
		{"config.toml", "toml"},
		{"README.md", "markdown"},
		{"analysis.r", "r"},
		{"main.dart", "dart"},
		{"module.vb", "vbnet"},
		{"script.ps1", "powershell"},
		{"unknown.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := detectLanguage(tt.path)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetLanguageIcon(t *testing.T) {
	tests := []struct {
		lang     string
		expected string
	}{
		{"go", "🐹"},
		{"python", "🐍"},
		{"rust", "🦀"},
		{"java", "☕"},
		{"ruby", "💎"},
		{"unknown", "📄"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := getLanguageIcon(tt.lang)
			if result != tt.expected {
				t.Errorf("getLanguageIcon(%q) = %q, want %q", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestBuildTree(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create structure:
	// tmpDir/
	//   src/
	//     main.go
	//     utils.go
	//   tests/
	//     main_test.go
	//   README.md

	srcDir := filepath.Join(tmpDir, "src")
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(testsDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "utils.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "main_test.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		projectDir: tmpDir,
		config:     config.DefaultConfig(),
	}

	// Test building tree
	root, stats, err := s.buildTree(context.Background(), tmpDir, "", 5, 0, true, false, nil)
	if err != nil {
		t.Fatalf("buildTree failed: %v", err)
	}

	if root.Type != "directory" {
		t.Errorf("root type = %q, want 'directory'", root.Type)
	}

	if stats.files != 4 {
		t.Errorf("stats.files = %d, want 4", stats.files)
	}

	if stats.dirs != 3 { // root + src + tests
		t.Errorf("stats.dirs = %d, want 3", stats.dirs)
	}

	// Check children count (should be 3: src, tests, README.md)
	if len(root.Children) != 3 {
		t.Errorf("root.Children count = %d, want 3", len(root.Children))
	}
}

func TestBuildTreeDepthLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	nested := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "deep.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		projectDir: tmpDir,
		config:     config.DefaultConfig(),
	}

	// Test with depth limit of 2
	root, stats, err := s.buildTree(context.Background(), tmpDir, "", 2, 0, true, false, nil)
	if err != nil {
		t.Fatalf("buildTree failed: %v", err)
	}

	// Should be truncated
	if !stats.truncated {
		t.Error("expected truncated = true with depth limit 2")
	}

	// Deep file should not be included
	if stats.files != 0 {
		t.Errorf("stats.files = %d, want 0 (deep file should be excluded)", stats.files)
	}

	// Check that we have some directories
	if len(root.Children) == 0 {
		t.Error("expected at least one child directory")
	}
}

func TestBuildTreeExcludeHidden(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "visible.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("config"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		projectDir: tmpDir,
		config:     config.DefaultConfig(),
	}

	// Without hidden files
	_, stats, err := s.buildTree(context.Background(), tmpDir, "", 5, 0, true, false, nil)
	if err != nil {
		t.Fatalf("buildTree failed: %v", err)
	}

	if stats.files != 1 {
		t.Errorf("without hidden: stats.files = %d, want 1", stats.files)
	}

	// With hidden files
	_, stats, err = s.buildTree(context.Background(), tmpDir, "", 5, 0, true, true, nil)
	if err != nil {
		t.Fatalf("buildTree failed: %v", err)
	}

	if stats.files < 2 {
		t.Errorf("with hidden: stats.files = %d, want >= 2", stats.files)
	}
}

func TestFormatTreeAsText(t *testing.T) {
	s := &Server{}

	root := &TreeNode{
		Name: "project",
		Type: "directory",
		Path: ".",
		Children: []*TreeNode{
			{
				Name:      "src",
				Type:      "directory",
				Path:      "src",
				FileCount: 2,
				Children: []*TreeNode{
					{Name: "main.go", Type: "file", Path: "src/main.go", Language: "go", Indexed: true},
					{Name: "utils.go", Type: "file", Path: "src/utils.go", Language: "go"},
				},
			},
			{Name: "README.md", Type: "file", Path: "README.md", Language: "markdown"},
		},
	}

	text := s.formatTreeAsText(root, "", true)

	// Check that output contains expected elements
	if text == "" {
		t.Error("formatTreeAsText returned empty string")
	}

	// Should contain directory name
	if !contains(text, "project/") {
		t.Error("expected 'project/' in output")
	}

	// Should contain file names
	if !contains(text, "main.go") {
		t.Error("expected 'main.go' in output")
	}

	// Should contain indexed marker
	if !contains(text, "✓") {
		t.Error("expected '✓' marker for indexed file")
	}
}

func TestFormatTreeAsMarkdown(t *testing.T) {
	s := &Server{}

	root := &TreeNode{
		Name: "project",
		Type: "directory",
		Path: ".",
		Children: []*TreeNode{
			{Name: "main.go", Type: "file", Path: "main.go", Language: "go", Indexed: true},
		},
	}

	md := s.formatTreeAsMarkdown(root, 0)

	// Check that output contains expected elements
	if md == "" {
		t.Error("formatTreeAsMarkdown returned empty string")
	}

	// Should contain folder icon
	if !contains(md, "📁") {
		t.Error("expected folder icon in output")
	}

	// Should contain go icon
	if !contains(md, "🐹") {
		t.Error("expected Go icon (🐹) in output")
	}

	// Should contain indexed marker
	if !contains(md, "✅") {
		t.Error("expected indexed marker (✅) in output")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
