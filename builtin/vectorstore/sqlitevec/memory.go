// Package sqlitevec implements memory storage and search using SQLite.
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

// createMemorySchema creates tables for memory storage.
func (s *Store) createMemorySchema() error {
	// Memories table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			summary TEXT,
			category TEXT NOT NULL,
			tags TEXT,
			importance REAL DEFAULT 0.5,
			confidence REAL DEFAULT 1.0,
			access_count INTEGER DEFAULT 0,
			channel TEXT DEFAULT 'main',
			related_memories TEXT,
			related_chunks TEXT,
			related_commits TEXT,
			session_id TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			accessed_at INTEGER NOT NULL,
			expires_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}

	// Indexes for memories
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_memories_channel ON memories(channel)",
		"CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category)",
		"CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance)",
		"CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_memories_accessed ON memories(accessed_at)",
		"CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id)",
		"CREATE INDEX IF NOT EXISTS idx_memories_expires ON memories(expires_at)",
	}
	for _, idx := range indexes {
		if _, err := s.db.Exec(idx); err != nil {
			return err
		}
	}

	// FTS5 for memory content search
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
			id,
			content,
			summary,
			content='memories',
			content_rowid='rowid',
			tokenize='porter unicode61'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create memories_fts table: %w", err)
	}

	// Triggers to keep FTS in sync
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
			INSERT INTO memories_fts(rowid, id, content, summary)
			VALUES (new.rowid, new.id, new.content, new.summary);
		END
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, id, content, summary)
			VALUES('delete', old.rowid, old.id, old.content, old.summary);
		END
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, id, content, summary)
			VALUES('delete', old.rowid, old.id, old.content, old.summary);
			INSERT INTO memories_fts(rowid, id, content, summary)
			VALUES (new.rowid, new.id, new.content, new.summary);
		END
	`)
	if err != nil {
		return err
	}

	// Memory sessions table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS memory_sessions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			channel TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			ended_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create memory_sessions table: %w", err)
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_channel ON memory_sessions(channel)`)
	if err != nil {
		return err
	}

	// Memory checkpoints table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS memory_checkpoints (
			id TEXT PRIMARY KEY,
			session_id TEXT,
			channel TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			created_at INTEGER NOT NULL,
			memory_ids TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create memory_checkpoints table: %w", err)
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_checkpoints_channel ON memory_checkpoints(channel)`)
	if err != nil {
		return err
	}

	return nil
}

// createMemoryEmbeddingsTable creates vector table for memory embeddings.
func (s *Store) createMemoryEmbeddingsTable(dimensions int) error {
	_, err := s.db.Exec(fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS memory_embeddings USING vec0(
			memory_id TEXT PRIMARY KEY,
			embedding float[%d]
		)
	`, dimensions))
	if err != nil {
		return fmt.Errorf("failed to create memory_embeddings table: %w", err)
	}
	return nil
}

// StoreMemory stores a single memory entry.
func (s *Store) StoreMemory(memory *types.MemoryEntry) error {
	return s.StoreMemories([]*types.MemoryEntry{memory})
}

// StoreMemories stores multiple memory entries.
func (s *Store) StoreMemories(memories []*types.MemoryEntry) error {
	if len(memories) == 0 {
		return nil
	}

	// Create embedding table if we have embeddings
	for _, m := range memories {
		if len(m.Embedding) > 0 {
			if err := s.createMemoryEmbeddingsTable(len(m.Embedding)); err != nil {
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

	memoryStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO memories
		(id, content, summary, category, tags, importance, confidence, access_count,
		 channel, related_memories, related_chunks, related_commits, session_id,
		 created_at, updated_at, accessed_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer memoryStmt.Close()

	embStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO memory_embeddings (memory_id, embedding)
		VALUES (?, ?)
	`)
	if err != nil {
		embStmt = nil
	} else {
		defer embStmt.Close()
	}

	for _, m := range memories {
		var tagsJSON, relMemJSON, relChunksJSON, relCommitsJSON string
		var expiresAt sql.NullInt64

		if len(m.Tags) > 0 {
			b, _ := json.Marshal(m.Tags)
			tagsJSON = string(b)
		}
		if len(m.RelatedMemories) > 0 {
			b, _ := json.Marshal(m.RelatedMemories)
			relMemJSON = string(b)
		}
		if len(m.RelatedChunks) > 0 {
			b, _ := json.Marshal(m.RelatedChunks)
			relChunksJSON = string(b)
		}
		if len(m.RelatedCommits) > 0 {
			b, _ := json.Marshal(m.RelatedCommits)
			relCommitsJSON = string(b)
		}
		if m.ExpiresAt != nil {
			expiresAt.Int64 = m.ExpiresAt.Unix()
			expiresAt.Valid = true
		}

		_, err := memoryStmt.Exec(
			m.ID, m.Content, m.Summary, string(m.Category), tagsJSON,
			m.Importance, m.Confidence, m.AccessCount,
			m.Channel, relMemJSON, relChunksJSON, relCommitsJSON, m.SessionID,
			m.CreatedAt.Unix(), m.UpdatedAt.Unix(), m.AccessedAt.Unix(), expiresAt,
		)
		if err != nil {
			return fmt.Errorf("failed to store memory %s: %w", m.ID, err)
		}

		// Store embedding if present
		if embStmt != nil && len(m.Embedding) > 0 {
			embBytes := floatsToBytes(m.Embedding)
			_, err := embStmt.Exec(m.ID, embBytes)
			if err != nil {
				return fmt.Errorf("failed to store embedding for memory %s: %w", m.ID, err)
			}
		}
	}

	return tx.Commit()
}

// GetMemory retrieves a memory by ID.
func (s *Store) GetMemory(id string) (*types.MemoryEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, content, summary, category, tags, importance, confidence, access_count,
		       channel, related_memories, related_chunks, related_commits, session_id,
		       created_at, updated_at, accessed_at, expires_at
		FROM memories WHERE id = ?
	`, id)

	return scanMemory(row)
}

// UpdateMemory updates an existing memory entry.
func (s *Store) UpdateMemory(memory *types.MemoryEntry) error {
	memory.UpdatedAt = time.Now()
	return s.StoreMemory(memory)
}

// DeleteMemory deletes a memory by ID.
func (s *Store) DeleteMemory(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete embedding
	_, err = tx.Exec("DELETE FROM memory_embeddings WHERE memory_id = ?", id)
	if err != nil {
		// Ignore if table doesn't exist
	}

	// Delete memory (FTS will be updated by trigger)
	_, err = tx.Exec("DELETE FROM memories WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteMemoriesByChannel deletes all memories in a channel.
func (s *Store) DeleteMemoriesByChannel(channel string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get memory IDs first
	rows, err := tx.Query("SELECT id FROM memories WHERE channel = ?", channel)
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
		_, _ = tx.Exec("DELETE FROM memory_embeddings WHERE memory_id = ?", id)
	}

	// Delete memories
	_, err = tx.Exec("DELETE FROM memories WHERE channel = ?", channel)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SearchMemories performs hybrid search across memories.
func (s *Store) SearchMemories(ctx context.Context, req *types.MemorySearchRequest) ([]*types.MemorySearchResult, error) {
	candidateLimit := req.Limit * 5
	if req.UseReranker && req.RerankCandidates > 0 {
		candidateLimit = req.RerankCandidates
	}
	if candidateLimit < 100 {
		candidateLimit = 100
	}

	// Set default weights
	vectorWeight := req.VectorWeight
	recencyWeight := req.RecencyWeight
	importanceWeight := req.ImportanceWeight
	frequencyWeight := req.FrequencyWeight

	if vectorWeight == 0 && recencyWeight == 0 && importanceWeight == 0 && frequencyWeight == 0 {
		vectorWeight = 0.5
		recencyWeight = 0.2
		importanceWeight = 0.2
		frequencyWeight = 0.1
	}

	// Collect candidates from both vector and BM25 search
	candidates := make(map[string]*types.MemorySearchResult)

	// Vector search
	if len(req.QueryVec) > 0 {
		results, err := s.vectorSearchMemories(ctx, req.QueryVec, candidateLimit, req)
		if err == nil {
			for _, r := range results {
				candidates[r.Memory.ID] = r
			}
		}
	}

	// BM25 search
	if req.Query != "" {
		results, err := s.bm25SearchMemories(ctx, req.Query, candidateLimit, req)
		if err == nil {
			for _, r := range results {
				if existing, ok := candidates[r.Memory.ID]; ok {
					// Merge scores
					existing.VectorScore = (existing.VectorScore + r.VectorScore) / 2
				} else {
					candidates[r.Memory.ID] = r
				}
			}
		}
	}

	// Calculate hybrid scores
	now := time.Now()
	var results []*types.MemorySearchResult

	for _, r := range candidates {
		// Recency score (exponential decay, 30-day half-life)
		age := now.Sub(r.Memory.AccessedAt)
		r.RecencyScore = float32(math.Exp(-age.Hours() / (24 * 30 * math.Ln2)))

		// Importance score
		r.ImportanceScore = r.Memory.Importance * r.Memory.Confidence

		// Frequency score (log scale)
		if r.Memory.AccessCount > 0 {
			r.FrequencyScore = float32(math.Log10(float64(r.Memory.AccessCount)+1) / 3)
			if r.FrequencyScore > 1 {
				r.FrequencyScore = 1
			}
		}

		// Combined score
		r.Score = r.VectorScore*vectorWeight +
			r.RecencyScore*recencyWeight +
			r.ImportanceScore*importanceWeight +
			r.FrequencyScore*frequencyWeight

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

	// Update access count and accessed_at for returned memories
	for _, r := range results {
		s.touchMemory(r.Memory.ID)
	}

	return results, nil
}

// vectorSearchMemories performs vector similarity search on memories.
func (s *Store) vectorSearchMemories(ctx context.Context, queryVec []float32, limit int, req *types.MemorySearchRequest) ([]*types.MemorySearchResult, error) {
	embBytes := floatsToBytes(queryVec)

	query := `
		SELECT
			me.memory_id,
			vec_distance_cosine(me.embedding, ?) as distance
		FROM memory_embeddings me
		JOIN memories m ON me.memory_id = m.id
		WHERE 1=1
	`
	args := []any{embBytes}

	// Apply filters
	query, args = s.applyMemoryFilters(query, args, req)

	query += " ORDER BY distance ASC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*types.MemorySearchResult
	for rows.Next() {
		var id string
		var distance float64
		if err := rows.Scan(&id, &distance); err != nil {
			return nil, err
		}

		memory, err := s.GetMemory(id)
		if err != nil || memory == nil {
			continue
		}

		score := float32(1.0 - distance)
		results = append(results, &types.MemorySearchResult{
			Memory:      memory,
			VectorScore: score,
		})
	}

	return results, nil
}

// bm25SearchMemories performs BM25 search on memories.
func (s *Store) bm25SearchMemories(ctx context.Context, query string, limit int, req *types.MemorySearchRequest) ([]*types.MemorySearchResult, error) {
	sqlQuery := `
		SELECT fts.id, bm25(memories_fts) as score
		FROM memories_fts fts
		JOIN memories m ON fts.id = m.id
		WHERE memories_fts MATCH ?
	`
	args := []any{escapeFTSQuery(query)}

	// Apply filters
	sqlQuery, args = s.applyMemoryFilters(sqlQuery, args, req)

	sqlQuery += " ORDER BY score LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*types.MemorySearchResult
	for rows.Next() {
		var id string
		var score float64
		if err := rows.Scan(&id, &score); err != nil {
			return nil, err
		}

		memory, err := s.GetMemory(id)
		if err != nil || memory == nil {
			continue
		}

		// Normalize BM25 score
		normalizedScore := float32(1.0 / (1.0 + math.Abs(score)))
		results = append(results, &types.MemorySearchResult{
			Memory:      memory,
			VectorScore: normalizedScore,
		})
	}

	return results, nil
}

// applyMemoryFilters adds filter clauses to a memory query.
func (s *Store) applyMemoryFilters(query string, args []any, req *types.MemorySearchRequest) (string, []any) {
	if req == nil {
		return query, args
	}

	if req.Channel != "" {
		query += " AND m.channel = ?"
		args = append(args, req.Channel)
	}

	if len(req.Categories) > 0 {
		placeholders := make([]string, len(req.Categories))
		for i, c := range req.Categories {
			placeholders[i] = "?"
			args = append(args, string(c))
		}
		query += " AND m.category IN (" + strings.Join(placeholders, ",") + ")"
	}

	if req.MinImportance > 0 {
		query += " AND m.importance >= ?"
		args = append(args, req.MinImportance)
	}

	if !req.IncludeExpired {
		query += " AND (m.expires_at IS NULL OR m.expires_at > ?)"
		args = append(args, time.Now().Unix())
	}

	if req.CreatedAfter != nil {
		query += " AND m.created_at >= ?"
		args = append(args, req.CreatedAfter.Unix())
	}

	if req.CreatedBefore != nil {
		query += " AND m.created_at <= ?"
		args = append(args, req.CreatedBefore.Unix())
	}

	if req.AccessedAfter != nil {
		query += " AND m.accessed_at >= ?"
		args = append(args, req.AccessedAfter.Unix())
	}

	// Tag filtering requires JSON parsing - simplified version
	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			query += " AND m.tags LIKE ?"
			args = append(args, "%"+tag+"%")
		}
	}

	return query, args
}

// touchMemory updates access count and accessed_at.
func (s *Store) touchMemory(id string) {
	_, _ = s.db.Exec(`
		UPDATE memories
		SET access_count = access_count + 1, accessed_at = ?
		WHERE id = ?
	`, time.Now().Unix(), id)
}

// RecallMemories is a simplified recall for common use cases.
func (s *Store) RecallMemories(ctx context.Context, query string, channel string, limit int) ([]*types.MemoryEntry, error) {
	req := &types.MemorySearchRequest{
		Query:   query,
		Channel: channel,
		Limit:   limit,
	}

	results, err := s.SearchMemories(ctx, req)
	if err != nil {
		return nil, err
	}

	memories := make([]*types.MemoryEntry, len(results))
	for i, r := range results {
		memories[i] = r.Memory
	}

	return memories, nil
}

// CreateSession creates a new memory session.
func (s *Store) CreateSession(session *types.MemorySession) error {
	_, err := s.db.Exec(`
		INSERT INTO memory_sessions (id, name, description, channel, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.Name, session.Description, session.Channel, session.StartedAt.Unix())
	return err
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(id string) (*types.MemorySession, error) {
	row := s.db.QueryRow(`
		SELECT id, name, description, channel, started_at, ended_at
		FROM memory_sessions WHERE id = ?
	`, id)

	var session types.MemorySession
	var description sql.NullString
	var startedAtUnix int64
	var endedAtUnix sql.NullInt64

	err := row.Scan(&session.ID, &session.Name, &description, &session.Channel,
		&startedAtUnix, &endedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	session.Description = description.String
	session.StartedAt = time.Unix(startedAtUnix, 0)
	if endedAtUnix.Valid {
		t := time.Unix(endedAtUnix.Int64, 0)
		session.EndedAt = &t
	}

	// Get memory count
	row = s.db.QueryRow("SELECT COUNT(*) FROM memories WHERE session_id = ?", id)
	row.Scan(&session.MemoryCount)

	return &session, nil
}

// EndSession marks a session as ended.
func (s *Store) EndSession(id string) error {
	_, err := s.db.Exec(`
		UPDATE memory_sessions SET ended_at = ? WHERE id = ?
	`, time.Now().Unix(), id)
	return err
}

// GetActiveSessions returns active sessions for a channel.
func (s *Store) GetActiveSessions(channel string) ([]*types.MemorySession, error) {
	query := "SELECT id, name, description, channel, started_at, ended_at FROM memory_sessions WHERE ended_at IS NULL"
	args := []any{}

	if channel != "" {
		query += " AND channel = ?"
		args = append(args, channel)
	}

	query += " ORDER BY started_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*types.MemorySession
	for rows.Next() {
		var session types.MemorySession
		var description sql.NullString
		var startedAtUnix int64
		var endedAtUnix sql.NullInt64

		err := rows.Scan(&session.ID, &session.Name, &description, &session.Channel,
			&startedAtUnix, &endedAtUnix)
		if err != nil {
			return nil, err
		}

		session.Description = description.String
		session.StartedAt = time.Unix(startedAtUnix, 0)

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// CreateCheckpoint creates a memory checkpoint.
func (s *Store) CreateCheckpoint(checkpoint *types.MemoryCheckpoint) error {
	memoryIDsJSON, _ := json.Marshal(checkpoint.MemoryIDs)

	_, err := s.db.Exec(`
		INSERT INTO memory_checkpoints (id, session_id, channel, name, description, created_at, memory_ids)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, checkpoint.ID, checkpoint.SessionID, checkpoint.Channel, checkpoint.Name,
		checkpoint.Description, checkpoint.CreatedAt.Unix(), string(memoryIDsJSON))
	return err
}

// GetCheckpoint retrieves a checkpoint by ID.
func (s *Store) GetCheckpoint(id string) (*types.MemoryCheckpoint, error) {
	row := s.db.QueryRow(`
		SELECT id, session_id, channel, name, description, created_at, memory_ids
		FROM memory_checkpoints WHERE id = ?
	`, id)

	var checkpoint types.MemoryCheckpoint
	var sessionID, description sql.NullString
	var createdAtUnix int64
	var memoryIDsJSON string

	err := row.Scan(&checkpoint.ID, &sessionID, &checkpoint.Channel, &checkpoint.Name,
		&description, &createdAtUnix, &memoryIDsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	checkpoint.SessionID = sessionID.String
	checkpoint.Description = description.String
	checkpoint.CreatedAt = time.Unix(createdAtUnix, 0)
	json.Unmarshal([]byte(memoryIDsJSON), &checkpoint.MemoryIDs)

	return &checkpoint, nil
}

// RestoreCheckpoint restores memories from a checkpoint.
func (s *Store) RestoreCheckpoint(id string) error {
	checkpoint, err := s.GetCheckpoint(id)
	if err != nil {
		return err
	}
	if checkpoint == nil {
		return fmt.Errorf("checkpoint not found: %s", id)
	}

	// Delete memories not in checkpoint
	if len(checkpoint.MemoryIDs) > 0 {
		placeholders := make([]string, len(checkpoint.MemoryIDs))
		args := make([]any, len(checkpoint.MemoryIDs)+1)
		args[0] = checkpoint.Channel

		for i, id := range checkpoint.MemoryIDs {
			placeholders[i] = "?"
			args[i+1] = id
		}

		_, err = s.db.Exec(
			"DELETE FROM memories WHERE channel = ? AND id NOT IN ("+strings.Join(placeholders, ",")+")",
			args...)
		if err != nil {
			return err
		}
	}

	return nil
}

// ListCheckpoints lists checkpoints for a channel.
func (s *Store) ListCheckpoints(channel string, limit int) ([]*types.MemoryCheckpoint, error) {
	rows, err := s.db.Query(`
		SELECT id, session_id, channel, name, description, created_at, memory_ids
		FROM memory_checkpoints
		WHERE channel = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, channel, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkpoints []*types.MemoryCheckpoint
	for rows.Next() {
		var checkpoint types.MemoryCheckpoint
		var sessionID, description sql.NullString
		var createdAtUnix int64
		var memoryIDsJSON string

		err := rows.Scan(&checkpoint.ID, &sessionID, &checkpoint.Channel, &checkpoint.Name,
			&description, &createdAtUnix, &memoryIDsJSON)
		if err != nil {
			return nil, err
		}

		checkpoint.SessionID = sessionID.String
		checkpoint.Description = description.String
		checkpoint.CreatedAt = time.Unix(createdAtUnix, 0)
		json.Unmarshal([]byte(memoryIDsJSON), &checkpoint.MemoryIDs)

		checkpoints = append(checkpoints, &checkpoint)
	}

	return checkpoints, nil
}

// PruneExpired removes expired memories.
func (s *Store) PruneExpired() (int, error) {
	now := time.Now().Unix()

	// Get IDs of expired memories
	rows, err := s.db.Query("SELECT id FROM memories WHERE expires_at IS NOT NULL AND expires_at < ?", now)
	if err != nil {
		return 0, err
	}

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()

	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Delete embeddings
	for _, id := range ids {
		_, _ = tx.Exec("DELETE FROM memory_embeddings WHERE memory_id = ?", id)
	}

	// Delete memories
	result, err := tx.Exec("DELETE FROM memories WHERE expires_at IS NOT NULL AND expires_at < ?", now)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(affected), nil
}

// Consolidate merges similar memories (placeholder for future implementation).
func (s *Store) Consolidate(channel string) error {
	// TODO: Implement memory consolidation using vector similarity
	// This would find similar memories and merge them
	return nil
}

// DecayConfidence applies time-based confidence decay to memories.
func (s *Store) DecayConfidence(halfLifeDays int) error {
	if halfLifeDays <= 0 {
		halfLifeDays = 30
	}

	// Calculate decay factor
	// confidence_new = confidence_old * exp(-ln(2) * age_days / half_life)
	now := time.Now().Unix()
	halfLifeSeconds := int64(halfLifeDays * 24 * 60 * 60)

	_, err := s.db.Exec(`
		UPDATE memories
		SET confidence = confidence * exp(-0.693147 * (? - created_at) / ?)
		WHERE confidence > 0.01
	`, now, halfLifeSeconds)

	return err
}

// GetMemoryStats returns statistics about memories.
func (s *Store) GetMemoryStats(channel string) (*types.MemoryStats, error) {
	stats := &types.MemoryStats{
		MemoriesByCategory: make(map[types.MemoryCategory]int),
		MemoriesByChannel:  make(map[string]int),
	}

	whereClause := ""
	args := []any{}
	if channel != "" {
		whereClause = " WHERE channel = ?"
		args = append(args, channel)
	}

	// Total memories
	row := s.db.QueryRow("SELECT COUNT(*) FROM memories"+whereClause, args...)
	row.Scan(&stats.TotalMemories)

	// By category
	rows, err := s.db.Query("SELECT category, COUNT(*) FROM memories"+whereClause+" GROUP BY category", args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var count int
			if rows.Scan(&cat, &count) == nil {
				stats.MemoriesByCategory[types.MemoryCategory(cat)] = count
			}
		}
	}

	// By channel (only if not filtering by channel)
	if channel == "" {
		rows, err := s.db.Query("SELECT channel, COUNT(*) FROM memories GROUP BY channel")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var ch string
				var count int
				if rows.Scan(&ch, &count) == nil {
					stats.MemoriesByChannel[ch] = count
				}
			}
		}
	}

	// Active sessions
	row = s.db.QueryRow("SELECT COUNT(*) FROM memory_sessions WHERE ended_at IS NULL")
	row.Scan(&stats.ActiveSessions)

	// Checkpoints
	row = s.db.QueryRow("SELECT COUNT(*) FROM memory_checkpoints")
	row.Scan(&stats.TotalCheckpoints)

	// Date range
	var oldest, newest sql.NullInt64
	row = s.db.QueryRow("SELECT MIN(created_at), MAX(created_at) FROM memories" + whereClause)
	if row.Scan(&oldest, &newest) == nil {
		if oldest.Valid {
			t := time.Unix(oldest.Int64, 0)
			stats.OldestMemory = &t
		}
		if newest.Valid {
			t := time.Unix(newest.Int64, 0)
			stats.NewestMemory = &t
		}
	}

	// Expired count
	row = s.db.QueryRow("SELECT COUNT(*) FROM memories WHERE expires_at IS NOT NULL AND expires_at < ?", time.Now().Unix())
	row.Scan(&stats.ExpiredCount)

	return stats, nil
}

// Helper function to scan a memory from a row.
func scanMemory(row *sql.Row) (*types.MemoryEntry, error) {
	var m types.MemoryEntry
	var summary, tags, relMem, relChunks, relCommits, sessionID sql.NullString
	var createdAt, updatedAt, accessedAt int64
	var expiresAt sql.NullInt64
	var category string

	err := row.Scan(
		&m.ID, &m.Content, &summary, &category, &tags,
		&m.Importance, &m.Confidence, &m.AccessCount,
		&m.Channel, &relMem, &relChunks, &relCommits, &sessionID,
		&createdAt, &updatedAt, &accessedAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	m.Summary = summary.String
	m.Category = types.MemoryCategory(category)
	m.SessionID = sessionID.String
	m.CreatedAt = time.Unix(createdAt, 0)
	m.UpdatedAt = time.Unix(updatedAt, 0)
	m.AccessedAt = time.Unix(accessedAt, 0)

	if expiresAt.Valid {
		t := time.Unix(expiresAt.Int64, 0)
		m.ExpiresAt = &t
	}

	if tags.Valid && tags.String != "" {
		json.Unmarshal([]byte(tags.String), &m.Tags)
	}
	if relMem.Valid && relMem.String != "" {
		json.Unmarshal([]byte(relMem.String), &m.RelatedMemories)
	}
	if relChunks.Valid && relChunks.String != "" {
		json.Unmarshal([]byte(relChunks.String), &m.RelatedChunks)
	}
	if relCommits.Valid && relCommits.String != "" {
		json.Unmarshal([]byte(relCommits.String), &m.RelatedCommits)
	}

	return &m, nil
}
