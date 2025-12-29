package sqlitevec

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

func TestMemoryStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memorytest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := New()
	if err := store.Init(tmpDir + "/test.db"); err != nil {
		if containsString(err.Error(), "fts5") || containsString(err.Error(), "FTS5") {
			t.Skip("FTS5 not available in this environment")
		}
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("StoreAndGetMemory", func(t *testing.T) {
		now := time.Now()
		memory := &types.MemoryEntry{
			ID:         "mem_001",
			Content:    "This is a test decision about architecture",
			Category:   types.MemoryCategoryDecision,
			Tags:       []string{"architecture", "testing"},
			Importance: 0.8,
			Confidence: 1.0,
			Channel:    "main",
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := store.StoreMemory(memory); err != nil {
			t.Fatalf("StoreMemory failed: %v", err)
		}

		got, err := store.GetMemory("mem_001")
		if err != nil {
			t.Fatalf("GetMemory failed: %v", err)
		}

		if got == nil {
			t.Fatal("expected memory, got nil")
		}

		if got.Content != memory.Content {
			t.Errorf("content mismatch: got %q, want %q", got.Content, memory.Content)
		}

		if got.Category != memory.Category {
			t.Errorf("category mismatch: got %q, want %q", got.Category, memory.Category)
		}
	})

	t.Run("UpdateMemory", func(t *testing.T) {
		memory, _ := store.GetMemory("mem_001")
		memory.Content = "Updated decision content"
		memory.UpdatedAt = time.Now()

		if err := store.UpdateMemory(memory); err != nil {
			t.Fatalf("UpdateMemory failed: %v", err)
		}

		got, err := store.GetMemory("mem_001")
		if err != nil {
			t.Fatal(err)
		}

		if got.Content != "Updated decision content" {
			t.Errorf("content not updated: got %q", got.Content)
		}
	})

	t.Run("SearchMemories", func(t *testing.T) {
		// Add more memories for search
		store.StoreMemory(&types.MemoryEntry{
			ID:         "mem_002",
			Content:    "Database connection pooling configuration",
			Category:   types.MemoryCategoryContext,
			Tags:       []string{"database", "config"},
			Importance: 0.6,
			Confidence: 1.0,
			Channel:    "main",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})

		store.StoreMemory(&types.MemoryEntry{
			ID:         "mem_003",
			Content:    "API authentication using JWT tokens",
			Category:   types.MemoryCategoryFact,
			Tags:       []string{"api", "auth"},
			Importance: 0.9,
			Confidence: 1.0,
			Channel:    "main",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})

		req := &types.MemorySearchRequest{
			Query:   "database",
			Channel: "main",
			Limit:   10,
		}

		results, err := store.SearchMemories(ctx, req)
		if err != nil {
			t.Fatalf("SearchMemories failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("expected search results, got none")
		}
	})

	t.Run("DeleteMemory", func(t *testing.T) {
		if err := store.DeleteMemory("mem_001"); err != nil {
			t.Fatalf("DeleteMemory failed: %v", err)
		}

		got, err := store.GetMemory("mem_001")
		if err != nil {
			t.Fatal(err)
		}

		if got != nil {
			t.Error("expected memory to be deleted")
		}
	})

	t.Run("MemoryStats", func(t *testing.T) {
		stats, err := store.GetMemoryStats("")
		if err != nil {
			t.Fatalf("GetMemoryStats failed: %v", err)
		}

		if stats.TotalMemories < 2 {
			t.Errorf("expected at least 2 memories, got %d", stats.TotalMemories)
		}
	})
}

func TestTodoStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "todotest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := New()
	if err := store.Init(tmpDir + "/test.db"); err != nil {
		if containsString(err.Error(), "fts5") || containsString(err.Error(), "FTS5") {
			t.Skip("FTS5 not available in this environment")
		}
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("CreateAndGetTodo", func(t *testing.T) {
		now := time.Now()
		todo := &types.TodoItem{
			ID:          "todo_001",
			Title:       "Implement user authentication",
			Description: "Add JWT-based auth to the API",
			Status:      types.TodoStatusPending,
			Priority:    types.TodoPriorityHigh,
			Tags:        []string{"auth", "api"},
			Channel:     "main",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := store.StoreTodo(todo); err != nil {
			t.Fatalf("StoreTodo failed: %v", err)
		}

		got, err := store.GetTodo("todo_001")
		if err != nil {
			t.Fatalf("GetTodo failed: %v", err)
		}

		if got == nil {
			t.Fatal("expected todo, got nil")
		}

		if got.Title != todo.Title {
			t.Errorf("title mismatch: got %q, want %q", got.Title, todo.Title)
		}

		if got.Status != types.TodoStatusPending {
			t.Errorf("status mismatch: got %q, want pending", got.Status)
		}
	})

	t.Run("UpdateTodoStatus", func(t *testing.T) {
		if err := store.UpdateTodoStatus("todo_001", types.TodoStatusInProgress); err != nil {
			t.Fatalf("UpdateTodoStatus failed: %v", err)
		}

		got, _ := store.GetTodo("todo_001")
		if got.Status != types.TodoStatusInProgress {
			t.Errorf("status not updated: got %q", got.Status)
		}
	})

	t.Run("CompleteTodo", func(t *testing.T) {
		if err := store.CompleteTodo("todo_001"); err != nil {
			t.Fatalf("CompleteTodo failed: %v", err)
		}

		got, _ := store.GetTodo("todo_001")
		if got.Status != types.TodoStatusCompleted {
			t.Errorf("status not completed: got %q", got.Status)
		}
		if got.CompletedAt == nil {
			t.Error("expected CompletedAt to be set")
		}
	})

	t.Run("ListTodos", func(t *testing.T) {
		// Add more todos
		store.StoreTodo(&types.TodoItem{
			ID:        "todo_002",
			Title:     "Fix database connection bug",
			Status:    types.TodoStatusPending,
			Priority:  types.TodoPriorityUrgent,
			Channel:   "main",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})

		store.StoreTodo(&types.TodoItem{
			ID:        "todo_003",
			Title:     "Write documentation",
			Status:    types.TodoStatusPending,
			Priority:  types.TodoPriorityLow,
			Channel:   "main",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})

		todos, err := store.ListTodos(ctx, "main", []types.TodoStatus{types.TodoStatusPending}, 10)
		if err != nil {
			t.Fatalf("ListTodos failed: %v", err)
		}

		if len(todos) < 2 {
			t.Errorf("expected at least 2 pending todos, got %d", len(todos))
		}
	})

	t.Run("SearchTodos", func(t *testing.T) {
		req := &types.TodoSearchRequest{
			Query:   "database",
			Channel: "main",
			Limit:   10,
		}

		results, err := store.SearchTodos(ctx, req)
		if err != nil {
			t.Fatalf("SearchTodos failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("expected search results, got none")
		}
	})

	t.Run("TodoHierarchy", func(t *testing.T) {
		// Create parent-child relationship
		store.StoreTodo(&types.TodoItem{
			ID:        "todo_parent",
			Title:     "Main feature task",
			Status:    types.TodoStatusInProgress,
			Priority:  types.TodoPriorityHigh,
			Channel:   "main",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})

		store.StoreTodo(&types.TodoItem{
			ID:        "todo_child1",
			Title:     "Subtask 1",
			Status:    types.TodoStatusPending,
			Priority:  types.TodoPriorityMedium,
			ParentID:  "todo_parent",
			Channel:   "main",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})

		store.StoreTodo(&types.TodoItem{
			ID:        "todo_child2",
			Title:     "Subtask 2",
			Status:    types.TodoStatusPending,
			Priority:  types.TodoPriorityMedium,
			ParentID:  "todo_parent",
			Channel:   "main",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})

		children, err := store.GetTodoChildren("todo_parent")
		if err != nil {
			t.Fatalf("GetTodoChildren failed: %v", err)
		}

		if len(children) != 2 {
			t.Errorf("expected 2 children, got %d", len(children))
		}
	})

	t.Run("DeleteTodo", func(t *testing.T) {
		if err := store.DeleteTodo("todo_003"); err != nil {
			t.Fatalf("DeleteTodo failed: %v", err)
		}

		got, err := store.GetTodo("todo_003")
		if err != nil {
			t.Fatal(err)
		}

		if got != nil {
			t.Error("expected todo to be deleted")
		}
	})

	t.Run("TodoStats", func(t *testing.T) {
		stats, err := store.GetTodoStats("")
		if err != nil {
			t.Fatalf("GetTodoStats failed: %v", err)
		}

		if stats.TotalTodos < 4 {
			t.Errorf("expected at least 4 todos, got %d", stats.TotalTodos)
		}
	})
}

func TestCheckpointRestore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := New()
	if err := store.Init(tmpDir + "/test.db"); err != nil {
		if containsString(err.Error(), "fts5") || containsString(err.Error(), "FTS5") {
			t.Skip("FTS5 not available in this environment")
		}
		t.Fatal(err)
	}
	defer store.Close()

	// Store some memories
	store.StoreMemory(&types.MemoryEntry{
		ID:         "mem_1",
		Content:    "First memory",
		Category:   types.MemoryCategoryNote,
		Channel:    "main",
		Importance: 0.5,
		Confidence: 1.0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	store.StoreMemory(&types.MemoryEntry{
		ID:         "mem_2",
		Content:    "Second memory",
		Category:   types.MemoryCategoryNote,
		Channel:    "main",
		Importance: 0.5,
		Confidence: 1.0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	// Create checkpoint
	checkpoint := &types.MemoryCheckpoint{
		ID:          "chk_1",
		Name:        "Before changes",
		Description: "Checkpoint before major changes",
		Channel:     "main",
		CreatedAt:   time.Now(),
	}

	if err := store.CreateCheckpoint(checkpoint); err != nil {
		t.Fatalf("CreateCheckpoint failed: %v", err)
	}

	// Get checkpoint and verify
	got, err := store.GetCheckpoint("chk_1")
	if err != nil {
		t.Fatalf("GetCheckpoint failed: %v", err)
	}

	if got.Name != "Before changes" {
		t.Errorf("name mismatch: got %q", got.Name)
	}

	// List checkpoints
	checkpoints, err := store.ListCheckpoints("main", 10)
	if err != nil {
		t.Fatalf("ListCheckpoints failed: %v", err)
	}

	if len(checkpoints) < 1 {
		t.Error("expected at least 1 checkpoint")
	}
}

func TestMemoryTTL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ttltest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := New()
	if err := store.Init(tmpDir + "/test.db"); err != nil {
		if containsString(err.Error(), "fts5") || containsString(err.Error(), "FTS5") {
			t.Skip("FTS5 not available in this environment")
		}
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now()
	pastExpiry := now.Add(-24 * time.Hour) // Expired 1 day ago

	// Store expired memory
	store.StoreMemory(&types.MemoryEntry{
		ID:         "mem_expired",
		Content:    "This should be pruned",
		Category:   types.MemoryCategoryNote,
		Channel:    "main",
		Importance: 0.5,
		Confidence: 1.0,
		ExpiresAt:  &pastExpiry,
		CreatedAt:  now.Add(-48 * time.Hour),
		UpdatedAt:  now.Add(-48 * time.Hour),
	})

	// Store valid memory
	futureExpiry := now.Add(24 * time.Hour)
	store.StoreMemory(&types.MemoryEntry{
		ID:         "mem_valid",
		Content:    "This should remain",
		Category:   types.MemoryCategoryNote,
		Channel:    "main",
		Importance: 0.5,
		Confidence: 1.0,
		ExpiresAt:  &futureExpiry,
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// Prune expired
	pruned, err := store.PruneExpired()
	if err != nil {
		t.Fatalf("PruneExpired failed: %v", err)
	}

	if pruned != 1 {
		t.Errorf("expected 1 pruned memory, got %d", pruned)
	}

	// Verify expired is gone
	got, _ := store.GetMemory("mem_expired")
	if got != nil {
		t.Error("expired memory should have been pruned")
	}

	// Verify valid remains
	got, _ = store.GetMemory("mem_valid")
	if got == nil {
		t.Error("valid memory should remain")
	}
}
