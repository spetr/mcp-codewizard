package sqlitevec

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

// containsString checks if a string contains a substring (case-insensitive).
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func TestGitHistoryStore(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "githistory_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Create store
	store := New()
	if err := store.Init(tmpFile.Name()); err != nil {
		// Skip if FTS5 is not available (common in test environments)
		if containsString(err.Error(), "fts5") || containsString(err.Error(), "FTS5") {
			t.Skip("FTS5 not available in this environment")
		}
		t.Fatalf("failed to init store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test StoreCommits
	t.Run("StoreCommits", func(t *testing.T) {
		commits := []*types.Commit{
			{
				Hash:        "abc123def456789012345678901234567890abcd",
				ShortHash:   "abc123d",
				Author:      "Test User",
				AuthorEmail: "test@test.com",
				Date:        time.Now().Add(-1 * time.Hour),
				Message:     "First commit",
				ParentHash:  "",
			},
			{
				Hash:        "def456789012345678901234567890abcdef1234",
				ShortHash:   "def4567",
				Author:      "Test User",
				AuthorEmail: "test@test.com",
				Date:        time.Now(),
				Message:     "Second commit",
				ParentHash:  "abc123def456789012345678901234567890abcd",
			},
		}

		err := store.StoreCommits(commits)
		if err != nil {
			t.Fatalf("StoreCommits failed: %v", err)
		}
	})

	// Test GetCommit
	t.Run("GetCommit", func(t *testing.T) {
		commit, err := store.GetCommit("abc123def456789012345678901234567890abcd")
		if err != nil {
			t.Fatalf("GetCommit failed: %v", err)
		}

		if commit == nil {
			t.Fatal("commit not found")
		}

		if commit.Author != "Test User" {
			t.Errorf("expected author 'Test User', got '%s'", commit.Author)
		}
		if commit.Message != "First commit" {
			t.Errorf("expected message 'First commit', got '%s'", commit.Message)
		}
	})

	// Test StoreChanges
	t.Run("StoreChanges", func(t *testing.T) {
		changes := []*types.Change{
			{
				ID:          "change1",
				CommitHash:  "abc123def456789012345678901234567890abcd",
				FilePath:    "main.go",
				ChangeType:  types.ChangeTypeAdded,
				DiffContent: "+package main\n+\n+func main() {}",
				Additions:   3,
				Deletions:   0,
			},
		}

		err := store.StoreChanges(changes)
		if err != nil {
			t.Fatalf("StoreChanges failed: %v", err)
		}
	})

	// Test GetChangesByCommit
	t.Run("GetChangesByCommit", func(t *testing.T) {
		changes, err := store.GetChangesByCommit("abc123def456789012345678901234567890abcd")
		if err != nil {
			t.Fatalf("GetChangesByCommit failed: %v", err)
		}

		if len(changes) != 1 {
			t.Errorf("expected 1 change, got %d", len(changes))
		}

		if len(changes) > 0 && changes[0].FilePath != "main.go" {
			t.Errorf("expected file 'main.go', got '%s'", changes[0].FilePath)
		}
	})

	// Test SetLastIndexedCommit and GetLastIndexedCommit
	t.Run("LastIndexedCommit", func(t *testing.T) {
		err := store.SetLastIndexedCommit("abc123def456789012345678901234567890abcd")
		if err != nil {
			t.Fatalf("SetLastIndexedCommit failed: %v", err)
		}

		lastHash, err := store.GetLastIndexedCommit()
		if err != nil {
			t.Fatalf("GetLastIndexedCommit failed: %v", err)
		}

		if lastHash != "abc123def456789012345678901234567890abcd" {
			t.Errorf("expected hash 'abc123...', got '%s'", lastHash)
		}
	})

	// Test StoreChunkHistory
	t.Run("StoreChunkHistory", func(t *testing.T) {
		entries := []*types.ChunkHistoryEntry{
			{
				ChunkID:     "chunk1",
				CommitHash:  "abc123def456789012345678901234567890abcd",
				ChangeType:  "M",
				DiffSummary: "Modified function",
				Date:        time.Now(),
				Author:      "Test User",
			},
		}

		err := store.StoreChunkHistory(entries)
		if err != nil {
			t.Fatalf("StoreChunkHistory failed: %v", err)
		}
	})

	// Test GetChunkHistory
	t.Run("GetChunkHistory", func(t *testing.T) {
		history, err := store.GetChunkHistory("chunk1", 10)
		if err != nil {
			t.Fatalf("GetChunkHistory failed: %v", err)
		}

		if len(history) != 1 {
			t.Errorf("expected 1 history entry, got %d", len(history))
		}
	})

	// Test GetGitHistoryStats
	t.Run("GetGitHistoryStats", func(t *testing.T) {
		stats, err := store.GetGitHistoryStats()
		if err != nil {
			t.Fatalf("GetGitHistoryStats failed: %v", err)
		}

		if stats.TotalCommits != 2 {
			t.Errorf("expected 2 commits, got %d", stats.TotalCommits)
		}
		if stats.TotalChanges != 1 {
			t.Errorf("expected 1 change, got %d", stats.TotalChanges)
		}
	})

	// Test SearchHistory
	t.Run("SearchHistory", func(t *testing.T) {
		req := &types.HistorySearchRequest{
			Query: "commit",
			Limit: 10,
		}

		results, err := store.SearchHistory(ctx, req)
		if err != nil {
			t.Fatalf("SearchHistory failed: %v", err)
		}

		// Results may be empty if no embeddings, but should not error
		t.Logf("found %d results", len(results))
	})
}

func TestGitHistoryStoreEmpty(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "githistory_empty_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Create store
	store := New()
	if err := store.Init(tmpFile.Name()); err != nil {
		// Skip if FTS5 is not available (common in test environments)
		if containsString(err.Error(), "fts5") || containsString(err.Error(), "FTS5") {
			t.Skip("FTS5 not available in this environment")
		}
		t.Fatalf("failed to init store: %v", err)
	}
	defer store.Close()

	// Test GetLastIndexedCommit on empty store
	t.Run("GetLastIndexedCommit_Empty", func(t *testing.T) {
		hash, err := store.GetLastIndexedCommit()
		if err != nil {
			t.Fatalf("GetLastIndexedCommit failed: %v", err)
		}
		if hash != "" {
			t.Errorf("expected empty hash, got '%s'", hash)
		}
	})

	// Test GetGitHistoryStats on empty store
	t.Run("GetGitHistoryStats_Empty", func(t *testing.T) {
		stats, err := store.GetGitHistoryStats()
		if err != nil {
			t.Fatalf("GetGitHistoryStats failed: %v", err)
		}
		if stats.TotalCommits != 0 {
			t.Errorf("expected 0 commits, got %d", stats.TotalCommits)
		}
	})
}
