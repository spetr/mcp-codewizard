// Package sqlitevec implements VectorStore using sqlite-vec for vector search
// and FTS5 for BM25 full-text search.
package sqlitevec

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

var (
	// Ensure sqlite-vec Auto() is called exactly once before any db connection
	vecAutoOnce sync.Once
)

// SchemaVersion is incremented when schema changes require reindexing.
const SchemaVersion = 1

// Store implements the VectorStore interface using sqlite-vec.
type Store struct {
	db             *sql.DB
	path           string
	dimensions     int
	enableFTS      bool
	vectorTableSQL string
}

// New creates a new sqlite-vec store.
func New() *Store {
	return &Store{
		enableFTS: true,
	}
}

// Name returns the store name.
func (s *Store) Name() string {
	return "sqlitevec"
}

// Init initializes the store at the given path.
func (s *Store) Init(path string) error {
	s.path = path

	// Register sqlite-vec extension before opening any database connection.
	// This must be called once before sql.Open() to ensure vec_* functions are available.
	vecAutoOnce.Do(func() {
		sqlite_vec.Auto()
	})

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with sqlite-vec extension
	// WAL mode for concurrent reads, busy_timeout to wait for locks instead of failing immediately
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Enable sqlite-vec extension
	if _, err := db.Exec("SELECT vec_version()"); err != nil {
		return fmt.Errorf("sqlite-vec extension not available: %w", err)
	}

	// Create schema
	if err := s.createSchema(); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Create git history schema
	if err := s.createGitHistorySchema(); err != nil {
		return fmt.Errorf("failed to create git history schema: %w", err)
	}

	// Create memory schema
	if err := s.createMemorySchema(); err != nil {
		return fmt.Errorf("failed to create memory schema: %w", err)
	}

	// Create todo schema
	if err := s.createTodoSchema(); err != nil {
		return fmt.Errorf("failed to create todo schema: %w", err)
	}

	// Check FTS health and auto-repair if corrupted
	if err := s.CheckFTSHealth(); err != nil {
		slog.Warn("FTS index unhealthy, rebuilding", "error", err)
		if rebuildErr := s.RebuildFTS(); rebuildErr != nil {
			slog.Error("failed to rebuild FTS index", "error", rebuildErr)
			// Continue anyway - search will work without FTS
		} else {
			slog.Info("FTS index rebuilt successfully")
		}
	}

	return nil
}

// createSchema creates all necessary tables.
func (s *Store) createSchema() error {
	// Metadata table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Chunks table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS chunks (
			id TEXT PRIMARY KEY,
			file_path TEXT NOT NULL,
			language TEXT NOT NULL,
			content TEXT NOT NULL,
			chunk_type TEXT NOT NULL,
			name TEXT,
			parent_name TEXT,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL,
			hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Index on file_path for deletion
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path)`)
	if err != nil {
		return err
	}

	// FTS5 for BM25 search
	if s.enableFTS {
		_, err = s.db.Exec(`
			CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
				id,
				content,
				name,
				content='chunks',
				content_rowid='rowid',
				tokenize='porter unicode61'
			)
		`)
		if err != nil {
			return err
		}

		// Triggers to keep FTS in sync
		_, err = s.db.Exec(`
			CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
				INSERT INTO chunks_fts(rowid, id, content, name)
				VALUES (new.rowid, new.id, new.content, new.name);
			END
		`)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`
			CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
				INSERT INTO chunks_fts(chunks_fts, rowid, id, content, name)
				VALUES('delete', old.rowid, old.id, old.content, old.name);
			END
		`)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`
			CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
				INSERT INTO chunks_fts(chunks_fts, rowid, id, content, name)
				VALUES('delete', old.rowid, old.id, old.content, old.name);
				INSERT INTO chunks_fts(rowid, id, content, name)
				VALUES (new.rowid, new.id, new.content, new.name);
			END
		`)
		if err != nil {
			return err
		}
	}

	// Symbols table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS symbols (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			file_path TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL,
			line_count INTEGER NOT NULL DEFAULT 1,
			signature TEXT,
			visibility TEXT,
			doc_comment TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_symbols_file_path ON symbols(file_path)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_symbols_line_count ON symbols(line_count)`)
	if err != nil {
		return err
	}

	// References table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS refs (
			id TEXT PRIMARY KEY,
			from_symbol TEXT NOT NULL,
			to_symbol TEXT NOT NULL,
			kind TEXT NOT NULL,
			file_path TEXT NOT NULL,
			line INTEGER NOT NULL,
			is_external BOOLEAN NOT NULL DEFAULT FALSE
		)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_refs_from ON refs(from_symbol)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_refs_to ON refs(to_symbol)`)
	if err != nil {
		return err
	}

	// File cache table for incremental indexing
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS file_cache (
			file_path TEXT PRIMARY KEY,
			file_hash TEXT NOT NULL,
			config_hash TEXT NOT NULL,
			indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

// createVectorTable creates the vector table with the specified dimensions.
func (s *Store) createVectorTable(dimensions int) error {
	if s.dimensions == dimensions {
		return nil // Already created
	}

	s.dimensions = dimensions

	// Drop existing vector table if dimensions changed
	_, _ = s.db.Exec("DROP TABLE IF EXISTS chunk_embeddings")

	// Create vector table using sqlite-vec
	_, err := s.db.Exec(fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS chunk_embeddings USING vec0(
			chunk_id TEXT PRIMARY KEY,
			embedding float[%d]
		)
	`, dimensions))
	if err != nil {
		return fmt.Errorf("failed to create vector table: %w", err)
	}

	return nil
}

// Close releases resources and closes connections.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// StoreChunks stores chunks with their embeddings.
func (s *Store) StoreChunks(chunks []*types.ChunkWithEmbedding) error {
	if len(chunks) == 0 {
		return nil
	}

	// Ensure vector table is created with correct dimensions
	if len(chunks[0].Embedding) > 0 {
		if err := s.createVectorTable(len(chunks[0].Embedding)); err != nil {
			return err
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statements
	chunkStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO chunks
		(id, file_path, language, content, chunk_type, name, parent_name, start_line, end_line, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer chunkStmt.Close()

	embeddingStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO chunk_embeddings (chunk_id, embedding)
		VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer embeddingStmt.Close()

	for _, cwe := range chunks {
		c := cwe.Chunk

		// Store chunk
		_, err := chunkStmt.Exec(
			c.ID, c.FilePath, c.Language, c.Content,
			string(c.ChunkType), c.Name, c.ParentName,
			c.StartLine, c.EndLine, c.Hash,
		)
		if err != nil {
			return fmt.Errorf("failed to store chunk %s: %w", c.ID, err)
		}

		// Store embedding
		if len(cwe.Embedding) > 0 {
			embBytes := floatsToBytes(cwe.Embedding)
			_, err := embeddingStmt.Exec(c.ID, embBytes)
			if err != nil {
				return fmt.Errorf("failed to store embedding for %s: %w", c.ID, err)
			}
		}
	}

	return tx.Commit()
}

// GetChunk retrieves a chunk by ID.
func (s *Store) GetChunk(id string) (*types.Chunk, error) {
	row := s.db.QueryRow(`
		SELECT id, file_path, language, content, chunk_type, name, parent_name, start_line, end_line, hash
		FROM chunks WHERE id = ?
	`, id)

	var chunk types.Chunk
	var chunkType string
	var name, parentName sql.NullString

	err := row.Scan(
		&chunk.ID, &chunk.FilePath, &chunk.Language, &chunk.Content,
		&chunkType, &name, &parentName, &chunk.StartLine, &chunk.EndLine, &chunk.Hash,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	chunk.ChunkType = types.ChunkType(chunkType)
	chunk.Name = name.String
	chunk.ParentName = parentName.String

	return &chunk, nil
}

// DeleteChunksByFile removes all chunks for a file.
func (s *Store) DeleteChunksByFile(filePath string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get chunk IDs first
	rows, err := tx.Query("SELECT id FROM chunks WHERE file_path = ?", filePath)
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
		_, err := tx.Exec("DELETE FROM chunk_embeddings WHERE chunk_id = ?", id)
		if err != nil {
			return err
		}
	}

	// Delete chunks (FTS will be updated by trigger)
	_, err = tx.Exec("DELETE FROM chunks WHERE file_path = ?", filePath)
	if err != nil {
		return err
	}

	// Delete symbols and references for this file
	_, err = tx.Exec("DELETE FROM symbols WHERE file_path = ?", filePath)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM refs WHERE file_path = ?", filePath)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Search performs hybrid search (BM25 + vector).
func (s *Store) Search(ctx context.Context, req *types.SearchRequest) ([]*types.SearchResult, error) {
	switch req.Mode {
	case types.SearchModeVector:
		return s.vectorSearch(ctx, req)
	case types.SearchModeBM25:
		return s.bm25Search(ctx, req)
	case types.SearchModeHybrid:
		return s.hybridSearch(ctx, req)
	default:
		return s.hybridSearch(ctx, req) // Default to hybrid
	}
}

// vectorSearch performs pure vector similarity search.
func (s *Store) vectorSearch(ctx context.Context, req *types.SearchRequest) ([]*types.SearchResult, error) {
	if len(req.QueryVec) == 0 {
		return nil, errors.New("query vector is required for vector search")
	}

	limit := req.Limit
	if req.UseReranker && req.RerankCandidates > 0 {
		limit = req.RerankCandidates
	}

	embBytes := floatsToBytes(req.QueryVec)

	// Vector similarity search using sqlite-vec
	query := `
		SELECT
			ce.chunk_id,
			vec_distance_cosine(ce.embedding, ?) as distance,
			c.file_path, c.language, c.content, c.chunk_type,
			c.name, c.parent_name, c.start_line, c.end_line, c.hash
		FROM chunk_embeddings ce
		JOIN chunks c ON ce.chunk_id = c.id
	`

	args := []any{embBytes}

	// Add filters
	whereClauses := []string{}
	if req.Filters != nil {
		if len(req.Filters.Languages) > 0 {
			placeholders := make([]string, len(req.Filters.Languages))
			for i, lang := range req.Filters.Languages {
				placeholders[i] = "?"
				args = append(args, lang)
			}
			whereClauses = append(whereClauses, "c.language IN ("+strings.Join(placeholders, ",")+")")
		}
		if len(req.Filters.ChunkTypes) > 0 {
			placeholders := make([]string, len(req.Filters.ChunkTypes))
			for i, ct := range req.Filters.ChunkTypes {
				placeholders[i] = "?"
				args = append(args, string(ct))
			}
			whereClauses = append(whereClauses, "c.chunk_type IN ("+strings.Join(placeholders, ",")+")")
		}
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY distance ASC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer rows.Close()

	var results []*types.SearchResult
	for rows.Next() {
		var (
			chunkID   string
			distance  float64
			chunk     types.Chunk
			chunkType string
		)

		err := rows.Scan(
			&chunkID, &distance,
			&chunk.FilePath, &chunk.Language, &chunk.Content, &chunkType,
			&chunk.Name, &chunk.ParentName, &chunk.StartLine, &chunk.EndLine, &chunk.Hash,
		)
		if err != nil {
			return nil, err
		}

		chunk.ID = chunkID
		chunk.ChunkType = types.ChunkType(chunkType)

		// Convert distance to similarity score (cosine distance → similarity)
		score := float32(1.0 - distance)

		results = append(results, &types.SearchResult{
			Chunk:       &chunk,
			Score:       score,
			VectorScore: score,
		})
	}

	return results, nil
}

// bm25Search performs BM25 full-text search.
func (s *Store) bm25Search(ctx context.Context, req *types.SearchRequest) ([]*types.SearchResult, error) {
	if req.Query == "" {
		return nil, errors.New("query text is required for BM25 search")
	}

	limit := req.Limit
	if req.UseReranker && req.RerankCandidates > 0 {
		limit = req.RerankCandidates
	}

	// FTS5 BM25 search
	query := `
		SELECT
			c.id, bm25(chunks_fts) as bm25_score,
			c.file_path, c.language, c.content, c.chunk_type,
			c.name, c.parent_name, c.start_line, c.end_line, c.hash
		FROM chunks_fts fts
		JOIN chunks c ON fts.id = c.id
		WHERE chunks_fts MATCH ?
	`

	args := []any{escapeFTSQuery(req.Query)}

	// Add filters
	if req.Filters != nil {
		if len(req.Filters.Languages) > 0 {
			placeholders := make([]string, len(req.Filters.Languages))
			for i, lang := range req.Filters.Languages {
				placeholders[i] = "?"
				args = append(args, lang)
			}
			query += " AND c.language IN (" + strings.Join(placeholders, ",") + ")"
		}
		if len(req.Filters.ChunkTypes) > 0 {
			placeholders := make([]string, len(req.Filters.ChunkTypes))
			for i, ct := range req.Filters.ChunkTypes {
				placeholders[i] = "?"
				args = append(args, string(ct))
			}
			query += " AND c.chunk_type IN (" + strings.Join(placeholders, ",") + ")"
		}
	}

	query += " ORDER BY bm25_score LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("BM25 search failed: %w", err)
	}
	defer rows.Close()

	var results []*types.SearchResult
	for rows.Next() {
		var (
			chunkID   string
			bm25Score float64
			chunk     types.Chunk
			chunkType string
		)

		err := rows.Scan(
			&chunkID, &bm25Score,
			&chunk.FilePath, &chunk.Language, &chunk.Content, &chunkType,
			&chunk.Name, &chunk.ParentName, &chunk.StartLine, &chunk.EndLine, &chunk.Hash,
		)
		if err != nil {
			return nil, err
		}

		chunk.ID = chunkID
		chunk.ChunkType = types.ChunkType(chunkType)

		// BM25 scores are negative (lower is better), normalize to 0-1
		score := float32(1.0 / (1.0 + math.Abs(bm25Score)))

		results = append(results, &types.SearchResult{
			Chunk:     &chunk,
			Score:     score,
			BM25Score: score,
		})
	}

	return results, nil
}

// hybridSearch combines vector and BM25 search with weighted scoring.
func (s *Store) hybridSearch(ctx context.Context, req *types.SearchRequest) ([]*types.SearchResult, error) {
	// Get candidates from both search methods
	candidateLimit := req.Limit * 3 // Get more candidates for combining
	if req.UseReranker && req.RerankCandidates > 0 {
		candidateLimit = req.RerankCandidates
	}

	// Collect results from both methods
	vectorResults := make(map[string]*types.SearchResult)
	bm25Results := make(map[string]*types.SearchResult)

	// Vector search if we have embeddings
	if len(req.QueryVec) > 0 {
		vecReq := *req
		vecReq.Mode = types.SearchModeVector
		vecReq.Limit = candidateLimit
		vecReq.UseReranker = false

		results, err := s.vectorSearch(ctx, &vecReq)
		if err != nil {
			return nil, err
		}
		for _, r := range results {
			vectorResults[r.Chunk.ID] = r
		}
	}

	// BM25 search if we have query text
	if req.Query != "" {
		bm25Req := *req
		bm25Req.Mode = types.SearchModeBM25
		bm25Req.Limit = candidateLimit
		bm25Req.UseReranker = false

		results, err := s.bm25Search(ctx, &bm25Req)
		if err != nil {
			// BM25 might fail if no FTS index, continue with vector only
			if len(vectorResults) > 0 {
				// Convert to slice and return
				var finalResults []*types.SearchResult
				for _, r := range vectorResults {
					finalResults = append(finalResults, r)
				}
				sort.Slice(finalResults, func(i, j int) bool {
					return finalResults[i].Score > finalResults[j].Score
				})
				if len(finalResults) > req.Limit {
					finalResults = finalResults[:req.Limit]
				}
				return finalResults, nil
			}
			return nil, err
		}
		for _, r := range bm25Results {
			bm25Results[r.Chunk.ID] = r
		}
		// Store results in map
		for _, r := range results {
			bm25Results[r.Chunk.ID] = r
		}
	}

	// Combine results with weighted scoring
	vectorWeight := req.VectorWeight
	bm25Weight := req.BM25Weight
	if vectorWeight == 0 && bm25Weight == 0 {
		vectorWeight = 0.7
		bm25Weight = 0.3
	}

	combinedResults := make(map[string]*types.SearchResult)

	// Process all unique chunks
	for id, vr := range vectorResults {
		result := &types.SearchResult{
			Chunk:       vr.Chunk,
			VectorScore: vr.VectorScore,
		}
		if br, ok := bm25Results[id]; ok {
			result.BM25Score = br.BM25Score
		}
		result.Score = result.VectorScore*vectorWeight + result.BM25Score*bm25Weight
		combinedResults[id] = result
	}

	for id, br := range bm25Results {
		if _, exists := combinedResults[id]; !exists {
			result := &types.SearchResult{
				Chunk:     br.Chunk,
				BM25Score: br.BM25Score,
			}
			result.Score = result.VectorScore*vectorWeight + result.BM25Score*bm25Weight
			combinedResults[id] = result
		}
	}

	// Convert to slice and sort by score
	var results []*types.SearchResult
	for _, r := range combinedResults {
		results = append(results, r)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results, nil
}

// StoreSymbols stores symbols.
func (s *Store) StoreSymbols(symbols []*types.Symbol) error {
	if len(symbols) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO symbols
		(id, name, kind, file_path, start_line, end_line, line_count, signature, visibility, doc_comment)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, sym := range symbols {
		lineCount := sym.LineCount
		if lineCount == 0 {
			lineCount = sym.ComputeLineCount()
		}
		_, err := stmt.Exec(
			sym.ID, sym.Name, string(sym.Kind), sym.FilePath,
			sym.StartLine, sym.EndLine, lineCount, sym.Signature, sym.Visibility, sym.DocComment,
		)
		if err != nil {
			return fmt.Errorf("failed to store symbol %s: %w", sym.ID, err)
		}
	}

	return tx.Commit()
}

// GetSymbol retrieves a symbol by ID.
func (s *Store) GetSymbol(id string) (*types.Symbol, error) {
	row := s.db.QueryRow(`
		SELECT id, name, kind, file_path, start_line, end_line, line_count, signature, visibility, doc_comment
		FROM symbols WHERE id = ?
	`, id)

	var sym types.Symbol
	var kind string
	var signature, visibility, docComment sql.NullString

	err := row.Scan(
		&sym.ID, &sym.Name, &kind, &sym.FilePath,
		&sym.StartLine, &sym.EndLine, &sym.LineCount, &signature, &visibility, &docComment,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	sym.Kind = types.SymbolKind(kind)
	sym.Signature = signature.String
	sym.Visibility = visibility.String
	sym.DocComment = docComment.String

	return &sym, nil
}

// FindSymbols searches symbols by name pattern and kind.
func (s *Store) FindSymbols(query string, kind types.SymbolKind, limit int) ([]*types.Symbol, error) {
	return s.FindSymbolsAdvanced(query, kind, 0, "", limit)
}

// FindSymbolsAdvanced searches symbols with additional filtering options.
// minLines: minimum line count (0 = no filter)
// sortBy: "lines" to sort by line_count descending, "name" for name ascending, "" for default
func (s *Store) FindSymbolsAdvanced(query string, kind types.SymbolKind, minLines int, sortBy string, limit int) ([]*types.Symbol, error) {
	sqlQuery := `
		SELECT id, name, kind, file_path, start_line, end_line, line_count, signature, visibility, doc_comment
		FROM symbols WHERE 1=1
	`
	args := []any{}

	if query != "" {
		sqlQuery += " AND name LIKE ?"
		args = append(args, "%"+query+"%")
	}

	if kind != "" {
		sqlQuery += " AND kind = ?"
		args = append(args, string(kind))
	}

	if minLines > 0 {
		sqlQuery += " AND line_count >= ?"
		args = append(args, minLines)
	}

	switch sortBy {
	case "lines":
		sqlQuery += " ORDER BY line_count DESC"
	case "name":
		sqlQuery += " ORDER BY name ASC"
	default:
		sqlQuery += " ORDER BY name ASC"
	}

	sqlQuery += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []*types.Symbol
	for rows.Next() {
		var sym types.Symbol
		var kindStr string
		var signature, visibility, docComment sql.NullString

		err := rows.Scan(
			&sym.ID, &sym.Name, &kindStr, &sym.FilePath,
			&sym.StartLine, &sym.EndLine, &sym.LineCount, &signature, &visibility, &docComment,
		)
		if err != nil {
			return nil, err
		}

		sym.Kind = types.SymbolKind(kindStr)
		sym.Signature = signature.String
		sym.Visibility = visibility.String
		sym.DocComment = docComment.String

		symbols = append(symbols, &sym)
	}

	return symbols, nil
}

// FindLongFunctions returns functions sorted by line count (longest first).
func (s *Store) FindLongFunctions(minLines int, limit int) ([]*types.Symbol, error) {
	sqlQuery := `
		SELECT id, name, kind, file_path, start_line, end_line, line_count, signature, visibility, doc_comment
		FROM symbols
		WHERE kind IN ('function', 'method')
		AND line_count >= ?
		ORDER BY line_count DESC
		LIMIT ?
	`

	rows, err := s.db.Query(sqlQuery, minLines, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []*types.Symbol
	for rows.Next() {
		var sym types.Symbol
		var kindStr string
		var signature, visibility, docComment sql.NullString

		err := rows.Scan(
			&sym.ID, &sym.Name, &kindStr, &sym.FilePath,
			&sym.StartLine, &sym.EndLine, &sym.LineCount, &signature, &visibility, &docComment,
		)
		if err != nil {
			return nil, err
		}

		sym.Kind = types.SymbolKind(kindStr)
		sym.Signature = signature.String
		sym.Visibility = visibility.String
		sym.DocComment = docComment.String

		symbols = append(symbols, &sym)
	}

	return symbols, nil
}

// StoreReferences stores references.
func (s *Store) StoreReferences(refs []*types.Reference) error {
	if len(refs) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO refs
		(id, from_symbol, to_symbol, kind, file_path, line, is_external)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, ref := range refs {
		_, err := stmt.Exec(
			ref.ID, ref.FromSymbol, ref.ToSymbol, string(ref.Kind),
			ref.FilePath, ref.Line, ref.IsExternal,
		)
		if err != nil {
			return fmt.Errorf("failed to store reference %s: %w", ref.ID, err)
		}
	}

	return tx.Commit()
}

// GetCallers returns references TO a symbol.
func (s *Store) GetCallers(symbolID string, limit int) ([]*types.Reference, error) {
	// Symbol ID format is "filepath:name:line", but refs may store:
	// - Just the name: "GetLanguage"
	// - Qualified name: "dart.GetLanguage", "simple.DetectLanguage"
	// - Full symbol ID (rare)
	symbolName := extractSymbolName(symbolID)
	qualifiedName := extractQualifiedName(symbolID)

	// Build query with all possible matches
	query := `
		SELECT id, from_symbol, to_symbol, kind, file_path, line, is_external
		FROM refs WHERE to_symbol = ? OR to_symbol = ?`
	args := []interface{}{symbolID, symbolName}

	if qualifiedName != "" {
		query += ` OR to_symbol = ?`
		args = append(args, qualifiedName)
	}

	// Also match qualified references ending with .symbolName (e.g., "simple.DetectLanguage")
	query += ` OR to_symbol LIKE ?`
	args = append(args, "%."+symbolName)

	query += ` LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReferences(rows)
}

// extractSymbolName extracts the symbol name from a symbol ID.
// Symbol ID format: "filepath:name:line" -> returns "name"
func extractSymbolName(symbolID string) string {
	parts := strings.Split(symbolID, ":")
	if len(parts) >= 2 {
		// Return the second-to-last part (name), handling cases like:
		// "path/to/file.go:FuncName:123" -> "FuncName"
		// "FuncName" -> "FuncName" (already just a name)
		return parts[len(parts)-2]
	}
	return symbolID
}

// extractQualifiedName extracts a qualified name (pkg.Name) from a symbol ID.
// Symbol ID format: "path/to/pkg/file.go:FuncName:123" -> returns "pkg.FuncName"
// This helps match references like "dart.GetLanguage" to symbol "dart/binding.go:GetLanguage:16"
func extractQualifiedName(symbolID string) string {
	parts := strings.Split(symbolID, ":")
	if len(parts) < 2 {
		return ""
	}

	filePath := strings.Join(parts[:len(parts)-2], ":")
	funcName := parts[len(parts)-2]

	// Extract package name from file path (last directory component)
	// e.g., "path/to/dart/binding.go" -> "dart"
	dir := filepath.Dir(filePath)
	pkgName := filepath.Base(dir)

	// Skip common non-package directories
	if pkgName == "." || pkgName == "/" || pkgName == "" {
		return ""
	}

	return pkgName + "." + funcName
}

// GetCallees returns references FROM a symbol.
func (s *Store) GetCallees(symbolID string, limit int) ([]*types.Reference, error) {
	// Symbol ID format is "filepath:name:line", but refs store only "name" in from_symbol.
	// Extract the symbol name to match against refs.
	symbolName := extractSymbolName(symbolID)

	rows, err := s.db.Query(`
		SELECT id, from_symbol, to_symbol, kind, file_path, line, is_external
		FROM refs WHERE from_symbol = ? OR from_symbol = ? LIMIT ?
	`, symbolID, symbolName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReferences(rows)
}

// FindReferencesByKind returns all references of a specific kind.
func (s *Store) FindReferencesByKind(kind types.RefKind, limit int) ([]*types.Reference, error) {
	rows, err := s.db.Query(`
		SELECT id, from_symbol, to_symbol, kind, file_path, line, is_external
		FROM refs WHERE kind = ? LIMIT ?
	`, string(kind), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReferences(rows)
}

// GetAllReferences returns all references for building call graphs.
func (s *Store) GetAllReferences(limit int) ([]*types.Reference, error) {
	rows, err := s.db.Query(`
		SELECT id, from_symbol, to_symbol, kind, file_path, line, is_external
		FROM refs WHERE is_external = 0 LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReferences(rows)
}

func scanReferences(rows *sql.Rows) ([]*types.Reference, error) {
	var refs []*types.Reference
	for rows.Next() {
		var ref types.Reference
		var kind string

		err := rows.Scan(
			&ref.ID, &ref.FromSymbol, &ref.ToSymbol, &kind,
			&ref.FilePath, &ref.Line, &ref.IsExternal,
		)
		if err != nil {
			return nil, err
		}

		ref.Kind = types.RefKind(kind)
		refs = append(refs, &ref)
	}
	return refs, nil
}

// GetMetadata returns index metadata.
func (s *Store) GetMetadata() (*types.IndexMetadata, error) {
	row := s.db.QueryRow("SELECT value FROM metadata WHERE key = 'index_metadata'")

	var jsonData string
	err := row.Scan(&jsonData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var meta types.IndexMetadata
	if err := json.Unmarshal([]byte(jsonData), &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// SetMetadata stores index metadata.
func (s *Store) SetMetadata(meta *types.IndexMetadata) error {
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO metadata (key, value) VALUES ('index_metadata', ?)
	`, string(jsonData))
	return err
}

// GetStats returns store statistics.
func (s *Store) GetStats() (*types.StoreStats, error) {
	stats := &types.StoreStats{}

	// Count chunks
	row := s.db.QueryRow("SELECT COUNT(*) FROM chunks")
	if err := row.Scan(&stats.TotalChunks); err != nil {
		return nil, err
	}

	// Count symbols
	row = s.db.QueryRow("SELECT COUNT(*) FROM symbols")
	if err := row.Scan(&stats.TotalSymbols); err != nil {
		return nil, err
	}

	// Count references
	row = s.db.QueryRow("SELECT COUNT(*) FROM refs")
	if err := row.Scan(&stats.TotalReferences); err != nil {
		return nil, err
	}

	// Count unique files
	row = s.db.QueryRow("SELECT COUNT(DISTINCT file_path) FROM chunks")
	if err := row.Scan(&stats.IndexedFiles); err != nil {
		return nil, err
	}

	// Get DB file size
	if info, err := os.Stat(s.path); err == nil {
		stats.DBSizeBytes = info.Size()
	}

	// Get last indexed time
	meta, err := s.GetMetadata()
	if err == nil && meta != nil {
		stats.LastIndexed = meta.LastUpdated
	}

	return stats, nil
}

// GetFileHash returns the cached hash for a file.
func (s *Store) GetFileHash(filePath string) (string, error) {
	row := s.db.QueryRow("SELECT file_hash FROM file_cache WHERE file_path = ?", filePath)

	var hash string
	err := row.Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

// SetFileHash stores the hash for a file.
func (s *Store) SetFileHash(filePath, hash, configHash string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO file_cache (file_path, file_hash, config_hash, indexed_at)
		VALUES (?, ?, ?, ?)
	`, filePath, hash, configHash, time.Now())
	return err
}

// GetAllFileHashes returns all cached file hashes.
func (s *Store) GetAllFileHashes() (map[string]string, error) {
	rows, err := s.db.Query("SELECT file_path, file_hash FROM file_cache")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make(map[string]string)
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, err
		}
		hashes[path] = hash
	}

	return hashes, nil
}

// DeleteFileCache removes file from cache.
func (s *Store) DeleteFileCache(filePath string) error {
	_, err := s.db.Exec("DELETE FROM file_cache WHERE file_path = ?", filePath)
	return err
}

// Helper functions

// floatsToBytes converts float32 slice to bytes for sqlite-vec.
func floatsToBytes(floats []float32) []byte {
	bytes := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := math.Float32bits(f)
		bytes[i*4] = byte(bits)
		bytes[i*4+1] = byte(bits >> 8)
		bytes[i*4+2] = byte(bits >> 16)
		bytes[i*4+3] = byte(bits >> 24)
	}
	return bytes
}

// escapeFTSQuery escapes special characters in FTS5 query.
func escapeFTSQuery(query string) string {
	// FTS5 special characters that need escaping
	special := []string{"*", "\"", "(", ")", ":", "-", "^", "~"}
	result := query
	for _, s := range special {
		result = strings.ReplaceAll(result, s, "\""+s+"\"")
	}
	return result
}

// CheckFTSHealth verifies that the FTS index is in sync with the chunks table.
// Returns nil if healthy, error describing the issue otherwise.
func (s *Store) CheckFTSHealth() error {
	if !s.enableFTS {
		return nil
	}

	// Check if FTS table exists
	var exists int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='chunks_fts'
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check FTS table existence: %w", err)
	}
	if exists == 0 {
		return nil // FTS not created yet, will be created on first use
	}

	// Try a simple query that exercises the FTS JOIN
	// This will fail if there are orphaned FTS entries
	_, err = s.db.Exec(`
		SELECT c.id FROM chunks_fts fts
		JOIN chunks c ON fts.rowid = c.rowid
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("FTS index corrupted: %w", err)
	}

	return nil
}

// RebuildFTS rebuilds the FTS index from the chunks table.
// This fixes corruption issues where FTS has references to deleted rows.
func (s *Store) RebuildFTS() error {
	if !s.enableFTS {
		return nil
	}

	_, err := s.db.Exec(`INSERT INTO chunks_fts(chunks_fts) VALUES('rebuild')`)
	if err != nil {
		return fmt.Errorf("failed to rebuild FTS index: %w", err)
	}

	return nil
}

// Ensure Store implements VectorStore interface
var _ provider.VectorStore = (*Store)(nil)
