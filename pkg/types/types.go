// Package types contains shared data types used across the codeindex project.
package types

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// SourceFile represents a source code file to be indexed.
type SourceFile struct {
	Path     string // Absolute path to the file
	Content  []byte // File content
	Language string // Detected language (go, python, javascript, etc.)
	Hash     string // SHA256 hash for incremental indexing
}

// ComputeHash calculates SHA256 hash of the file content.
func (f *SourceFile) ComputeHash() string {
	h := sha256.Sum256(f.Content)
	return hex.EncodeToString(h[:])
}

// ChunkType represents the type of code chunk.
type ChunkType string

const (
	ChunkTypeFunction ChunkType = "function"
	ChunkTypeClass    ChunkType = "class"
	ChunkTypeMethod   ChunkType = "method"
	ChunkTypeBlock    ChunkType = "block"
	ChunkTypeFile     ChunkType = "file"
)

// Chunk represents a piece of code that will be embedded.
type Chunk struct {
	ID         string    // Unique ID: {filepath}:{startline}:{hash[:8]}
	FilePath   string    // Path to source file
	Language   string    // Programming language
	Content    string    // Chunk content
	ChunkType  ChunkType // Type of chunk
	Name       string    // Name of function/class/method
	ParentName string    // Parent scope name
	StartLine  int       // Starting line number (1-based)
	EndLine    int       // Ending line number (1-based)
	Hash       string    // SHA256 of content
}

// GenerateID creates a unique ID for the chunk.
func (c *Chunk) GenerateID() string {
	h := sha256.Sum256([]byte(c.Content))
	hashPrefix := hex.EncodeToString(h[:4])
	return c.FilePath + ":" + strconv.Itoa(c.StartLine) + ":" + hashPrefix
}

// ChunkWithEmbedding is a Chunk with its vector embedding.
type ChunkWithEmbedding struct {
	Chunk     *Chunk
	Embedding []float32
}

// SymbolKind represents the type of symbol.
type SymbolKind string

const (
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindType      SymbolKind = "type"
	SymbolKindVariable  SymbolKind = "variable"
	SymbolKindConstant  SymbolKind = "constant"
	SymbolKindInterface SymbolKind = "interface"
	SymbolKindMethod    SymbolKind = "method"
)

// Symbol represents a code symbol (function, type, variable, etc.).
type Symbol struct {
	ID         string     // Unique identifier
	Name       string     // Symbol name
	Kind       SymbolKind // Type of symbol
	FilePath   string     // File where symbol is defined
	StartLine  int        // Starting line
	EndLine    int        // Ending line
	LineCount  int        // Number of lines (EndLine - StartLine + 1)
	Signature  string     // For functions: func(x int) error
	Visibility string     // public, private
	DocComment string     // Documentation comment
}

// ComputeLineCount calculates the line count from StartLine and EndLine.
func (s *Symbol) ComputeLineCount() int {
	if s.EndLine >= s.StartLine {
		return s.EndLine - s.StartLine + 1
	}
	return 1
}

// RefKind represents the type of reference.
type RefKind string

const (
	RefKindCall      RefKind = "call"
	RefKindTypeUse   RefKind = "type_use"
	RefKindImport    RefKind = "import"
	RefKindImplement RefKind = "implement"
)

// Reference represents a reference from one symbol to another.
type Reference struct {
	ID         string  // Unique identifier
	FromSymbol string  // Symbol ID that references
	ToSymbol   string  // Symbol ID being referenced
	Kind       RefKind // Type of reference
	FilePath   string  // File where reference occurs
	Line       int     // Line number
	IsExternal bool    // True if ToSymbol is not in our index (e.g., fmt.Println)
}

// CodePattern represents a detected design pattern.
type CodePattern struct {
	ID         string   // Unique identifier
	Name       string   // Pattern name: repository, middleware, singleton, factory
	FilePath   string   // File where pattern is detected
	StartLine  int      // Starting line
	EndLine    int      // Ending line
	Confidence float32  // Confidence score (0-1)
	Evidence   []string // Why we think this is the pattern
}

// SearchMode represents the type of search to perform.
type SearchMode string

const (
	SearchModeVector SearchMode = "vector"
	SearchModeBM25   SearchMode = "bm25"
	SearchModeHybrid SearchMode = "hybrid"
)

// SearchFilters contains filters for search queries.
type SearchFilters struct {
	Languages  []string    // Filter by language
	ChunkTypes []ChunkType // Filter by chunk type
	FilePaths  []string    // Glob patterns for file paths
}

// SearchRequest represents a search query.
type SearchRequest struct {
	Query    string    // Text query (for BM25 and reranking)
	QueryVec []float32 // Query embedding (for vector search)
	Limit    int       // Max results to return
	Filters  *SearchFilters
	Mode     SearchMode // vector, bm25, hybrid

	// Hybrid search weights
	VectorWeight float32 // Default 0.7
	BM25Weight   float32 // Default 0.3

	// Reranking
	UseReranker      bool // Default true if reranker is configured
	RerankCandidates int  // How many candidates for reranking (default 100)

	// Context
	IncludeContext bool // Include surrounding lines
	ContextLines   int  // Default 5
}

// SearchResult represents a single search result.
type SearchResult struct {
	Chunk         *Chunk
	Score         float32 // Final score (after reranking if enabled)
	VectorScore   float32 // For debugging
	BM25Score     float32 // For debugging
	RerankScore   float32 // For debugging (0 if reranking disabled)
	ContextBefore string  // Lines before chunk (if IncludeContext=true)
	ContextAfter  string  // Lines after chunk
}

// StoreStats contains statistics about the index.
type StoreStats struct {
	TotalChunks     int
	TotalSymbols    int
	TotalReferences int
	IndexedFiles    int
	LastIndexed     time.Time
	DBSizeBytes     int64
}

// IndexMetadata contains metadata about the index.
type IndexMetadata struct {
	SchemaVersion int       // For detecting incompatible changes
	CreatedAt     time.Time // When index was created
	LastUpdated   time.Time // Last update time
	ToolVersion   string    // Version of mcp-codewizard
	ConfigHash    string    // Hash of configuration

	// Provider info
	EmbeddingProvider   string // "ollama"
	EmbeddingModel      string // "nomic-embed-code"
	EmbeddingDimensions int    // 768
	ChunkingStrategy    string // "treesitter"
	RerankerModel       string // "qwen3-reranker" or ""

	// Stats
	Stats StoreStats
}

// IndexProgress represents the current state of indexing.
type IndexProgress struct {
	Phase           string // "scanning", "chunking", "embedding", "storing"
	TotalFiles      int
	ProcessedFiles  int
	TotalChunks     int
	ProcessedChunks int
	CurrentFile     string
	Error           error // Non-fatal error (e.g., cannot parse file)
}

// ProjectComplexity represents the size/complexity of a project.
type ProjectComplexity string

const (
	ComplexitySmall  ProjectComplexity = "small"  // < 100 files
	ComplexityMedium ProjectComplexity = "medium" // 100-1000 files
	ComplexityLarge  ProjectComplexity = "large"  // 1000-10000 files
	ComplexityHuge   ProjectComplexity = "huge"   // > 10000 files
)

// DetermineComplexity returns the complexity based on file count.
func DetermineComplexity(fileCount int) ProjectComplexity {
	switch {
	case fileCount < 100:
		return ComplexitySmall
	case fileCount < 1000:
		return ComplexityMedium
	case fileCount < 10000:
		return ComplexityLarge
	default:
		return ComplexityHuge
	}
}

// =============================================================================
// Git History Types - 3D Search (Code + Semantics + Time)
// =============================================================================

// ChangeType represents the type of change in a commit.
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "A" // File added
	ChangeTypeModified ChangeType = "M" // File modified
	ChangeTypeDeleted  ChangeType = "D" // File deleted
	ChangeTypeRenamed  ChangeType = "R" // File renamed
	ChangeTypeCopied   ChangeType = "C" // File copied
)

// Commit represents a git commit with metadata and embeddings.
type Commit struct {
	Hash             string    `json:"hash"`
	ShortHash        string    `json:"short_hash"` // First 8 characters
	Author           string    `json:"author"`
	AuthorEmail      string    `json:"author_email"`
	Date             time.Time `json:"date"`
	Message          string    `json:"message"`
	MessageEmbedding []float32 `json:"-"` // Embedding of commit message
	ParentHash       string    `json:"parent_hash"`

	// Statistics
	FilesChanged int `json:"files_changed"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`

	// Metadata
	IsMerge     bool     `json:"is_merge"`
	IsTagged    bool     `json:"is_tagged"`
	Tags        []string `json:"tags,omitempty"`
	Branch      string   `json:"branch,omitempty"`
}

// Change represents a single file change within a commit.
type Change struct {
	ID               string     `json:"id"` // {commit_hash[:8]}:{file_path}
	CommitHash       string     `json:"commit_hash"`
	FilePath         string     `json:"file_path"`
	ChangeType       ChangeType `json:"change_type"`
	OldPath          string     `json:"old_path,omitempty"` // For renames

	// Diff content
	DiffContent      string    `json:"diff_content,omitempty"`
	DiffEmbedding    []float32 `json:"-"` // Embedding of diff (for significant changes)
	Additions        int       `json:"additions"`
	Deletions        int       `json:"deletions"`

	// Semantic linking - which functions/chunks were affected
	AffectedFunctions []string `json:"affected_functions,omitempty"`
	AffectedChunkIDs  []string `json:"affected_chunk_ids,omitempty"`

	// Hunks - individual change blocks within the file
	Hunks []DiffHunk `json:"hunks,omitempty"`
}

// DiffHunk represents a single hunk in a diff.
type DiffHunk struct {
	OldStart  int    `json:"old_start"`
	OldLines  int    `json:"old_lines"`
	NewStart  int    `json:"new_start"`
	NewLines  int    `json:"new_lines"`
	Content   string `json:"content"`
	Context   string `json:"context,omitempty"` // Function/class name if detected
}

// ChunkHistoryEntry links a chunk to its commit history.
type ChunkHistoryEntry struct {
	ChunkID     string     `json:"chunk_id"`
	CommitHash  string     `json:"commit_hash"`
	ChangeType  string     `json:"change_type"` // "created", "modified", "deleted"
	DiffSummary string     `json:"diff_summary,omitempty"`
	Date        time.Time  `json:"date"`
	Author      string     `json:"author"`
}

// HistorySearchRequest represents a search query across git history.
type HistorySearchRequest struct {
	// Semantic query
	Query    string    `json:"query"`
	QueryVec []float32 `json:"-"`

	// Time filters
	TimeFrom *time.Time `json:"time_from,omitempty"`
	TimeTo   *time.Time `json:"time_to,omitempty"`

	// Commit filters
	CommitFrom string   `json:"commit_from,omitempty"` // Start commit hash
	CommitTo   string   `json:"commit_to,omitempty"`   // End commit hash (default: HEAD)
	Authors    []string `json:"authors,omitempty"`     // Filter by author email

	// File filters
	Paths       []string     `json:"paths,omitempty"`        // Glob patterns
	ChangeTypes []ChangeType `json:"change_types,omitempty"` // A, M, D, R

	// Function/symbol filters
	Functions []string `json:"functions,omitempty"` // Affected function names

	// Search options
	Limit            int     `json:"limit"`
	IncludeContext   bool    `json:"include_context"`   // Include before/after code
	IncludeDiff      bool    `json:"include_diff"`      // Include diff content
	UseReranker      bool    `json:"use_reranker"`
	RerankCandidates int     `json:"rerank_candidates"`

	// Scoring options
	TimeDecayFactor float32 `json:"time_decay_factor"` // 0 = no decay, 1 = strong decay
}

// HistorySearchResult represents a single result from history search.
type HistorySearchResult struct {
	Change        *Change  `json:"change"`
	Commit        *Commit  `json:"commit"`
	Score         float32  `json:"score"`
	MessageScore  float32  `json:"message_score,omitempty"`  // Commit message similarity
	DiffScore     float32  `json:"diff_score,omitempty"`     // Diff content similarity
	RecencyScore  float32  `json:"recency_score,omitempty"`  // Time-based score

	// Context
	BeforeCode    string   `json:"before_code,omitempty"`    // Code before the change
	AfterCode     string   `json:"after_code,omitempty"`     // Code after the change
	RelatedChanges []*Change `json:"related_changes,omitempty"` // Other changes in same commit
}

// ChunkEvolution represents how a chunk evolved over time.
type ChunkEvolution struct {
	ChunkID      string               `json:"chunk_id"`
	CurrentName  string               `json:"current_name"`
	FilePath     string               `json:"file_path"`
	CreatedAt    time.Time            `json:"created_at"`
	CreatedBy    string               `json:"created_by"`
	LastModified time.Time            `json:"last_modified"`
	ModifyCount  int                  `json:"modify_count"`
	Authors      map[string]int       `json:"authors"`      // Author -> change count
	History      []*ChunkHistoryEntry `json:"history"`
}

// RegressionCandidate represents a potential regression-introducing commit.
type RegressionCandidate struct {
	Commit      *Commit   `json:"commit"`
	Changes     []*Change `json:"changes"`
	Score       float32   `json:"score"`       // How likely this introduced the bug
	Reasoning   []string  `json:"reasoning"`   // Why this is a candidate
	DiffPreview string    `json:"diff_preview,omitempty"`
}

// ContributorInsight represents contributor knowledge about code areas.
type ContributorInsight struct {
	Author       string   `json:"author"`
	Email        string   `json:"email"`
	CommitCount  int      `json:"commit_count"`
	LinesChanged int      `json:"lines_changed"`
	LastActive   time.Time `json:"last_active"`
	TopFunctions []string `json:"top_functions"` // Most frequently changed functions
	TopFiles     []string `json:"top_files"`     // Most frequently changed files
	Expertise    float32  `json:"expertise"`     // 0-1 expertise score for the queried area
}

// GitIndexConfig controls how git history is indexed.
type GitIndexConfig struct {
	// Full indexing for recent commits
	FullHistoryDays int `json:"full_history_days"` // Default: 30

	// Sampled indexing for older commits
	SampledHistoryDays int     `json:"sampled_history_days"` // Default: 365
	SampleRate         float32 `json:"sample_rate"`          // 0.2 = every 5th commit

	// Beyond sampled period - only tags/releases
	OnlyTags bool `json:"only_tags"`

	// Embedding options
	EmbedCommitMessages bool `json:"embed_commit_messages"` // Default: true
	EmbedDiffs          bool `json:"embed_diffs"`           // Default: true for changes > MinDiffLines
	MinDiffLines        int  `json:"min_diff_lines"`        // Min lines to embed diff (default: 10)
	MaxDiffLines        int  `json:"max_diff_lines"`        // Max lines to embed diff (default: 500)

	// Limits
	MaxCommits int `json:"max_commits"` // Absolute limit
}

// GitIndexProgress represents progress of git history indexing.
type GitIndexProgress struct {
	Phase           string `json:"phase"` // "scanning", "parsing", "embedding", "storing"
	TotalCommits    int    `json:"total_commits"`
	ProcessedCommits int   `json:"processed_commits"`
	TotalChanges    int    `json:"total_changes"`
	ProcessedChanges int   `json:"processed_changes"`
	CurrentCommit   string `json:"current_commit"`
	Error           error  `json:"-"`
}

// GitHistoryStats contains statistics about indexed git history.
type GitHistoryStats struct {
	TotalCommits          int       `json:"total_commits"`
	TotalChanges          int       `json:"total_changes"`
	CommitsWithEmbeddings int       `json:"commits_with_embeddings"`
	ChangesWithEmbeddings int       `json:"changes_with_embeddings"`
	OldestCommit          time.Time `json:"oldest_commit"`
	NewestCommit          time.Time `json:"newest_commit"`
	UniqueAuthors         int       `json:"unique_authors"`
	LastIndexed           time.Time `json:"last_indexed"`
}

// =============================================================================
// Memory Types - Persistent Context and Knowledge
// =============================================================================

// MemoryCategory represents the type of memory entry.
type MemoryCategory string

const (
	MemoryCategoryDecision MemoryCategory = "decision" // Architectural/design decisions
	MemoryCategoryContext  MemoryCategory = "context"  // Project context, conventions
	MemoryCategoryFact     MemoryCategory = "fact"     // Facts about the codebase
	MemoryCategoryNote     MemoryCategory = "note"     // General notes
	MemoryCategoryError    MemoryCategory = "error"    // Resolved errors, debugging insights
	MemoryCategoryReview   MemoryCategory = "review"   // Code review feedback
)

// MemoryEntry represents a persistent memory/knowledge entry.
type MemoryEntry struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`   // The actual memory content
	Summary   string         `json:"summary"`   // Short summary (max 100 chars)
	Embedding []float32      `json:"-"`         // Vector embedding for semantic search
	Category  MemoryCategory `json:"category"`
	Tags      []string       `json:"tags,omitempty"`

	// Importance & Scoring
	Importance  float32 `json:"importance"`   // 0-1, can be auto or manual
	Confidence  float32 `json:"confidence"`   // 0-1, decays over time
	AccessCount int     `json:"access_count"` // How many times recalled

	// Context isolation (branch-aware)
	Channel string `json:"channel"` // Git branch or custom channel (default: "main")

	// Relationships
	RelatedMemories []string `json:"related_memories,omitempty"` // IDs of related memories
	RelatedChunks   []string `json:"related_chunks,omitempty"`   // IDs of related code chunks
	RelatedCommits  []string `json:"related_commits,omitempty"`  // Hashes of related commits

	// Timestamps
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	AccessedAt time.Time  `json:"accessed_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"` // Optional TTL

	// Session tracking
	SessionID string `json:"session_id,omitempty"` // Session that created this
}

// MemorySearchRequest represents a search query for memories.
type MemorySearchRequest struct {
	Query    string    `json:"query"`
	QueryVec []float32 `json:"-"`

	// Filters
	Categories []MemoryCategory `json:"categories,omitempty"`
	Tags       []string         `json:"tags,omitempty"`
	Channel    string           `json:"channel,omitempty"` // Empty = all channels
	MinImportance float32       `json:"min_importance,omitempty"`

	// Time filters
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	AccessedAfter *time.Time `json:"accessed_after,omitempty"`

	// Options
	Limit            int     `json:"limit"`
	IncludeExpired   bool    `json:"include_expired"`
	UseReranker      bool    `json:"use_reranker"`
	RerankCandidates int     `json:"rerank_candidates"`

	// Scoring weights for hybrid search
	VectorWeight    float32 `json:"vector_weight"`    // Default 0.5
	RecencyWeight   float32 `json:"recency_weight"`   // Default 0.2
	ImportanceWeight float32 `json:"importance_weight"` // Default 0.2
	FrequencyWeight float32 `json:"frequency_weight"` // Default 0.1
}

// MemorySearchResult represents a memory search result.
type MemorySearchResult struct {
	Memory          *MemoryEntry `json:"memory"`
	Score           float32      `json:"score"`       // Final combined score
	VectorScore     float32      `json:"vector_score"`
	RecencyScore    float32      `json:"recency_score"`
	ImportanceScore float32      `json:"importance_score"`
	FrequencyScore  float32      `json:"frequency_score"`
}

// MemorySession represents a logical session for memory grouping.
type MemorySession struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Channel     string    `json:"channel"` // Git branch
	StartedAt   time.Time `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	MemoryCount int       `json:"memory_count"`
}

// MemoryCheckpoint represents a snapshot of memory state.
type MemoryCheckpoint struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	Channel     string    `json:"channel"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	MemoryIDs   []string  `json:"memory_ids"` // IDs of memories in this checkpoint
}

// MemoryStats contains statistics about memories.
type MemoryStats struct {
	TotalMemories    int            `json:"total_memories"`
	MemoriesByCategory map[MemoryCategory]int `json:"memories_by_category"`
	MemoriesByChannel map[string]int `json:"memories_by_channel"`
	ActiveSessions   int            `json:"active_sessions"`
	TotalCheckpoints int            `json:"total_checkpoints"`
	OldestMemory     *time.Time     `json:"oldest_memory,omitempty"`
	NewestMemory     *time.Time     `json:"newest_memory,omitempty"`
	ExpiredCount     int            `json:"expired_count"`
}

// =============================================================================
// Todo/Task Types - Task Management with Smart Search
// =============================================================================

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusCancelled  TodoStatus = "cancelled"
	TodoStatusBlocked    TodoStatus = "blocked"
)

// TodoPriority represents the priority of a todo item.
type TodoPriority string

const (
	TodoPriorityLow    TodoPriority = "low"
	TodoPriorityMedium TodoPriority = "medium"
	TodoPriorityHigh   TodoPriority = "high"
	TodoPriorityUrgent TodoPriority = "urgent"
)

// TodoItem represents a task or todo item.
type TodoItem struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Embedding   []float32    `json:"-"` // Embedding of title + description
	Status      TodoStatus   `json:"status"`
	Priority    TodoPriority `json:"priority"`
	Tags        []string     `json:"tags,omitempty"`

	// Hierarchy
	ParentID    string   `json:"parent_id,omitempty"`    // For subtasks
	ChildrenIDs []string `json:"children_ids,omitempty"` // Subtask IDs

	// Context isolation (branch-aware)
	Channel string `json:"channel"` // Git branch or custom channel

	// Relationships
	RelatedMemories []string `json:"related_memories,omitempty"` // Related memory IDs
	RelatedChunks   []string `json:"related_chunks,omitempty"`   // Related code chunk IDs
	RelatedCommits  []string `json:"related_commits,omitempty"`  // Related commit hashes
	BlockedBy       []string `json:"blocked_by,omitempty"`       // IDs of blocking todos

	// Progress tracking
	Progress    int    `json:"progress"`     // 0-100 percentage
	Effort      string `json:"effort,omitempty"` // e.g., "2h", "1d", "S/M/L"
	ActualEffort string `json:"actual_effort,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Session tracking
	SessionID string `json:"session_id,omitempty"`
}

// TodoSearchRequest represents a search query for todos.
type TodoSearchRequest struct {
	Query    string    `json:"query"`
	QueryVec []float32 `json:"-"`

	// Filters
	Statuses   []TodoStatus   `json:"statuses,omitempty"`
	Priorities []TodoPriority `json:"priorities,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Channel    string         `json:"channel,omitempty"`
	ParentID   string         `json:"parent_id,omitempty"` // Empty = top-level only

	// Time filters
	DueBefore    *time.Time `json:"due_before,omitempty"`
	DueAfter     *time.Time `json:"due_after,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`

	// Options
	Limit            int  `json:"limit"`
	IncludeCompleted bool `json:"include_completed"`
	IncludeChildren  bool `json:"include_children"` // Include subtasks
	UseReranker      bool `json:"use_reranker"`
}

// TodoSearchResult represents a todo search result.
type TodoSearchResult struct {
	Todo        *TodoItem `json:"todo"`
	Score       float32   `json:"score"`
	VectorScore float32   `json:"vector_score"`
	Children    []*TodoItem `json:"children,omitempty"` // If include_children
}

// TodoStats contains statistics about todos.
type TodoStats struct {
	TotalTodos       int                   `json:"total_todos"`
	TodosByStatus    map[TodoStatus]int    `json:"todos_by_status"`
	TodosByPriority  map[TodoPriority]int  `json:"todos_by_priority"`
	TodosByChannel   map[string]int        `json:"todos_by_channel"`
	OverdueTodos     int                   `json:"overdue_todos"`
	CompletedToday   int                   `json:"completed_today"`
	CompletedThisWeek int                  `json:"completed_this_week"`
}
