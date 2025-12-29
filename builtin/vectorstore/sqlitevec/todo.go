// Package sqlitevec implements todo/task storage and search using SQLite.
package sqlitevec

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

// createTodoSchema creates tables for todo storage.
func (s *Store) createTodoSchema() error {
	// Todos table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			priority TEXT NOT NULL DEFAULT 'medium',
			tags TEXT,
			parent_id TEXT,
			channel TEXT DEFAULT 'main',
			related_memories TEXT,
			related_chunks TEXT,
			related_commits TEXT,
			blocked_by TEXT,
			progress INTEGER DEFAULT 0,
			effort TEXT,
			actual_effort TEXT,
			session_id TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			due_date INTEGER,
			started_at INTEGER,
			completed_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create todos table: %w", err)
	}

	// Indexes for todos
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_todos_channel ON todos(channel)",
		"CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status)",
		"CREATE INDEX IF NOT EXISTS idx_todos_priority ON todos(priority)",
		"CREATE INDEX IF NOT EXISTS idx_todos_parent ON todos(parent_id)",
		"CREATE INDEX IF NOT EXISTS idx_todos_created ON todos(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_todos_due ON todos(due_date)",
		"CREATE INDEX IF NOT EXISTS idx_todos_session ON todos(session_id)",
	}
	for _, idx := range indexes {
		if _, err := s.db.Exec(idx); err != nil {
			return err
		}
	}

	// FTS5 for todo search
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS todos_fts USING fts5(
			id,
			title,
			description,
			content='todos',
			content_rowid='rowid',
			tokenize='porter unicode61'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create todos_fts table: %w", err)
	}

	// Triggers to keep FTS in sync
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS todos_ai AFTER INSERT ON todos BEGIN
			INSERT INTO todos_fts(rowid, id, title, description)
			VALUES (new.rowid, new.id, new.title, new.description);
		END
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS todos_ad AFTER DELETE ON todos BEGIN
			INSERT INTO todos_fts(todos_fts, rowid, id, title, description)
			VALUES('delete', old.rowid, old.id, old.title, old.description);
		END
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS todos_au AFTER UPDATE ON todos BEGIN
			INSERT INTO todos_fts(todos_fts, rowid, id, title, description)
			VALUES('delete', old.rowid, old.id, old.title, old.description);
			INSERT INTO todos_fts(rowid, id, title, description)
			VALUES (new.rowid, new.id, new.title, new.description);
		END
	`)
	if err != nil {
		return err
	}

	return nil
}

// createTodoEmbeddingsTable creates vector table for todo embeddings.
func (s *Store) createTodoEmbeddingsTable(dimensions int) error {
	_, err := s.db.Exec(fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS todo_embeddings USING vec0(
			todo_id TEXT PRIMARY KEY,
			embedding float[%d]
		)
	`, dimensions))
	if err != nil {
		return fmt.Errorf("failed to create todo_embeddings table: %w", err)
	}
	return nil
}

// StoreTodo stores a single todo item.
func (s *Store) StoreTodo(todo *types.TodoItem) error {
	return s.StoreTodos([]*types.TodoItem{todo})
}

// StoreTodos stores multiple todo items.
func (s *Store) StoreTodos(todos []*types.TodoItem) error {
	if len(todos) == 0 {
		return nil
	}

	// Create embedding table if we have embeddings
	for _, t := range todos {
		if len(t.Embedding) > 0 {
			if err := s.createTodoEmbeddingsTable(len(t.Embedding)); err != nil {
				return err
			}
			break
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	todoStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO todos
		(id, title, description, status, priority, tags, parent_id, channel,
		 related_memories, related_chunks, related_commits, blocked_by,
		 progress, effort, actual_effort, session_id,
		 created_at, updated_at, due_date, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer todoStmt.Close()

	embStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO todo_embeddings (todo_id, embedding)
		VALUES (?, ?)
	`)
	if err != nil {
		embStmt = nil
	} else {
		defer embStmt.Close()
	}

	for _, t := range todos {
		var tagsJSON, relMemJSON, relChunksJSON, relCommitsJSON, blockedByJSON string
		var dueDate, startedAt, completedAt sql.NullInt64

		if len(t.Tags) > 0 {
			b, _ := json.Marshal(t.Tags)
			tagsJSON = string(b)
		}
		if len(t.RelatedMemories) > 0 {
			b, _ := json.Marshal(t.RelatedMemories)
			relMemJSON = string(b)
		}
		if len(t.RelatedChunks) > 0 {
			b, _ := json.Marshal(t.RelatedChunks)
			relChunksJSON = string(b)
		}
		if len(t.RelatedCommits) > 0 {
			b, _ := json.Marshal(t.RelatedCommits)
			relCommitsJSON = string(b)
		}
		if len(t.BlockedBy) > 0 {
			b, _ := json.Marshal(t.BlockedBy)
			blockedByJSON = string(b)
		}
		if t.DueDate != nil {
			dueDate.Int64 = t.DueDate.Unix()
			dueDate.Valid = true
		}
		if t.StartedAt != nil {
			startedAt.Int64 = t.StartedAt.Unix()
			startedAt.Valid = true
		}
		if t.CompletedAt != nil {
			completedAt.Int64 = t.CompletedAt.Unix()
			completedAt.Valid = true
		}

		_, err := todoStmt.Exec(
			t.ID, t.Title, t.Description, string(t.Status), string(t.Priority),
			tagsJSON, t.ParentID, t.Channel,
			relMemJSON, relChunksJSON, relCommitsJSON, blockedByJSON,
			t.Progress, t.Effort, t.ActualEffort, t.SessionID,
			t.CreatedAt.Unix(), t.UpdatedAt.Unix(), dueDate, startedAt, completedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to store todo %s: %w", t.ID, err)
		}

		// Store embedding if present
		if embStmt != nil && len(t.Embedding) > 0 {
			embBytes := floatsToBytes(t.Embedding)
			_, err := embStmt.Exec(t.ID, embBytes)
			if err != nil {
				return fmt.Errorf("failed to store embedding for todo %s: %w", t.ID, err)
			}
		}
	}

	return tx.Commit()
}

// GetTodo retrieves a todo by ID.
func (s *Store) GetTodo(id string) (*types.TodoItem, error) {
	row := s.db.QueryRow(`
		SELECT id, title, description, status, priority, tags, parent_id, channel,
		       related_memories, related_chunks, related_commits, blocked_by,
		       progress, effort, actual_effort, session_id,
		       created_at, updated_at, due_date, started_at, completed_at
		FROM todos WHERE id = ?
	`, id)

	return scanTodo(row)
}

// UpdateTodo updates an existing todo.
func (s *Store) UpdateTodo(todo *types.TodoItem) error {
	todo.UpdatedAt = time.Now()
	return s.StoreTodo(todo)
}

// DeleteTodo deletes a todo by ID.
func (s *Store) DeleteTodo(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete embedding
	_, _ = tx.Exec("DELETE FROM todo_embeddings WHERE todo_id = ?", id)

	// Delete todo (FTS will be updated by trigger)
	_, err = tx.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Also delete children
	_, _ = tx.Exec("DELETE FROM todos WHERE parent_id = ?", id)

	return tx.Commit()
}

// UpdateTodoStatus updates just the status of a todo.
func (s *Store) UpdateTodoStatus(id string, status types.TodoStatus) error {
	now := time.Now().Unix()

	var startedAt, completedAt sql.NullInt64
	if status == types.TodoStatusInProgress {
		startedAt.Int64 = now
		startedAt.Valid = true
	}
	if status == types.TodoStatusCompleted {
		completedAt.Int64 = now
		completedAt.Valid = true
	}

	_, err := s.db.Exec(`
		UPDATE todos
		SET status = ?, updated_at = ?,
		    started_at = COALESCE(started_at, ?),
		    completed_at = ?
		WHERE id = ?
	`, string(status), now, startedAt, completedAt, id)

	return err
}

// CompleteTodo marks a todo as completed.
func (s *Store) CompleteTodo(id string) error {
	return s.UpdateTodoStatus(id, types.TodoStatusCompleted)
}

// StartTodo marks a todo as in progress.
func (s *Store) StartTodo(id string) error {
	return s.UpdateTodoStatus(id, types.TodoStatusInProgress)
}

// DeleteTodosByChannel deletes all todos in a channel.
func (s *Store) DeleteTodosByChannel(channel string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get todo IDs first
	rows, err := tx.Query("SELECT id FROM todos WHERE channel = ?", channel)
	if err != nil {
		return err
	}

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()

	// Delete embeddings
	for _, id := range ids {
		_, _ = tx.Exec("DELETE FROM todo_embeddings WHERE todo_id = ?", id)
	}

	// Delete todos
	_, err = tx.Exec("DELETE FROM todos WHERE channel = ?", channel)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetTodoChildren returns children of a todo.
func (s *Store) GetTodoChildren(parentID string) ([]*types.TodoItem, error) {
	rows, err := s.db.Query(`
		SELECT id, title, description, status, priority, tags, parent_id, channel,
		       related_memories, related_chunks, related_commits, blocked_by,
		       progress, effort, actual_effort, session_id,
		       created_at, updated_at, due_date, started_at, completed_at
		FROM todos WHERE parent_id = ?
		ORDER BY priority DESC, created_at ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTodos(rows)
}

// MoveTodo moves a todo to a new parent.
func (s *Store) MoveTodo(id string, newParentID string) error {
	_, err := s.db.Exec(`
		UPDATE todos SET parent_id = ?, updated_at = ? WHERE id = ?
	`, newParentID, time.Now().Unix(), id)
	return err
}

// SearchTodos performs hybrid search on todos.
func (s *Store) SearchTodos(ctx context.Context, req *types.TodoSearchRequest) ([]*types.TodoSearchResult, error) {
	candidateLimit := req.Limit * 3
	if candidateLimit < 50 {
		candidateLimit = 50
	}

	// Collect candidates from both vector and BM25 search
	candidates := make(map[string]*types.TodoSearchResult)

	// Vector search
	if len(req.QueryVec) > 0 {
		results, err := s.vectorSearchTodos(ctx, req.QueryVec, candidateLimit, req)
		if err == nil {
			for _, r := range results {
				candidates[r.Todo.ID] = r
			}
		}
	}

	// BM25 search
	if req.Query != "" {
		results, err := s.bm25SearchTodos(ctx, req.Query, candidateLimit, req)
		if err == nil {
			for _, r := range results {
				if existing, ok := candidates[r.Todo.ID]; ok {
					existing.VectorScore = (existing.VectorScore + r.VectorScore) / 2
				} else {
					candidates[r.Todo.ID] = r
				}
			}
		}
	}

	// If no search query, just list with filters
	if len(req.QueryVec) == 0 && req.Query == "" {
		todos, err := s.ListTodos(ctx, req.Channel, req.Statuses, candidateLimit)
		if err != nil {
			return nil, err
		}
		for _, t := range todos {
			candidates[t.ID] = &types.TodoSearchResult{
				Todo:        t,
				Score:       1.0,
				VectorScore: 1.0,
			}
		}
	}

	// Apply priority and urgency scoring
	now := time.Now()
	var results []*types.TodoSearchResult

	for _, r := range candidates {
		// Priority bonus
		switch r.Todo.Priority {
		case types.TodoPriorityUrgent:
			r.Score += 0.4
		case types.TodoPriorityHigh:
			r.Score += 0.2
		case types.TodoPriorityMedium:
			r.Score += 0.1
		}

		// Due date urgency
		if r.Todo.DueDate != nil {
			daysUntilDue := r.Todo.DueDate.Sub(now).Hours() / 24
			if daysUntilDue < 0 {
				r.Score += 0.5 // Overdue!
			} else if daysUntilDue < 1 {
				r.Score += 0.3 // Due today
			} else if daysUntilDue < 7 {
				r.Score += 0.1 // Due this week
			}
		}

		results = append(results, r)
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	// Include children if requested
	if req.IncludeChildren {
		for _, r := range results {
			children, _ := s.GetTodoChildren(r.Todo.ID)
			r.Children = children
		}
	}

	return results, nil
}

// vectorSearchTodos performs vector similarity search on todos.
func (s *Store) vectorSearchTodos(ctx context.Context, queryVec []float32, limit int, req *types.TodoSearchRequest) ([]*types.TodoSearchResult, error) {
	embBytes := floatsToBytes(queryVec)

	query := `
		SELECT
			te.todo_id,
			vec_distance_cosine(te.embedding, ?) as distance
		FROM todo_embeddings te
		JOIN todos t ON te.todo_id = t.id
		WHERE 1=1
	`
	args := []any{embBytes}

	// Apply filters
	query, args = s.applyTodoFilters(query, args, req)

	query += " ORDER BY distance ASC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*types.TodoSearchResult
	for rows.Next() {
		var id string
		var distance float64
		if err := rows.Scan(&id, &distance); err != nil {
			return nil, err
		}

		todo, err := s.GetTodo(id)
		if err != nil || todo == nil {
			continue
		}

		score := float32(1.0 - distance)
		results = append(results, &types.TodoSearchResult{
			Todo:        todo,
			Score:       score,
			VectorScore: score,
		})
	}

	return results, nil
}

// bm25SearchTodos performs BM25 search on todos.
func (s *Store) bm25SearchTodos(ctx context.Context, query string, limit int, req *types.TodoSearchRequest) ([]*types.TodoSearchResult, error) {
	sqlQuery := `
		SELECT fts.id, bm25(todos_fts) as score
		FROM todos_fts fts
		JOIN todos t ON fts.id = t.id
		WHERE todos_fts MATCH ?
	`
	args := []any{escapeFTSQuery(query)}

	// Apply filters
	sqlQuery, args = s.applyTodoFilters(sqlQuery, args, req)

	sqlQuery += " ORDER BY score LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*types.TodoSearchResult
	for rows.Next() {
		var id string
		var score float64
		if err := rows.Scan(&id, &score); err != nil {
			return nil, err
		}

		todo, err := s.GetTodo(id)
		if err != nil || todo == nil {
			continue
		}

		normalizedScore := float32(1.0 / (1.0 + math.Abs(score)))
		results = append(results, &types.TodoSearchResult{
			Todo:        todo,
			Score:       normalizedScore,
			VectorScore: normalizedScore,
		})
	}

	return results, nil
}

// applyTodoFilters adds filter clauses to a todo query.
func (s *Store) applyTodoFilters(query string, args []any, req *types.TodoSearchRequest) (string, []any) {
	if req == nil {
		return query, args
	}

	if req.Channel != "" {
		query += " AND t.channel = ?"
		args = append(args, req.Channel)
	}

	if len(req.Statuses) > 0 {
		placeholders := make([]string, len(req.Statuses))
		for i, s := range req.Statuses {
			placeholders[i] = "?"
			args = append(args, string(s))
		}
		query += " AND t.status IN (" + strings.Join(placeholders, ",") + ")"
	} else if !req.IncludeCompleted {
		query += " AND t.status NOT IN ('completed', 'cancelled')"
	}

	if len(req.Priorities) > 0 {
		placeholders := make([]string, len(req.Priorities))
		for i, p := range req.Priorities {
			placeholders[i] = "?"
			args = append(args, string(p))
		}
		query += " AND t.priority IN (" + strings.Join(placeholders, ",") + ")"
	}

	if req.ParentID != "" {
		query += " AND t.parent_id = ?"
		args = append(args, req.ParentID)
	} else {
		// Top-level only by default
		query += " AND (t.parent_id IS NULL OR t.parent_id = '')"
	}

	if req.DueBefore != nil {
		query += " AND t.due_date <= ?"
		args = append(args, req.DueBefore.Unix())
	}

	if req.DueAfter != nil {
		query += " AND t.due_date >= ?"
		args = append(args, req.DueAfter.Unix())
	}

	if req.CreatedAfter != nil {
		query += " AND t.created_at >= ?"
		args = append(args, req.CreatedAfter.Unix())
	}

	// Tag filtering
	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			query += " AND t.tags LIKE ?"
			args = append(args, "%"+tag+"%")
		}
	}

	return query, args
}

// ListTodos lists todos with basic filters.
func (s *Store) ListTodos(ctx context.Context, channel string, statuses []types.TodoStatus, limit int) ([]*types.TodoItem, error) {
	query := `
		SELECT id, title, description, status, priority, tags, parent_id, channel,
		       related_memories, related_chunks, related_commits, blocked_by,
		       progress, effort, actual_effort, session_id,
		       created_at, updated_at, due_date, started_at, completed_at
		FROM todos WHERE 1=1
	`
	args := []any{}

	if channel != "" {
		query += " AND channel = ?"
		args = append(args, channel)
	}

	if len(statuses) > 0 {
		placeholders := make([]string, len(statuses))
		for i, s := range statuses {
			placeholders[i] = "?"
			args = append(args, string(s))
		}
		query += " AND status IN (" + strings.Join(placeholders, ",") + ")"
	}

	// Only top-level
	query += " AND (parent_id IS NULL OR parent_id = '')"

	query += " ORDER BY CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END, created_at DESC"
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTodos(rows)
}

// GetOverdueTodos returns overdue todos.
func (s *Store) GetOverdueTodos(channel string) ([]*types.TodoItem, error) {
	now := time.Now().Unix()

	query := `
		SELECT id, title, description, status, priority, tags, parent_id, channel,
		       related_memories, related_chunks, related_commits, blocked_by,
		       progress, effort, actual_effort, session_id,
		       created_at, updated_at, due_date, started_at, completed_at
		FROM todos
		WHERE due_date < ? AND status NOT IN ('completed', 'cancelled')
	`
	args := []any{now}

	if channel != "" {
		query += " AND channel = ?"
		args = append(args, channel)
	}

	query += " ORDER BY due_date ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTodos(rows)
}

// GetTodosByPriority returns todos by priority.
func (s *Store) GetTodosByPriority(channel string, priority types.TodoPriority, limit int) ([]*types.TodoItem, error) {
	query := `
		SELECT id, title, description, status, priority, tags, parent_id, channel,
		       related_memories, related_chunks, related_commits, blocked_by,
		       progress, effort, actual_effort, session_id,
		       created_at, updated_at, due_date, started_at, completed_at
		FROM todos
		WHERE priority = ? AND status NOT IN ('completed', 'cancelled')
	`
	args := []any{string(priority)}

	if channel != "" {
		query += " AND channel = ?"
		args = append(args, channel)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTodos(rows)
}

// GetBlockedTodos returns todos that are blocked.
func (s *Store) GetBlockedTodos(channel string) ([]*types.TodoItem, error) {
	query := `
		SELECT id, title, description, status, priority, tags, parent_id, channel,
		       related_memories, related_chunks, related_commits, blocked_by,
		       progress, effort, actual_effort, session_id,
		       created_at, updated_at, due_date, started_at, completed_at
		FROM todos
		WHERE status = 'blocked'
	`
	args := []any{}

	if channel != "" {
		query += " AND channel = ?"
		args = append(args, channel)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTodos(rows)
}

// GetTodoStats returns statistics about todos.
func (s *Store) GetTodoStats(channel string) (*types.TodoStats, error) {
	stats := &types.TodoStats{
		TodosByStatus:   make(map[types.TodoStatus]int),
		TodosByPriority: make(map[types.TodoPriority]int),
		TodosByChannel:  make(map[string]int),
	}

	whereClause := ""
	args := []any{}
	if channel != "" {
		whereClause = " WHERE channel = ?"
		args = append(args, channel)
	}

	// Total todos
	row := s.db.QueryRow("SELECT COUNT(*) FROM todos"+whereClause, args...)
	row.Scan(&stats.TotalTodos)

	// By status
	rows, err := s.db.Query("SELECT status, COUNT(*) FROM todos"+whereClause+" GROUP BY status", args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status string
			var count int
			if rows.Scan(&status, &count) == nil {
				stats.TodosByStatus[types.TodoStatus(status)] = count
			}
		}
	}

	// By priority
	rows, err = s.db.Query("SELECT priority, COUNT(*) FROM todos"+whereClause+" GROUP BY priority", args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var priority string
			var count int
			if rows.Scan(&priority, &count) == nil {
				stats.TodosByPriority[types.TodoPriority(priority)] = count
			}
		}
	}

	// By channel (only if not filtering)
	if channel == "" {
		rows, err := s.db.Query("SELECT channel, COUNT(*) FROM todos GROUP BY channel")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var ch string
				var count int
				if rows.Scan(&ch, &count) == nil {
					stats.TodosByChannel[ch] = count
				}
			}
		}
	}

	// Overdue count
	now := time.Now().Unix()
	row = s.db.QueryRow("SELECT COUNT(*) FROM todos WHERE due_date < ? AND status NOT IN ('completed', 'cancelled')"+
		func() string {
			if channel != "" {
				return " AND channel = ?"
			}
			return ""
		}(), func() []any {
		if channel != "" {
			return []any{now, channel}
		}
		return []any{now}
	}()...)
	row.Scan(&stats.OverdueTodos)

	// Completed today
	startOfDay := time.Now().Truncate(24 * time.Hour).Unix()
	row = s.db.QueryRow("SELECT COUNT(*) FROM todos WHERE completed_at >= ?"+
		func() string {
			if channel != "" {
				return " AND channel = ?"
			}
			return ""
		}(), func() []any {
		if channel != "" {
			return []any{startOfDay, channel}
		}
		return []any{startOfDay}
	}()...)
	row.Scan(&stats.CompletedToday)

	// Completed this week
	startOfWeek := time.Now().AddDate(0, 0, -7).Unix()
	row = s.db.QueryRow("SELECT COUNT(*) FROM todos WHERE completed_at >= ?"+
		func() string {
			if channel != "" {
				return " AND channel = ?"
			}
			return ""
		}(), func() []any {
		if channel != "" {
			return []any{startOfWeek, channel}
		}
		return []any{startOfWeek}
	}()...)
	row.Scan(&stats.CompletedThisWeek)

	return stats, nil
}

// Helper function to scan a todo from a row.
func scanTodo(row *sql.Row) (*types.TodoItem, error) {
	var t types.TodoItem
	var description, tags, parentID, relMem, relChunks, relCommits, blockedBy sql.NullString
	var effort, actualEffort, sessionID sql.NullString
	var status, priority string
	var createdAt, updatedAt int64
	var dueDate, startedAt, completedAt sql.NullInt64

	err := row.Scan(
		&t.ID, &t.Title, &description, &status, &priority, &tags,
		&parentID, &t.Channel, &relMem, &relChunks, &relCommits, &blockedBy,
		&t.Progress, &effort, &actualEffort, &sessionID,
		&createdAt, &updatedAt, &dueDate, &startedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t.Description = description.String
	t.Status = types.TodoStatus(status)
	t.Priority = types.TodoPriority(priority)
	t.ParentID = parentID.String
	t.Effort = effort.String
	t.ActualEffort = actualEffort.String
	t.SessionID = sessionID.String
	t.CreatedAt = time.Unix(createdAt, 0)
	t.UpdatedAt = time.Unix(updatedAt, 0)

	if dueDate.Valid {
		due := time.Unix(dueDate.Int64, 0)
		t.DueDate = &due
	}
	if startedAt.Valid {
		started := time.Unix(startedAt.Int64, 0)
		t.StartedAt = &started
	}
	if completedAt.Valid {
		completed := time.Unix(completedAt.Int64, 0)
		t.CompletedAt = &completed
	}

	if tags.Valid && tags.String != "" {
		json.Unmarshal([]byte(tags.String), &t.Tags)
	}
	if relMem.Valid && relMem.String != "" {
		json.Unmarshal([]byte(relMem.String), &t.RelatedMemories)
	}
	if relChunks.Valid && relChunks.String != "" {
		json.Unmarshal([]byte(relChunks.String), &t.RelatedChunks)
	}
	if relCommits.Valid && relCommits.String != "" {
		json.Unmarshal([]byte(relCommits.String), &t.RelatedCommits)
	}
	if blockedBy.Valid && blockedBy.String != "" {
		json.Unmarshal([]byte(blockedBy.String), &t.BlockedBy)
	}

	return &t, nil
}

// scanTodos scans multiple todos from rows.
func scanTodos(rows *sql.Rows) ([]*types.TodoItem, error) {
	var todos []*types.TodoItem

	for rows.Next() {
		var t types.TodoItem
		var description, tags, parentID, relMem, relChunks, relCommits, blockedBy sql.NullString
		var effort, actualEffort, sessionID sql.NullString
		var status, priority string
		var createdAt, updatedAt int64
		var dueDate, startedAt, completedAt sql.NullInt64

		err := rows.Scan(
			&t.ID, &t.Title, &description, &status, &priority, &tags,
			&parentID, &t.Channel, &relMem, &relChunks, &relCommits, &blockedBy,
			&t.Progress, &effort, &actualEffort, &sessionID,
			&createdAt, &updatedAt, &dueDate, &startedAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}

		t.Description = description.String
		t.Status = types.TodoStatus(status)
		t.Priority = types.TodoPriority(priority)
		t.ParentID = parentID.String
		t.Effort = effort.String
		t.ActualEffort = actualEffort.String
		t.SessionID = sessionID.String
		t.CreatedAt = time.Unix(createdAt, 0)
		t.UpdatedAt = time.Unix(updatedAt, 0)

		if dueDate.Valid {
			due := time.Unix(dueDate.Int64, 0)
			t.DueDate = &due
		}
		if startedAt.Valid {
			started := time.Unix(startedAt.Int64, 0)
			t.StartedAt = &started
		}
		if completedAt.Valid {
			completed := time.Unix(completedAt.Int64, 0)
			t.CompletedAt = &completed
		}

		if tags.Valid && tags.String != "" {
			json.Unmarshal([]byte(tags.String), &t.Tags)
		}
		if relMem.Valid && relMem.String != "" {
			json.Unmarshal([]byte(relMem.String), &t.RelatedMemories)
		}
		if relChunks.Valid && relChunks.String != "" {
			json.Unmarshal([]byte(relChunks.String), &t.RelatedChunks)
		}
		if relCommits.Valid && relCommits.String != "" {
			json.Unmarshal([]byte(relCommits.String), &t.RelatedCommits)
		}
		if blockedBy.Valid && blockedBy.String != "" {
			json.Unmarshal([]byte(blockedBy.String), &t.BlockedBy)
		}

		todos = append(todos, &t)
	}

	return todos, nil
}
