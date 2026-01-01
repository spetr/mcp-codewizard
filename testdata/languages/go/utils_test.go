// Package main - utils_test.go tests test function detection.
// All functions here should be excluded from dead code analysis.
package main

import (
	"testing"
)

// TestProcessData tests ProcessData function.
// Should be excluded from dead code analysis.
func TestProcessData(t *testing.T) {
	result := ProcessData([]string{"hello", "world"})
	expected := "HELLO, WORLD"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestHashString tests HashString function.
// Should be excluded - even though it calls dead code.
func TestHashString(t *testing.T) {
	result := HashString("test")
	if len(result) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(result))
	}
}

// TestFilterStrings tests FilterStrings function.
func TestFilterStrings(t *testing.T) {
	items := []string{"apple", "banana", "cherry"}
	result := FilterStrings(items, func(s string) bool {
		return len(s) > 5
	})
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

// TestSimpleFunction tests SimpleFunction.
func TestSimpleFunction(t *testing.T) {
	if SimpleFunction(5) != 10 {
		t.Error("SimpleFunction(5) should be 10")
	}
}

// TestNestedIfFunction tests NestedIfFunction.
func TestNestedIfFunction(t *testing.T) {
	tests := []struct {
		x, y     int
		expected string
	}{
		{1, 1, "both positive"},
		{1, -1, "only x positive"},
		{-1, 1, "x not positive"},
	}

	for _, tt := range tests {
		result := NestedIfFunction(tt.x, tt.y)
		if result != tt.expected {
			t.Errorf("NestedIfFunction(%d, %d) = %q, want %q",
				tt.x, tt.y, result, tt.expected)
		}
	}
}

// BenchmarkProcessData benchmarks ProcessData.
// Benchmark functions should also be excluded.
func BenchmarkProcessData(b *testing.B) {
	items := []string{"a", "b", "c", "d", "e"}
	for i := 0; i < b.N; i++ {
		ProcessData(items)
	}
}

// BenchmarkHashString benchmarks HashString.
func BenchmarkHashString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashString("benchmark test string")
	}
}

// ExampleProcessData demonstrates ProcessData usage.
// Example functions should also be excluded.
func ExampleProcessData() {
	result := ProcessData([]string{"hello", "world"})
	// Output: HELLO, WORLD
	_ = result
}

// testHelper is a test helper - should be excluded.
func testHelper(t *testing.T, name string) {
	t.Helper()
	t.Logf("Running test: %s", name)
}

// setupTest sets up test environment - should be excluded.
func setupTest(t *testing.T) func() {
	t.Helper()
	// Setup code
	return func() {
		// Cleanup code
	}
}

// TestMain is the test entry point - should be excluded.
func TestMain(m *testing.M) {
	// Setup
	code := m.Run()
	// Teardown
	_ = code
}

// FuzzProcessData is a fuzz test - should be excluded.
func FuzzProcessData(f *testing.F) {
	f.Add("hello")
	f.Add("world")
	f.Fuzz(func(t *testing.T, input string) {
		_ = ProcessData([]string{input})
	})
}
