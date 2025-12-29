// Package sqlitevec implements git history storage and search using SQLite.
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

// createGitHistorySchema creates tables for git history storage.
func (s *Store) createGitHistorySchema() error {
	// Commits table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS commits (
			hash TEXT PRIMARY KEY,
			short_hash TEXT NOT NULL,
			author TEXT NOT NULL,
			author_email TEXT NOT NULL,
			date INTEGER NOT NULL,
			message TEXT NOT NULL,
			parent_hash TEXT,
			files_changed INTEGER DEFAULT 0,
			insertions INTEGER DEFAULT 0,
			deletions INTEGER DEFAULT 0,
			is_merge BOOLEAN DEFAULT FALSE,
			is_tagged BOOLEAN DEFAULT FALSE,
			tags TEXT,
			branch TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create commits table: %w", err)
	}

	// Indexes for commits
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_commits_date ON commits(date)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_commits_author ON commits(author_email)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_commits_parent ON commits(parent_hash)`)
	if err != nil {
		return err
	}

	// FTS5 for commit message search
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS commits_fts USING fts5(
			hash,
			message,
			content='commits',
			content_rowid='rowid',
			tokenize='porter unicode61'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create commits_fts table: %w", err)
	}

	// Triggers to keep FTS in sync
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS commits_ai AFTER INSERT ON commits BEGIN
			INSERT INTO commits_fts(rowid, hash, message)
			VALUES (new.rowid, new.hash, new.message);
		END
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS commits_ad AFTER DELETE ON commits BEGIN
			INSERT INTO commits_fts(commits_fts, rowid, hash, message)
			VALUES('delete', old.rowid, old.hash, old.message);
		END
	`)
	if err != nil {
		return err
	}

	// Changes table (file changes within commits)
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS changes (
			id TEXT PRIMARY KEY,
			commit_hash TEXT NOT NULL,
			file_path TEXT NOT NULL,
			change_type TEXT NOT NULL,
			old_path TEXT,
			diff_content TEXT,
			additions INTEGER DEFAULT 0,
			deletions INTEGER DEFAULT 0,
			affected_functions TEXT,
			affected_chunk_ids TEXT,
			hunks TEXT,
			FOREIGN KEY (commit_hash) REFERENCES commits(hash)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create changes table: %w", err)
	}

	// Indexes for changes
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_changes_commit ON changes(commit_hash)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_changes_file ON changes(file_path)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_changes_type ON changes(change_type)`)
	if err != nil {
		return err
	}

	// Chunk history table (links chunks to their commit history)
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS chunk_history (
			chunk_id TEXT NOT NULL,
			commit_hash TEXT NOT NULL,
			change_type TEXT NOT NULL,
			diff_summary TEXT,
			date INTEGER NOT NULL,
			author TEXT NOT NULL,
			PRIMARY KEY (chunk_id, commit_hash),
			FOREIGN KEY (commit_hash) REFERENCES commits(hash)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create chunk_history table: %w", err)
	}

	// Index for chunk history
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_chunk_history_chunk ON chunk_history(chunk_id)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_chunk_history_date ON chunk_history(date)`)
	if err != nil {
		return err
	}

	// Git indexing metadata
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS git_index_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create git_index_meta table: %w", err)
	}

	return nil
}

// createCommitEmbeddingsTable creates vector table for commit message embeddings.
func (s *Store) createCommitEmbeddingsTable(dimensions int) error {
	_, err := s.db.Exec(fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS commit_embeddings USING vec0(
			commit_hash TEXT PRIMARY KEY,
			embedding float[%d]
		)
	`, dimensions))
	if err != nil {
		return fmt.Errorf("failed to create commit_embeddings table: %w", err)
	}
	return nil
}

// createChangeEmbeddingsTable creates vector table for diff embeddings.
func (s *Store) createChangeEmbeddingsTable(dimensions int) error {
	_, err := s.db.Exec(fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS change_embeddings USING vec0(
			change_id TEXT PRIMARY KEY,
			embedding float[%d]
		)
	`, dimensions))
	if err != nil {
		return fmt.Errorf("failed to create change_embeddings table: %w", err)
	}
	return nil
}

// StoreCommits stores commits with optional embeddings.
func (s *Store) StoreCommits(commits []*types.Commit) error {
	if len(commits) == 0 {
		return nil
	}

	// Create embedding table if we have embeddings
	for _, c := range commits {
		if len(c.MessageEmbedding) > 0 {
			if err := s.createCommitEmbeddingsTable(len(c.MessageEmbedding)); err != nil {
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

	commitStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO commits
		(hash, short_hash, author, author_email, date, message, parent_hash,
		 files_changed, insertions, deletions, is_merge, is_tagged, tags, branch)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer commitStmt.Close()

	embStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO commit_embeddings (commit_hash, embedding)
		VALUES (?, ?)
	`)
	if err != nil {
		// Embedding table might not exist yet
		embStmt = nil
	} else {
		defer embStmt.Close()
	}

	for _, c := range commits {
		var tagsJSON string
		if len(c.Tags) > 0 {
			b, _ := json.Marshal(c.Tags)
			tagsJSON = string(b)
		}

		_, err := commitStmt.Exec(
			c.Hash, c.ShortHash, c.Author, c.AuthorEmail,
			c.Date.Unix(), c.Message, c.ParentHash,
			c.FilesChanged, c.Insertions, c.Deletions,
			c.IsMerge, c.IsTagged, tagsJSON, c.Branch,
		)
		if err != nil {
			return fmt.Errorf("failed to store commit %s: %w", c.Hash, err)
		}

		// Store embedding if present
		if embStmt != nil && len(c.MessageEmbedding) > 0 {
			embBytes := floatsToBytes(c.MessageEmbedding)
			_, err := embStmt.Exec(c.Hash, embBytes)
			if err != nil {
				return fmt.Errorf("failed to store embedding for commit %s: %w", c.Hash, err)
			}
		}
	}

	return tx.Commit()
}

// GetCommit retrieves a commit by hash.
func (s *Store) GetCommit(hash string) (*types.Commit, error) {
	row := s.db.QueryRow(`
		SELECT hash, short_hash, author, author_email, date, message, parent_hash,
		       files_changed, insertions, deletions, is_merge, is_tagged, tags, branch
		FROM commits WHERE hash = ? OR short_hash = ?
	`, hash, hash)

	return scanCommit(row)
}

// GetCommitRange returns commits between two hashes.
func (s *Store) GetCommitRange(fromHash, toHash string) ([]*types.Commit, error) {
	// Get the date range from the commits
	var fromDate, toDate int64

	if fromHash != "" {
		row := s.db.QueryRow("SELECT date FROM commits WHERE hash = ? OR short_hash = ?", fromHash, fromHash)
		if err := row.Scan(&fromDate); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	if toHash != "" {
		row := s.db.QueryRow("SELECT date FROM commits WHERE hash = ? OR short_hash = ?", toHash, toHash)
		if err := row.Scan(&toDate); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	} else {
		toDate = time.Now().Unix()
	}

	rows, err := s.db.Query(`
		SELECT hash, short_hash, author, author_email, date, message, parent_hash,
		       files_changed, insertions, deletions, is_merge, is_tagged, tags, branch
		FROM commits
		WHERE date >= ? AND date <= ?
		ORDER BY date DESC
	`, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommits(rows)
}

// GetCommitsByTimeRange returns commits within a time range.
func (s *Store) GetCommitsByTimeRange(from, to *time.Time, limit int) ([]*types.Commit, error) {
	var fromUnix, toUnix int64

	if from != nil {
		fromUnix = from.Unix()
	}
	if to != nil {
		toUnix = to.Unix()
	} else {
		toUnix = time.Now().Unix()
	}

	rows, err := s.db.Query(`
		SELECT hash, short_hash, author, author_email, date, message, parent_hash,
		       files_changed, insertions, deletions, is_merge, is_tagged, tags, branch
		FROM commits
		WHERE date >= ? AND date <= ?
		ORDER BY date DESC
		LIMIT ?
	`, fromUnix, toUnix, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommits(rows)
}

// GetLastIndexedCommit returns the hash of the last indexed commit.
func (s *Store) GetLastIndexedCommit() (string, error) {
	row := s.db.QueryRow("SELECT value FROM git_index_meta WHERE key = 'last_indexed_commit'")
	var hash string
	err := row.Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

// SetLastIndexedCommit sets the hash of the last indexed commit.
func (s *Store) SetLastIndexedCommit(hash string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO git_index_meta (key, value)
		VALUES ('last_indexed_commit', ?)
	`, hash)
	return err
}

// StoreChanges stores file changes with optional embeddings.
func (s *Store) StoreChanges(changes []*types.Change) error {
	if len(changes) == 0 {
		return nil
	}

	// Create embedding table if we have embeddings
	for _, c := range changes {
		if len(c.DiffEmbedding) > 0 {
			if err := s.createChangeEmbeddingsTable(len(c.DiffEmbedding)); err != nil {
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

	changeStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO changes
		(id, commit_hash, file_path, change_type, old_path, diff_content,
		 additions, deletions, affected_functions, affected_chunk_ids, hunks)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer changeStmt.Close()

	embStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO change_embeddings (change_id, embedding)
		VALUES (?, ?)
	`)
	if err != nil {
		embStmt = nil
	} else {
		defer embStmt.Close()
	}

	for _, c := range changes {
		var affectedFuncsJSON, affectedChunksJSON, hunksJSON string

		if len(c.AffectedFunctions) > 0 {
			b, _ := json.Marshal(c.AffectedFunctions)
			affectedFuncsJSON = string(b)
		}
		if len(c.AffectedChunkIDs) > 0 {
			b, _ := json.Marshal(c.AffectedChunkIDs)
			affectedChunksJSON = string(b)
		}
		if len(c.Hunks) > 0 {
			b, _ := json.Marshal(c.Hunks)
			hunksJSON = string(b)
		}

		_, err := changeStmt.Exec(
			c.ID, c.CommitHash, c.FilePath, string(c.ChangeType), c.OldPath,
			c.DiffContent, c.Additions, c.Deletions,
			affectedFuncsJSON, affectedChunksJSON, hunksJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to store change %s: %w", c.ID, err)
		}

		// Store embedding if present
		if embStmt != nil && len(c.DiffEmbedding) > 0 {
			embBytes := floatsToBytes(c.DiffEmbedding)
			_, err := embStmt.Exec(c.ID, embBytes)
			if err != nil {
				return fmt.Errorf("failed to store embedding for change %s: %w", c.ID, err)
			}
		}
	}

	return tx.Commit()
}

// GetChange retrieves a change by ID.
func (s *Store) GetChange(id string) (*types.Change, error) {
	row := s.db.QueryRow(`
		SELECT id, commit_hash, file_path, change_type, old_path, diff_content,
		       additions, deletions, affected_functions, affected_chunk_ids, hunks
		FROM changes WHERE id = ?
	`, id)

	return scanChange(row)
}

// GetChangesByCommit returns all changes for a commit.
func (s *Store) GetChangesByCommit(commitHash string) ([]*types.Change, error) {
	rows, err := s.db.Query(`
		SELECT id, commit_hash, file_path, change_type, old_path, diff_content,
		       additions, deletions, affected_functions, affected_chunk_ids, hunks
		FROM changes WHERE commit_hash = ?
	`, commitHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanChanges(rows)
}

// GetChangesByFile returns changes for a specific file.
func (s *Store) GetChangesByFile(filePath string, limit int) ([]*types.Change, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.commit_hash, c.file_path, c.change_type, c.old_path, c.diff_content,
		       c.additions, c.deletions, c.affected_functions, c.affected_chunk_ids, c.hunks
		FROM changes c
		JOIN commits cm ON c.commit_hash = cm.hash
		WHERE c.file_path = ?
		ORDER BY cm.date DESC
		LIMIT ?
	`, filePath, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanChanges(rows)
}

// StoreChunkHistory stores chunk history entries.
func (s *Store) StoreChunkHistory(entries []*types.ChunkHistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO chunk_history
		(chunk_id, commit_hash, change_type, diff_summary, date, author)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		_, err := stmt.Exec(
			e.ChunkID, e.CommitHash, e.ChangeType,
			e.DiffSummary, e.Date.Unix(), e.Author,
		)
		if err != nil {
			return fmt.Errorf("failed to store chunk history: %w", err)
		}
	}

	return tx.Commit()
}

// GetChunkHistory returns history for a chunk.
func (s *Store) GetChunkHistory(chunkID string, limit int) ([]*types.ChunkHistoryEntry, error) {
	rows, err := s.db.Query(`
		SELECT chunk_id, commit_hash, change_type, diff_summary, date, author
		FROM chunk_history
		WHERE chunk_id = ?
		ORDER BY date DESC
		LIMIT ?
	`, chunkID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*types.ChunkHistoryEntry
	for rows.Next() {
		var e types.ChunkHistoryEntry
		var dateUnix int64
		var diffSummary sql.NullString

		err := rows.Scan(&e.ChunkID, &e.CommitHash, &e.ChangeType, &diffSummary, &dateUnix, &e.Author)
		if err != nil {
			return nil, err
		}

		e.Date = time.Unix(dateUnix, 0)
		e.DiffSummary = diffSummary.String
		entries = append(entries, &e)
	}

	return entries, nil
}

// GetChunkEvolution returns detailed evolution info for a chunk.
func (s *Store) GetChunkEvolution(chunkID string) (*types.ChunkEvolution, error) {
	// Get chunk info
	chunk, err := s.GetChunk(chunkID)
	if err != nil {
		return nil, err
	}
	if chunk == nil {
		return nil, fmt.Errorf("chunk not found: %s", chunkID)
	}

	// Get history entries
	history, err := s.GetChunkHistory(chunkID, 1000)
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return &types.ChunkEvolution{
			ChunkID:     chunkID,
			CurrentName: chunk.Name,
			FilePath:    chunk.FilePath,
			Authors:     make(map[string]int),
			History:     []*types.ChunkHistoryEntry{},
		}, nil
	}

	// Build evolution
	evolution := &types.ChunkEvolution{
		ChunkID:      chunkID,
		CurrentName:  chunk.Name,
		FilePath:     chunk.FilePath,
		Authors:      make(map[string]int),
		History:      history,
		ModifyCount:  len(history),
	}

	// Find first and last change
	for i, h := range history {
		evolution.Authors[h.Author]++

		if i == 0 {
			evolution.LastModified = h.Date
		}
		if i == len(history)-1 || h.ChangeType == "created" {
			evolution.CreatedAt = h.Date
			evolution.CreatedBy = h.Author
		}
	}

	return evolution, nil
}

// SearchHistory performs semantic search across git history.
func (s *Store) SearchHistory(ctx context.Context, req *types.HistorySearchRequest) ([]*types.HistorySearchResult, error) {
	candidateLimit := req.Limit * 5
	if req.UseReranker && req.RerankCandidates > 0 {
		candidateLimit = req.RerankCandidates
	}

	var results []*types.HistorySearchResult

	// Search commit messages (vector + BM25)
	if len(req.QueryVec) > 0 {
		commits, err := s.SearchCommitMessages(ctx, req.QueryVec, candidateLimit)
		if err != nil {
			return nil, fmt.Errorf("commit message search failed: %w", err)
		}

		for _, commit := range commits {
			// Get changes for this commit
			changes, _ := s.GetChangesByCommit(commit.Hash)

			// Apply filters
			if !s.matchesHistoryFilters(commit, changes, req) {
				continue
			}

			for _, change := range changes {
				results = append(results, &types.HistorySearchResult{
					Change:       change,
					Commit:       commit,
					MessageScore: 1.0, // Placeholder - actual scoring done below
				})
			}
		}
	}

	// Search diff content (vector)
	if len(req.QueryVec) > 0 {
		changes, err := s.SearchDiffs(ctx, req.QueryVec, candidateLimit)
		if err != nil {
			// Diff embeddings might not exist
			changes = nil
		}

		for _, change := range changes {
			// Check if already in results
			found := false
			for _, r := range results {
				if r.Change.ID == change.ID {
					r.DiffScore = 1.0 // Mark as found in diff search
					found = true
					break
				}
			}

			if !found {
				commit, _ := s.GetCommit(change.CommitHash)
				if commit != nil && s.matchesHistoryFilters(commit, []*types.Change{change}, req) {
					results = append(results, &types.HistorySearchResult{
						Change:    change,
						Commit:    commit,
						DiffScore: 1.0,
					})
				}
			}
		}
	}

	// BM25 search on commit messages if no vector
	if len(req.QueryVec) == 0 && req.Query != "" {
		commits, err := s.bm25SearchCommits(ctx, req.Query, candidateLimit)
		if err != nil {
			return nil, err
		}

		for _, commit := range commits {
			changes, _ := s.GetChangesByCommit(commit.Hash)
			if !s.matchesHistoryFilters(commit, changes, req) {
				continue
			}

			for _, change := range changes {
				results = append(results, &types.HistorySearchResult{
					Change:       change,
					Commit:       commit,
					MessageScore: 1.0,
				})
			}
		}
	}

	// Apply time decay scoring
	now := time.Now()
	for _, r := range results {
		age := now.Sub(r.Commit.Date)
		decayDays := 30.0 // Half-life in days
		if req.TimeDecayFactor > 0 {
			decayDays = float64(30 / req.TimeDecayFactor)
		}
		decayFactor := math.Exp(-age.Hours() / (24 * decayDays * math.Ln2))
		r.RecencyScore = float32(decayFactor)

		// Combined score
		r.Score = (r.MessageScore*0.4 + r.DiffScore*0.4 + r.RecencyScore*0.2)
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	// Add context if requested
	if req.IncludeContext {
		for _, r := range results {
			// Get related changes in same commit
			allChanges, _ := s.GetChangesByCommit(r.Commit.Hash)
			for _, c := range allChanges {
				if c.ID != r.Change.ID {
					r.RelatedChanges = append(r.RelatedChanges, c)
				}
			}
		}
	}

	return results, nil
}

// SearchCommitMessages searches commit messages by vector similarity.
func (s *Store) SearchCommitMessages(ctx context.Context, queryVec []float32, limit int) ([]*types.Commit, error) {
	embBytes := floatsToBytes(queryVec)

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			ce.commit_hash,
			vec_distance_cosine(ce.embedding, ?) as distance
		FROM commit_embeddings ce
		ORDER BY distance ASC
		LIMIT ?
	`, embBytes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []*types.Commit
	for rows.Next() {
		var hash string
		var distance float64
		if err := rows.Scan(&hash, &distance); err != nil {
			return nil, err
		}

		commit, err := s.GetCommit(hash)
		if err != nil {
			continue
		}
		if commit != nil {
			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// SearchDiffs searches diff content by vector similarity.
func (s *Store) SearchDiffs(ctx context.Context, queryVec []float32, limit int) ([]*types.Change, error) {
	embBytes := floatsToBytes(queryVec)

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			ce.change_id,
			vec_distance_cosine(ce.embedding, ?) as distance
		FROM change_embeddings ce
		ORDER BY distance ASC
		LIMIT ?
	`, embBytes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []*types.Change
	for rows.Next() {
		var id string
		var distance float64
		if err := rows.Scan(&id, &distance); err != nil {
			return nil, err
		}

		change, err := s.GetChange(id)
		if err != nil {
			continue
		}
		if change != nil {
			changes = append(changes, change)
		}
	}

	return changes, nil
}

// bm25SearchCommits searches commits using BM25.
func (s *Store) bm25SearchCommits(ctx context.Context, query string, limit int) ([]*types.Commit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT hash, bm25(commits_fts) as score
		FROM commits_fts
		WHERE commits_fts MATCH ?
		ORDER BY score
		LIMIT ?
	`, escapeFTSQuery(query), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []*types.Commit
	for rows.Next() {
		var hash string
		var score float64
		if err := rows.Scan(&hash, &score); err != nil {
			return nil, err
		}

		commit, err := s.GetCommit(hash)
		if err != nil {
			continue
		}
		if commit != nil {
			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// matchesHistoryFilters checks if a commit/changes match the search filters.
func (s *Store) matchesHistoryFilters(commit *types.Commit, changes []*types.Change, req *types.HistorySearchRequest) bool {
	// Time filters
	if req.TimeFrom != nil && commit.Date.Before(*req.TimeFrom) {
		return false
	}
	if req.TimeTo != nil && commit.Date.After(*req.TimeTo) {
		return false
	}

	// Author filter
	if len(req.Authors) > 0 {
		found := false
		for _, author := range req.Authors {
			if strings.EqualFold(commit.AuthorEmail, author) || strings.EqualFold(commit.Author, author) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Path filter
	if len(req.Paths) > 0 {
		found := false
		for _, change := range changes {
			for _, pattern := range req.Paths {
				if matchGlob(pattern, change.FilePath) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Change type filter
	if len(req.ChangeTypes) > 0 {
		found := false
		for _, change := range changes {
			for _, ct := range req.ChangeTypes {
				if change.ChangeType == ct {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Function filter
	if len(req.Functions) > 0 {
		found := false
		for _, change := range changes {
			for _, fn := range change.AffectedFunctions {
				for _, reqFn := range req.Functions {
					if strings.Contains(strings.ToLower(fn), strings.ToLower(reqFn)) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// FindRegressionCandidates finds commits that might have introduced a bug.
func (s *Store) FindRegressionCandidates(ctx context.Context, description string, knownGood, knownBad string, limit int) ([]*types.RegressionCandidate, error) {
	// Get commits in range
	commits, err := s.GetCommitRange(knownGood, knownBad)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, nil
	}

	// Simple keyword matching for now (can be enhanced with embeddings)
	keywords := strings.Fields(strings.ToLower(description))

	var candidates []*types.RegressionCandidate
	for _, commit := range commits {
		changes, _ := s.GetChangesByCommit(commit.Hash)

		score := float32(0)
		var reasoning []string

		// Check commit message
		msgLower := strings.ToLower(commit.Message)
		for _, kw := range keywords {
			if strings.Contains(msgLower, kw) {
				score += 0.2
				reasoning = append(reasoning, fmt.Sprintf("Commit message contains '%s'", kw))
			}
		}

		// Check affected functions
		for _, change := range changes {
			for _, fn := range change.AffectedFunctions {
				fnLower := strings.ToLower(fn)
				for _, kw := range keywords {
					if strings.Contains(fnLower, kw) {
						score += 0.3
						reasoning = append(reasoning, fmt.Sprintf("Changed function '%s' matches keyword", fn))
					}
				}
			}

			// Check diff content
			if change.DiffContent != "" {
				diffLower := strings.ToLower(change.DiffContent)
				for _, kw := range keywords {
					if strings.Contains(diffLower, kw) {
						score += 0.1
						reasoning = append(reasoning, fmt.Sprintf("Diff content contains '%s'", kw))
					}
				}
			}
		}

		// Prefer more recent commits (more likely to be the regression point)
		if len(commits) > 1 {
			idx := 0
			for i, c := range commits {
				if c.Hash == commit.Hash {
					idx = i
					break
				}
			}
			recencyBonus := float32(idx) / float32(len(commits)) * 0.2
			score += recencyBonus
		}

		if score > 0 {
			// Get diff preview
			var diffPreview string
			if len(changes) > 0 && changes[0].DiffContent != "" {
				diff := changes[0].DiffContent
				if len(diff) > 500 {
					diff = diff[:500] + "..."
				}
				diffPreview = diff
			}

			candidates = append(candidates, &types.RegressionCandidate{
				Commit:      commit,
				Changes:     changes,
				Score:       score,
				Reasoning:   reasoning,
				DiffPreview: diffPreview,
			})
		}
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// GetContributorInsights returns contributor information for code areas.
func (s *Store) GetContributorInsights(ctx context.Context, paths []string, symbol string) ([]*types.ContributorInsight, error) {
	// Build query based on paths or symbol
	var query string
	var args []any

	if len(paths) > 0 {
		// Build path conditions
		conditions := make([]string, len(paths))
		for i, p := range paths {
			conditions[i] = "c.file_path LIKE ?"
			args = append(args, strings.ReplaceAll(p, "*", "%"))
		}
		query = fmt.Sprintf(`
			SELECT cm.author, cm.author_email, COUNT(*) as commits,
			       SUM(c.additions + c.deletions) as lines,
			       MAX(cm.date) as last_active
			FROM changes c
			JOIN commits cm ON c.commit_hash = cm.hash
			WHERE %s
			GROUP BY cm.author_email
			ORDER BY commits DESC
		`, strings.Join(conditions, " OR "))
	} else if symbol != "" {
		query = `
			SELECT cm.author, cm.author_email, COUNT(*) as commits,
			       SUM(c.additions + c.deletions) as lines,
			       MAX(cm.date) as last_active
			FROM changes c
			JOIN commits cm ON c.commit_hash = cm.hash
			WHERE c.affected_functions LIKE ?
			GROUP BY cm.author_email
			ORDER BY commits DESC
		`
		args = append(args, "%"+symbol+"%")
	} else {
		return nil, fmt.Errorf("either paths or symbol is required")
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var insights []*types.ContributorInsight
	var maxCommits int

	for rows.Next() {
		var insight types.ContributorInsight
		var lastActiveUnix int64

		err := rows.Scan(&insight.Author, &insight.Email, &insight.CommitCount,
			&insight.LinesChanged, &lastActiveUnix)
		if err != nil {
			return nil, err
		}

		insight.LastActive = time.Unix(lastActiveUnix, 0)

		if insight.CommitCount > maxCommits {
			maxCommits = insight.CommitCount
		}

		insights = append(insights, &insight)
	}

	// Calculate expertise scores
	for _, insight := range insights {
		insight.Expertise = float32(insight.CommitCount) / float32(maxCommits)
	}

	return insights, nil
}

// GetGitHistoryStats returns statistics about indexed git history.
func (s *Store) GetGitHistoryStats() (*types.GitHistoryStats, error) {
	stats := &types.GitHistoryStats{}

	// Count commits
	row := s.db.QueryRow("SELECT COUNT(*) FROM commits")
	if err := row.Scan(&stats.TotalCommits); err != nil {
		return nil, err
	}

	// Count changes
	row = s.db.QueryRow("SELECT COUNT(*) FROM changes")
	if err := row.Scan(&stats.TotalChanges); err != nil {
		return nil, err
	}

	// Count commits with embeddings
	row = s.db.QueryRow("SELECT COUNT(*) FROM commit_embeddings")
	row.Scan(&stats.CommitsWithEmbeddings) // Ignore error if table doesn't exist

	// Count changes with embeddings
	row = s.db.QueryRow("SELECT COUNT(*) FROM change_embeddings")
	row.Scan(&stats.ChangesWithEmbeddings) // Ignore error if table doesn't exist

	// Get date range
	row = s.db.QueryRow("SELECT MIN(date), MAX(date) FROM commits")
	var minDate, maxDate sql.NullInt64
	if err := row.Scan(&minDate, &maxDate); err == nil {
		if minDate.Valid {
			stats.OldestCommit = time.Unix(minDate.Int64, 0)
		}
		if maxDate.Valid {
			stats.NewestCommit = time.Unix(maxDate.Int64, 0)
		}
	}

	// Count unique authors
	row = s.db.QueryRow("SELECT COUNT(DISTINCT author_email) FROM commits")
	row.Scan(&stats.UniqueAuthors)

	// Get last indexed time
	row = s.db.QueryRow("SELECT value FROM git_index_meta WHERE key = 'last_indexed_time'")
	var lastIndexedStr string
	if err := row.Scan(&lastIndexedStr); err == nil {
		if t, err := time.Parse(time.RFC3339, lastIndexedStr); err == nil {
			stats.LastIndexed = t
		}
	}

	return stats, nil
}

// Helper functions

func scanCommit(row *sql.Row) (*types.Commit, error) {
	var c types.Commit
	var dateUnix int64
	var parentHash, tags, branch sql.NullString

	err := row.Scan(
		&c.Hash, &c.ShortHash, &c.Author, &c.AuthorEmail,
		&dateUnix, &c.Message, &parentHash,
		&c.FilesChanged, &c.Insertions, &c.Deletions,
		&c.IsMerge, &c.IsTagged, &tags, &branch,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.Date = time.Unix(dateUnix, 0)
	c.ParentHash = parentHash.String
	c.Branch = branch.String

	if tags.Valid && tags.String != "" {
		json.Unmarshal([]byte(tags.String), &c.Tags)
	}

	return &c, nil
}

func scanCommits(rows *sql.Rows) ([]*types.Commit, error) {
	var commits []*types.Commit

	for rows.Next() {
		var c types.Commit
		var dateUnix int64
		var parentHash, tags, branch sql.NullString

		err := rows.Scan(
			&c.Hash, &c.ShortHash, &c.Author, &c.AuthorEmail,
			&dateUnix, &c.Message, &parentHash,
			&c.FilesChanged, &c.Insertions, &c.Deletions,
			&c.IsMerge, &c.IsTagged, &tags, &branch,
		)
		if err != nil {
			return nil, err
		}

		c.Date = time.Unix(dateUnix, 0)
		c.ParentHash = parentHash.String
		c.Branch = branch.String

		if tags.Valid && tags.String != "" {
			json.Unmarshal([]byte(tags.String), &c.Tags)
		}

		commits = append(commits, &c)
	}

	return commits, nil
}

func scanChange(row *sql.Row) (*types.Change, error) {
	var c types.Change
	var changeType string
	var oldPath, diffContent, affectedFuncs, affectedChunks, hunks sql.NullString

	err := row.Scan(
		&c.ID, &c.CommitHash, &c.FilePath, &changeType, &oldPath,
		&diffContent, &c.Additions, &c.Deletions,
		&affectedFuncs, &affectedChunks, &hunks,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.ChangeType = types.ChangeType(changeType)
	c.OldPath = oldPath.String
	c.DiffContent = diffContent.String

	if affectedFuncs.Valid && affectedFuncs.String != "" {
		json.Unmarshal([]byte(affectedFuncs.String), &c.AffectedFunctions)
	}
	if affectedChunks.Valid && affectedChunks.String != "" {
		json.Unmarshal([]byte(affectedChunks.String), &c.AffectedChunkIDs)
	}
	if hunks.Valid && hunks.String != "" {
		json.Unmarshal([]byte(hunks.String), &c.Hunks)
	}

	return &c, nil
}

func scanChanges(rows *sql.Rows) ([]*types.Change, error) {
	var changes []*types.Change

	for rows.Next() {
		var c types.Change
		var changeType string
		var oldPath, diffContent, affectedFuncs, affectedChunks, hunks sql.NullString

		err := rows.Scan(
			&c.ID, &c.CommitHash, &c.FilePath, &changeType, &oldPath,
			&diffContent, &c.Additions, &c.Deletions,
			&affectedFuncs, &affectedChunks, &hunks,
		)
		if err != nil {
			return nil, err
		}

		c.ChangeType = types.ChangeType(changeType)
		c.OldPath = oldPath.String
		c.DiffContent = diffContent.String

		if affectedFuncs.Valid && affectedFuncs.String != "" {
			json.Unmarshal([]byte(affectedFuncs.String), &c.AffectedFunctions)
		}
		if affectedChunks.Valid && affectedChunks.String != "" {
			json.Unmarshal([]byte(affectedChunks.String), &c.AffectedChunkIDs)
		}
		if hunks.Valid && hunks.String != "" {
			json.Unmarshal([]byte(hunks.String), &c.Hunks)
		}

		changes = append(changes, &c)
	}

	return changes, nil
}

// matchGlob performs simple glob matching.
func matchGlob(pattern, str string) bool {
	// Simple implementation - supports * and **
	if pattern == "*" || pattern == "**" {
		return true
	}

	pattern = strings.ReplaceAll(pattern, "**", ".*")
	pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
	pattern = "^" + pattern + "$"

	// Use strings.Contains for now as a simple fallback
	patternClean := strings.ReplaceAll(pattern, ".*", "")
	patternClean = strings.ReplaceAll(patternClean, "[^/]*", "")
	patternClean = strings.Trim(patternClean, "^$")

	return strings.Contains(str, patternClean)
}
