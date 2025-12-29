package sqlitevec

import (
	"os"
	"testing"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

func TestGetCallersWithSymbolID(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-callers-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store := New()
	if err := store.Init(tmpFile.Name()); err != nil {
		if containsString(err.Error(), "fts5") || containsString(err.Error(), "FTS5") {
			t.Skip("FTS5 not available in this environment")
		}
		t.Fatal(err)
	}
	defer store.Close()

	// Store a reference where to_symbol is just the function name (as chunker does)
	refs := []*types.Reference{
		{
			ID:         "test.go:10:call:DefaultConfig",
			FromSymbol: "main",
			ToSymbol:   "DefaultConfig", // This is how chunker stores it - just the name
			Kind:       types.RefKindCall,
			FilePath:   "test.go",
			Line:       10,
			IsExternal: false,
		},
		{
			ID:         "other.go:20:call:DefaultConfig",
			FromSymbol: "init",
			ToSymbol:   "DefaultConfig",
			Kind:       types.RefKindCall,
			FilePath:   "other.go",
			Line:       20,
			IsExternal: false,
		},
	}

	if err := store.StoreReferences(refs); err != nil {
		t.Fatal(err)
	}

	// Test 1: Query with full symbol ID (as dead code analyzer does)
	fullSymbolID := "internal/config/config.go:DefaultConfig:96"
	callers, err := store.GetCallers(fullSymbolID, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(callers) != 2 {
		t.Errorf("GetCallers with full ID: expected 2 callers, got %d", len(callers))
	}

	// Test 2: Query with just the name
	callers2, err := store.GetCallers("DefaultConfig", 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(callers2) != 2 {
		t.Errorf("GetCallers with name: expected 2 callers, got %d", len(callers2))
	}

	// Test 3: Query with non-existent symbol
	callers3, err := store.GetCallers("NonExistent", 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(callers3) != 0 {
		t.Errorf("GetCallers with non-existent: expected 0 callers, got %d", len(callers3))
	}
}

func TestExtractSymbolName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"internal/config/config.go:DefaultConfig:96", "DefaultConfig"},
		{"pkg/types/types.go:DetermineComplexity:228", "DetermineComplexity"},
		{"simple/chunker.go:DetectLanguage:218", "DetectLanguage"},
		{"DefaultConfig", "DefaultConfig"},
		{"a:b:c", "b"},
		{"name", "name"},
	}

	for _, tt := range tests {
		got := extractSymbolName(tt.input)
		if got != tt.want {
			t.Errorf("extractSymbolName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractQualifiedName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"path/to/dart/binding.go:GetLanguage:16", "dart.GetLanguage"},
		{"builtin/chunking/treesitter/json/binding.go:GetLanguage:15", "json.GetLanguage"},
		{"internal/config/config.go:DefaultConfig:96", "config.DefaultConfig"},
		{"pkg/types/types.go:DetermineComplexity:228", "types.DetermineComplexity"},
		{"file.go:Func:1", ""},       // edge case: no directory -> empty (expected)
		{"Func", ""},                 // no path info
	}

	for _, tt := range tests {
		got := extractQualifiedName(tt.input)
		if got != tt.want {
			t.Errorf("extractQualifiedName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
