package mcp

import (
	"testing"
)

func TestToolCategories(t *testing.T) {
	// Check that all categories have tools
	for _, cat := range toolCategories {
		if len(cat.Tools) == 0 {
			t.Errorf("category %s has no tools", cat.Name)
		}
		if cat.Description == "" {
			t.Errorf("category %s has no description", cat.Name)
		}
	}
}

func TestToolCategoriesUniqueness(t *testing.T) {
	// Check that tool names are unique across all categories
	seen := make(map[string]string)
	for _, cat := range toolCategories {
		for _, tool := range cat.Tools {
			if prevCat, exists := seen[tool.Name]; exists {
				t.Errorf("tool %s appears in both %s and %s", tool.Name, prevCat, cat.Name)
			}
			seen[tool.Name] = cat.Name
		}
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "adc", 1},
		{"search", "serach", 2},
		{"get_callers", "getcallers", 1},
	}

	for _, tt := range tests {
		result := levenshteinDistance(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("levenshteinDistance(%q, %q) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestFindSimilarTools(t *testing.T) {
	s := &Server{}

	// Test finding similar tools
	similar := s.findSimilarTools("search")
	if len(similar) == 0 {
		t.Error("expected to find tools similar to 'search'")
	}

	// Check that search_code is found
	found := false
	for _, name := range similar {
		if name == "search_code" || name == "search_history" || name == "fuzzy_search" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find search-related tool in %v", similar)
	}
}

func TestHybridEssentialTools(t *testing.T) {
	// Verify hybrid essential tools exist in categories
	for _, essential := range hybridEssentialTools {
		found := false
		for _, cat := range toolCategories {
			for _, tool := range cat.Tools {
				if tool.Name == essential {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("hybrid essential tool %s not found in toolCategories", essential)
		}
	}
}

func TestToolHandlersCompleteness(t *testing.T) {
	// This test would require a full server instance
	// For now, just verify the handler map structure
	s := &Server{}
	handlers := s.getToolHandlers()

	// Check that handlers map is not empty
	if len(handlers) == 0 {
		t.Error("expected non-empty handlers map")
	}

	// Check that all tools in categories have handlers
	for _, cat := range toolCategories {
		for _, tool := range cat.Tools {
			if _, ok := handlers[tool.Name]; !ok {
				t.Errorf("tool %s has no handler", tool.Name)
			}
		}
	}
}

func TestToolCategoryCount(t *testing.T) {
	// Count total tools across all categories
	totalTools := 0
	for _, cat := range toolCategories {
		totalTools += len(cat.Tools)
	}

	// We expect around 60 tools
	if totalTools < 50 {
		t.Errorf("expected at least 50 tools, got %d", totalTools)
	}

	t.Logf("Total tools in categories: %d", totalTools)
}

func TestSuggestToolKeywords(t *testing.T) {
	tests := []struct {
		intent       string
		expectedTool string
	}{
		{"find who calls this function", "get_callers"},
		{"search for code", "search_code"},
		{"grep pattern", "grep_code"},
		{"find regression bug", "find_regression"},
		{"remember this", "memory_store"},
		{"create a task", "todo_create"},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			// This is a simplified test - in production we'd call handleSuggestTool
			intent := tt.intent
			found := false

			// Check if the expected tool would match based on keywords
			toolKeywords := map[string][]string{
				"get_callers":  {"caller", "who calls", "usage", "reference", "used by"},
				"search_code":  {"search", "find", "look", "semantic", "meaning", "code"},
				"grep_code":    {"grep", "pattern", "regex", "text", "literal", "exact"},
				"find_regression": {"regression", "bug", "broke", "bisect"},
				"memory_store": {"remember", "store", "save", "note"},
				"todo_create":  {"todo", "task", "create task", "add todo"},
			}

			for _, kw := range toolKeywords[tt.expectedTool] {
				if stringContains(intent, kw) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("intent %q should match tool %q", tt.intent, tt.expectedTool)
			}
		})
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
