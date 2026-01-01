// Package provider defines interfaces for pluggable components.
package provider

import (
	"context"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

// ChunkStore handles chunk storage operations.
type ChunkStore interface {
	// StoreChunks stores chunks with their embeddings.
	StoreChunks(chunks []*types.ChunkWithEmbedding) error

	// GetChunk retrieves a chunk by ID.
	GetChunk(id string) (*types.Chunk, error)

	// DeleteChunksByFile removes all chunks for a file.
	DeleteChunksByFile(filePath string) error
}

// SymbolStore handles symbol storage operations.
type SymbolStore interface {
	// StoreSymbols stores symbols.
	StoreSymbols(symbols []*types.Symbol) error

	// GetSymbol retrieves a symbol by ID.
	GetSymbol(id string) (*types.Symbol, error)

	// FindSymbols searches symbols by name pattern and kind.
	FindSymbols(query string, kind types.SymbolKind, limit int) ([]*types.Symbol, error)

	// FindSymbolsAdvanced searches symbols with additional filtering options.
	// minLines: minimum line count (0 = no filter)
	// sortBy: "lines" for line_count descending, "name" for name ascending
	FindSymbolsAdvanced(query string, kind types.SymbolKind, minLines int, sortBy string, limit int) ([]*types.Symbol, error)

	// FindLongFunctions returns functions sorted by line count (longest first).
	FindLongFunctions(minLines int, limit int) ([]*types.Symbol, error)
}

// ReferenceStore handles reference storage operations.
type ReferenceStore interface {
	// StoreReferences stores references.
	StoreReferences(refs []*types.Reference) error

	// GetCallers returns references TO a symbol.
	GetCallers(symbolID string, limit int) ([]*types.Reference, error)

	// GetCallees returns references FROM a symbol.
	GetCallees(symbolID string, limit int) ([]*types.Reference, error)

	// FindReferencesByKind returns all references of a specific kind.
	FindReferencesByKind(kind types.RefKind, limit int) ([]*types.Reference, error)

	// GetAllReferences returns all references for building call graphs.
	// Used by dead code analysis for efficient graph construction.
	GetAllReferences(limit int) ([]*types.Reference, error)
}

// Searcher handles search operations.
type Searcher interface {
	// Search performs hybrid search (BM25 + vector).
	Search(ctx context.Context, req *types.SearchRequest) ([]*types.SearchResult, error)
}

// MetadataStore handles index metadata.
type MetadataStore interface {
	// GetMetadata returns index metadata.
	GetMetadata() (*types.IndexMetadata, error)

	// SetMetadata stores index metadata.
	SetMetadata(meta *types.IndexMetadata) error

	// GetStats returns store statistics.
	GetStats() (*types.StoreStats, error)
}

// FileCache handles file hash caching for incremental indexing.
type FileCache interface {
	// GetFileHash returns the cached hash for a file.
	GetFileHash(filePath string) (string, error)

	// SetFileHash stores the hash for a file.
	SetFileHash(filePath, hash, configHash string) error

	// GetAllFileHashes returns all cached file hashes.
	GetAllFileHashes() (map[string]string, error)

	// DeleteFileCache removes file from cache.
	DeleteFileCache(filePath string) error
}

// Store is a minimal interface for basic store operations.
type Store interface {
	// Name returns the store name (e.g., "sqlitevec").
	Name() string

	// Init initializes the store at the given path.
	Init(path string) error

	// Close releases resources and closes connections.
	Close() error
}

// Maintainer provides maintenance operations for the store.
// Implementations should provide this interface for index optimization.
type Maintainer interface {
	// RebuildFTS rebuilds the FTS index from the source table.
	// This fixes corruption issues where FTS has references to deleted rows.
	RebuildFTS() error

	// CheckFTSHealth verifies that the FTS index is in sync with the source table.
	// Returns nil if healthy, error describing the issue otherwise.
	CheckFTSHealth() error
}

// GitHistoryStore handles git history storage and search operations.
type GitHistoryStore interface {
	// Commit operations
	StoreCommits(commits []*types.Commit) error
	GetCommit(hash string) (*types.Commit, error)
	GetCommitRange(fromHash, toHash string) ([]*types.Commit, error)
	GetCommitsByTimeRange(from, to *time.Time, limit int) ([]*types.Commit, error)
	GetLastIndexedCommit() (string, error)
	SetLastIndexedCommit(hash string) error

	// Change operations
	StoreChanges(changes []*types.Change) error
	GetChange(id string) (*types.Change, error)
	GetChangesByCommit(commitHash string) ([]*types.Change, error)
	GetChangesByFile(filePath string, limit int) ([]*types.Change, error)

	// Chunk history operations
	StoreChunkHistory(entries []*types.ChunkHistoryEntry) error
	GetChunkHistory(chunkID string, limit int) ([]*types.ChunkHistoryEntry, error)
	GetChunkEvolution(chunkID string) (*types.ChunkEvolution, error)

	// Search operations
	SearchHistory(ctx context.Context, req *types.HistorySearchRequest) ([]*types.HistorySearchResult, error)
	SearchCommitMessages(ctx context.Context, queryVec []float32, limit int) ([]*types.Commit, error)
	SearchDiffs(ctx context.Context, queryVec []float32, limit int) ([]*types.Change, error)

	// Analysis operations
	FindRegressionCandidates(ctx context.Context, description string, knownGood, knownBad string, limit int) ([]*types.RegressionCandidate, error)
	GetContributorInsights(ctx context.Context, paths []string, symbol string) ([]*types.ContributorInsight, error)

	// Statistics
	GetGitHistoryStats() (*types.GitHistoryStats, error)
}

// Ensure VectorStore embeds all sub-interfaces for backwards compatibility.
// This allows gradual migration while maintaining existing code.
var _ VectorStore = (*vectorStoreValidator)(nil)

type vectorStoreValidator struct{}

func (v *vectorStoreValidator) Name() string                                     { return "" }
func (v *vectorStoreValidator) Init(path string) error                           { return nil }
func (v *vectorStoreValidator) Close() error                                     { return nil }
func (v *vectorStoreValidator) StoreChunks(chunks []*types.ChunkWithEmbedding) error { return nil }
func (v *vectorStoreValidator) GetChunk(id string) (*types.Chunk, error)         { return nil, nil }
func (v *vectorStoreValidator) DeleteChunksByFile(filePath string) error         { return nil }
func (v *vectorStoreValidator) Search(ctx context.Context, req *types.SearchRequest) ([]*types.SearchResult, error) {
	return nil, nil
}
func (v *vectorStoreValidator) StoreSymbols(symbols []*types.Symbol) error           { return nil }
func (v *vectorStoreValidator) GetSymbol(id string) (*types.Symbol, error)           { return nil, nil }
func (v *vectorStoreValidator) FindSymbols(query string, kind types.SymbolKind, limit int) ([]*types.Symbol, error) {
	return nil, nil
}
func (v *vectorStoreValidator) FindSymbolsAdvanced(query string, kind types.SymbolKind, minLines int, sortBy string, limit int) ([]*types.Symbol, error) {
	return nil, nil
}
func (v *vectorStoreValidator) FindLongFunctions(minLines int, limit int) ([]*types.Symbol, error) {
	return nil, nil
}
func (v *vectorStoreValidator) StoreReferences(refs []*types.Reference) error { return nil }
func (v *vectorStoreValidator) GetCallers(symbolID string, limit int) ([]*types.Reference, error) {
	return nil, nil
}
func (v *vectorStoreValidator) GetCallees(symbolID string, limit int) ([]*types.Reference, error) {
	return nil, nil
}
func (v *vectorStoreValidator) FindReferencesByKind(kind types.RefKind, limit int) ([]*types.Reference, error) {
	return nil, nil
}
func (v *vectorStoreValidator) GetAllReferences(limit int) ([]*types.Reference, error) {
	return nil, nil
}
func (v *vectorStoreValidator) GetMetadata() (*types.IndexMetadata, error)       { return nil, nil }
func (v *vectorStoreValidator) SetMetadata(meta *types.IndexMetadata) error      { return nil }
func (v *vectorStoreValidator) GetStats() (*types.StoreStats, error)             { return nil, nil }
func (v *vectorStoreValidator) GetFileHash(filePath string) (string, error)      { return "", nil }
func (v *vectorStoreValidator) SetFileHash(filePath, hash, configHash string) error { return nil }
func (v *vectorStoreValidator) GetAllFileHashes() (map[string]string, error)     { return nil, nil }
func (v *vectorStoreValidator) DeleteFileCache(filePath string) error            { return nil }

// =============================================================================
// Memory Store - Persistent Context and Knowledge
// =============================================================================

// MemoryStore handles memory/knowledge storage and search operations.
type MemoryStore interface {
	// Memory CRUD operations
	StoreMemory(memory *types.MemoryEntry) error
	GetMemory(id string) (*types.MemoryEntry, error)
	UpdateMemory(memory *types.MemoryEntry) error
	DeleteMemory(id string) error

	// Batch operations
	StoreMemories(memories []*types.MemoryEntry) error
	DeleteMemoriesByChannel(channel string) error

	// Search operations
	SearchMemories(ctx context.Context, req *types.MemorySearchRequest) ([]*types.MemorySearchResult, error)
	RecallMemories(ctx context.Context, query string, channel string, limit int) ([]*types.MemoryEntry, error)

	// Session operations
	CreateSession(session *types.MemorySession) error
	GetSession(id string) (*types.MemorySession, error)
	EndSession(id string) error
	GetActiveSessions(channel string) ([]*types.MemorySession, error)

	// Checkpoint operations
	CreateCheckpoint(checkpoint *types.MemoryCheckpoint) error
	GetCheckpoint(id string) (*types.MemoryCheckpoint, error)
	RestoreCheckpoint(id string) error
	ListCheckpoints(channel string, limit int) ([]*types.MemoryCheckpoint, error)

	// Maintenance
	PruneExpired() (int, error)
	Consolidate(channel string) error // Merge similar memories
	DecayConfidence(halfLifeDays int) error // Apply time-based confidence decay

	// Statistics
	GetMemoryStats(channel string) (*types.MemoryStats, error)
}

// =============================================================================
// Todo Store - Task Management
// =============================================================================

// TodoStore handles todo/task storage and search operations.
type TodoStore interface {
	// Todo CRUD operations
	StoreTodo(todo *types.TodoItem) error
	GetTodo(id string) (*types.TodoItem, error)
	UpdateTodo(todo *types.TodoItem) error
	DeleteTodo(id string) error

	// Status operations
	UpdateTodoStatus(id string, status types.TodoStatus) error
	CompleteTodo(id string) error
	StartTodo(id string) error

	// Batch operations
	StoreTodos(todos []*types.TodoItem) error
	DeleteTodosByChannel(channel string) error

	// Hierarchy operations
	GetTodoChildren(parentID string) ([]*types.TodoItem, error)
	MoveTodo(id string, newParentID string) error

	// Search operations
	SearchTodos(ctx context.Context, req *types.TodoSearchRequest) ([]*types.TodoSearchResult, error)
	ListTodos(ctx context.Context, channel string, status []types.TodoStatus, limit int) ([]*types.TodoItem, error)

	// Query helpers
	GetOverdueTodos(channel string) ([]*types.TodoItem, error)
	GetTodosByPriority(channel string, priority types.TodoPriority, limit int) ([]*types.TodoItem, error)
	GetBlockedTodos(channel string) ([]*types.TodoItem, error)

	// Statistics
	GetTodoStats(channel string) (*types.TodoStats, error)
}
