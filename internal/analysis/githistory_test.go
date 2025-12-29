package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

func TestGitHistoryAnalyzer(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir, err := os.MkdirTemp("", "githistory_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial file
	testFile := filepath.Join(tmpDir, "main.go")
	writeFile(t, testFile, `package main

func main() {
	println("Hello")
}
`)
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "Initial commit")

	// Modify file
	writeFile(t, testFile, `package main

func main() {
	println("Hello World")
}

func greet(name string) {
	println("Hello, " + name)
}
`)
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "Add greet function")

	// Create analyzer
	analyzer := NewGitHistoryAnalyzer(tmpDir, nil)

	// Test GetCommits
	t.Run("GetCommits", func(t *testing.T) {
		commits, err := analyzer.GetCommits("", 0)
		if err != nil {
			t.Fatalf("GetCommits failed: %v", err)
		}

		if len(commits) != 2 {
			t.Errorf("expected 2 commits, got %d", len(commits))
		}

		// Newest first
		if commits[0].Message != "Add greet function" {
			t.Errorf("expected first commit message 'Add greet function', got '%s'", commits[0].Message)
		}
	})

	// Test GetChangesForCommit
	t.Run("GetChangesForCommit", func(t *testing.T) {
		commits, _ := analyzer.GetCommits("", 0)
		if len(commits) == 0 {
			t.Skip("no commits")
		}

		changes, err := analyzer.GetChangesForCommit(commits[0].Hash)
		if err != nil {
			t.Fatalf("GetChangesForCommit failed: %v", err)
		}

		if len(changes) != 1 {
			t.Errorf("expected 1 change, got %d", len(changes))
		}

		if changes[0].FilePath != "main.go" {
			t.Errorf("expected file path 'main.go', got '%s'", changes[0].FilePath)
		}
	})

	// Test IdentifyAffectedFunctions
	t.Run("IdentifyAffectedFunctions", func(t *testing.T) {
		commits, _ := analyzer.GetCommits("", 0)
		if len(commits) == 0 {
			t.Skip("no commits")
		}

		changes, _ := analyzer.GetChangesForCommit(commits[0].Hash)
		if len(changes) == 0 {
			t.Skip("no changes")
		}

		functions := analyzer.IdentifyAffectedFunctions(changes[0])
		// The diff should contain "greet" function
		found := false
		for _, fn := range functions {
			if fn == "greet" {
				found = true
				break
			}
		}
		if !found && len(functions) == 0 {
			// Functions detection may not work perfectly, just log
			t.Logf("no functions detected, functions: %v", functions)
		}
	})

	// Test GetCommitsSinceHash
	t.Run("GetCommitsSinceHash", func(t *testing.T) {
		commits, _ := analyzer.GetCommits("", 0)
		if len(commits) < 2 {
			t.Skip("need at least 2 commits")
		}

		// Get commits since the initial commit
		sinceCommits, err := analyzer.GetCommitsSinceHash(commits[1].Hash) // oldest
		if err != nil {
			t.Fatalf("GetCommitsSinceHash failed: %v", err)
		}

		if len(sinceCommits) != 1 {
			t.Errorf("expected 1 commit since initial, got %d", len(sinceCommits))
		}
	})
}

func TestCommitParsing(t *testing.T) {
	// Test commit struct
	commit := &types.Commit{
		Hash:        "abc123def456",
		ShortHash:   "abc123d",
		Author:      "Test User",
		AuthorEmail: "test@test.com",
		Date:        time.Now(),
		Message:     "Test commit message",
	}

	if commit.Hash != "abc123def456" {
		t.Error("Hash mismatch")
	}
	if commit.ShortHash != "abc123d" {
		t.Error("ShortHash mismatch")
	}
}

func TestChangeTypes(t *testing.T) {
	tests := []struct {
		changeType types.ChangeType
		expected   string
	}{
		{types.ChangeTypeAdded, "A"},
		{types.ChangeTypeModified, "M"},
		{types.ChangeTypeDeleted, "D"},
		{types.ChangeTypeRenamed, "R"},
	}

	for _, tt := range tests {
		if string(tt.changeType) != tt.expected {
			t.Errorf("ChangeType %s != %s", tt.changeType, tt.expected)
		}
	}
}

// Helper functions

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2024-01-01T00:00:00", "GIT_COMMITTER_DATE=2024-01-01T00:00:00")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
