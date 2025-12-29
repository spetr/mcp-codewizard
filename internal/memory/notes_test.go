package memory

import (
	"os"
	"testing"
)

func TestAddNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	note, err := store.AddNote(AddNoteRequest{
		Title:   "Test Note",
		Content: "This is test content",
		Tags:    []string{"test", "unit"},
	})
	if err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	if note.ID == "" {
		t.Error("Note ID is empty")
	}
	if note.Title != "Test Note" {
		t.Errorf("Title = %q, want %q", note.Title, "Test Note")
	}
	if note.Author != "ai" {
		t.Errorf("Author = %q, want %q", note.Author, "ai")
	}
	if len(note.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(note.Tags))
	}
}

func TestAddNoteValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Missing title
	_, err = store.AddNote(AddNoteRequest{
		Content: "Content without title",
	})
	if err == nil {
		t.Error("Expected error for missing title")
	}

	// Missing content
	_, err = store.AddNote(AddNoteRequest{
		Title: "Title without content",
	})
	if err == nil {
		t.Error("Expected error for missing content")
	}
}

func TestGetNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	note, err := store.AddNote(AddNoteRequest{
		Title:   "Test Note",
		Content: "Content",
	})
	if err != nil {
		t.Fatal(err)
	}

	retrieved, err := store.GetNote(note.ID)
	if err != nil {
		t.Fatalf("GetNote failed: %v", err)
	}

	if retrieved.ID != note.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, note.ID)
	}

	// Non-existent note
	_, err = store.GetNote("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent note")
	}
}

func TestUpdateNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	note, err := store.AddNote(AddNoteRequest{
		Title:   "Original Title",
		Content: "Original content",
	})
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Updated Title"
	updated, err := store.UpdateNote(UpdateNoteRequest{
		ID:    note.ID,
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Content != note.Content {
		t.Errorf("Content changed unexpectedly")
	}
}

func TestDeleteNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	note, err := store.AddNote(AddNoteRequest{
		Title:   "To Delete",
		Content: "Content",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteNote(note.ID); err != nil {
		t.Fatalf("DeleteNote failed: %v", err)
	}

	_, err = store.GetNote(note.ID)
	if err == nil {
		t.Error("Note should have been deleted")
	}

	// Delete non-existent
	err = store.DeleteNote("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent note")
	}
}

func TestListNotes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Add multiple notes
	_, err = store.AddNote(AddNoteRequest{
		Title:    "Note 1",
		Content:  "Content 1",
		Tags:     []string{"api"},
		FilePath: "api/handler.go",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.AddNote(AddNoteRequest{
		Title:   "Note 2",
		Content: "Content 2",
		Tags:    []string{"database"},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.AddNote(AddNoteRequest{
		Title:   "Note 3",
		Content: "Content 3 with special keyword",
		Tags:    []string{"api", "important"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// List all
	notes := store.ListNotes(ListNotesRequest{})
	if len(notes) != 3 {
		t.Errorf("ListNotes returned %d notes, want 3", len(notes))
	}

	// Filter by tag
	notes = store.ListNotes(ListNotesRequest{Tags: []string{"api"}})
	if len(notes) != 2 {
		t.Errorf("ListNotes with tag filter returned %d notes, want 2", len(notes))
	}

	// Filter by file path
	notes = store.ListNotes(ListNotesRequest{FilePath: "api/"})
	if len(notes) != 1 {
		t.Errorf("ListNotes with file filter returned %d notes, want 1", len(notes))
	}

	// Search
	notes = store.ListNotes(ListNotesRequest{Search: "keyword"})
	if len(notes) != 1 {
		t.Errorf("ListNotes with search returned %d notes, want 1", len(notes))
	}

	// Limit
	notes = store.ListNotes(ListNotesRequest{Limit: 2})
	if len(notes) != 2 {
		t.Errorf("ListNotes with limit returned %d notes, want 2", len(notes))
	}
}

func TestCountNotes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(StoreConfig{ProjectDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	if count := store.CountNotes(); count != 0 {
		t.Errorf("CountNotes = %d, want 0", count)
	}

	store.AddNote(AddNoteRequest{Title: "N1", Content: "C1"})
	store.AddNote(AddNoteRequest{Title: "N2", Content: "C2"})

	if count := store.CountNotes(); count != 2 {
		t.Errorf("CountNotes = %d, want 2", count)
	}
}
