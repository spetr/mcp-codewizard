package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if store.GetProjectDir() != tmpDir {
		t.Errorf("GetProjectDir = %q, want %q", store.GetProjectDir(), tmpDir)
	}

	// Check that directories were created
	memoryDir := filepath.Join(tmpDir, ".mcp-codewizard", "memory")
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		t.Error("memory directory was not created")
	}

	localDir := filepath.Join(tmpDir, ".mcp-codewizard", "local")
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		t.Error("local directory was not created")
	}
}

func TestStoreSaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Add some data
	note, err := store.AddNote(AddNoteRequest{
		Title:   "Test Note",
		Content: "Test content",
		Tags:    []string{"test", "unit"},
	})
	if err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	// Create new store and load
	store2, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Verify note was loaded
	loadedNote, err := store2.GetNote(note.ID)
	if err != nil {
		t.Fatalf("GetNote failed: %v", err)
	}

	if loadedNote.Title != note.Title {
		t.Errorf("Title = %q, want %q", loadedNote.Title, note.Title)
	}
	if loadedNote.Content != note.Content {
		t.Errorf("Content = %q, want %q", loadedNote.Content, note.Content)
	}
}

func TestStoreClear(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Add some data
	store.AddNote(AddNoteRequest{Title: "N1", Content: "C1"})
	store.AddNote(AddNoteRequest{Title: "N2", Content: "C2"})

	if count := store.CountNotes(); count != 2 {
		t.Errorf("CountNotes before clear = %d, want 2", count)
	}

	// Clear
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if count := store.CountNotes(); count != 0 {
		t.Errorf("CountNotes after clear = %d, want 0", count)
	}
}

func TestStoreExportImport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Add data
	store.AddNote(AddNoteRequest{Title: "Note 1", Content: "Content 1"})
	store.AddDecision(AddDecisionRequest{
		Title:    "Decision 1",
		Context:  "Context 1",
		Decision: "Decision 1",
	})

	// Export
	exported, err := store.Export()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if len(exported.Notes) != 1 {
		t.Errorf("Exported notes = %d, want 1", len(exported.Notes))
	}
	if len(exported.Decisions) != 1 {
		t.Errorf("Exported decisions = %d, want 1", len(exported.Decisions))
	}

	// Clear and import
	store.Clear()

	if err := store.Import(exported); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count := store.CountNotes(); count != 1 {
		t.Errorf("CountNotes after import = %d, want 1", count)
	}
	if count := store.CountDecisions(); count != 1 {
		t.Errorf("CountDecisions after import = %d, want 1", count)
	}
}

func TestStoreContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	ctx := WorkingContext{
		CurrentTask:   "Implement feature X",
		RelevantFiles: []string{"main.go", "handler.go"},
	}

	if err := store.SetContext(ctx); err != nil {
		t.Fatalf("SetContext failed: %v", err)
	}

	retrieved := store.GetContext()
	if retrieved.CurrentTask != ctx.CurrentTask {
		t.Errorf("CurrentTask = %q, want %q", retrieved.CurrentTask, ctx.CurrentTask)
	}
	if len(retrieved.RelevantFiles) != 2 {
		t.Errorf("RelevantFiles count = %d, want 2", len(retrieved.RelevantFiles))
	}
}

func TestStoreSearchHistory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	store.AddSearchToHistory("first search")
	store.AddSearchToHistory("second search")
	store.AddSearchToHistory("third search")

	searches := store.GetRecentSearches(2)
	if len(searches) != 2 {
		t.Errorf("GetRecentSearches returned %d, want 2", len(searches))
	}

	// Most recent should be first
	if searches[0] != "third search" {
		t.Errorf("First search = %q, want %q", searches[0], "third search")
	}
}
