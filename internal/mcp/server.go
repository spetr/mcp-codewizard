// Package mcp implements the MCP server for code search.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/spetr/mcp-codewizard/internal/analysis"
	"github.com/spetr/mcp-codewizard/internal/config"
	"github.com/spetr/mcp-codewizard/internal/index"
	"github.com/spetr/mcp-codewizard/internal/search"
	"github.com/spetr/mcp-codewizard/internal/wizard"
	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// Server implements the MCP server.
type Server struct {
	mcpServer  *server.MCPServer
	projectDir string
	config     *config.Config
	store      provider.VectorStore
	embedding  provider.EmbeddingProvider
	chunker    provider.ChunkingStrategy
	reranker   provider.Reranker
	search     *search.Engine
}

// Config contains server configuration.
type Config struct {
	ProjectDir string
	Config     *config.Config
	Store      provider.VectorStore
	Embedding  provider.EmbeddingProvider
	Chunker    provider.ChunkingStrategy
	Reranker   provider.Reranker
}

// New creates a new MCP server.
func New(cfg Config) (*Server, error) {
	s := &Server{
		projectDir: cfg.ProjectDir,
		config:     cfg.Config,
		store:      cfg.Store,
		embedding:  cfg.Embedding,
		chunker:    cfg.Chunker,
		reranker:   cfg.Reranker,
	}

	// Create search engine
	s.search = search.New(search.Config{
		Store:     cfg.Store,
		Embedding: cfg.Embedding,
		Reranker:  cfg.Reranker,
	})

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-codewizard",
		"0.1.0",
		server.WithLogging(),
	)

	// Register tools
	s.registerTools(mcpServer)

	s.mcpServer = mcpServer
	return s, nil
}

// registerTools registers all MCP tools.
func (s *Server) registerTools(mcpServer *server.MCPServer) {
	// index_codebase - Index the current project
	mcpServer.AddTool(mcp.NewTool("index_codebase",
		mcp.WithDescription("Index the codebase for semantic search"),
		mcp.WithBoolean("force", mcp.Description("Force reindex all files")),
	), s.handleIndexCodebase)

	// search_code - Semantic code search
	mcpServer.AddTool(mcp.NewTool("search_code",
		mcp.WithDescription("Search code using semantic similarity"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 10)")),
		mcp.WithString("mode", mcp.Description("Search mode: vector, bm25, hybrid (default)")),
		mcp.WithBoolean("no_rerank", mcp.Description("Disable reranking")),
		mcp.WithBoolean("include_context", mcp.Description("Include surrounding lines")),
		mcp.WithArray("languages", mcp.Description("Filter by languages")),
	), s.handleSearchCode)

	// get_chunk - Get a specific chunk with context
	mcpServer.AddTool(mcp.NewTool("get_chunk",
		mcp.WithDescription("Get a specific code chunk by ID with context"),
		mcp.WithString("chunk_id", mcp.Required(), mcp.Description("Chunk ID")),
		mcp.WithNumber("context_lines", mcp.Description("Lines of context (default 5)")),
	), s.handleGetChunk)

	// get_status - Get index status
	mcpServer.AddTool(mcp.NewTool("get_status",
		mcp.WithDescription("Get index status and statistics"),
	), s.handleGetStatus)

	// clear_index - Clear the index
	mcpServer.AddTool(mcp.NewTool("clear_index",
		mcp.WithDescription("Clear the search index"),
	), s.handleClearIndex)

	// get_callers - Get callers of a symbol
	mcpServer.AddTool(mcp.NewTool("get_callers",
		mcp.WithDescription("Get functions that call a specific symbol"),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Symbol name or ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum results")),
	), s.handleGetCallers)

	// get_callees - Get what a symbol calls
	mcpServer.AddTool(mcp.NewTool("get_callees",
		mcp.WithDescription("Get functions called by a specific symbol"),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Symbol name or ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum results")),
	), s.handleGetCallees)

	// get_symbols - List symbols
	mcpServer.AddTool(mcp.NewTool("get_symbols",
		mcp.WithDescription("Search for symbols by name"),
		mcp.WithString("query", mcp.Description("Symbol name pattern")),
		mcp.WithString("kind", mcp.Description("Symbol kind: function, type, variable, constant, interface, method")),
		mcp.WithNumber("limit", mcp.Description("Maximum results")),
	), s.handleGetSymbols)

	// get_entry_points - Find entry points
	mcpServer.AddTool(mcp.NewTool("get_entry_points",
		mcp.WithDescription("Get main entry points (main functions, HTTP handlers, tests, init functions)"),
		mcp.WithNumber("limit", mcp.Description("Maximum results")),
		mcp.WithString("type", mcp.Description("Filter by type: main, handler, test, init, cli")),
	), s.handleGetEntryPoints)

	// get_import_graph - Get import relationships
	mcpServer.AddTool(mcp.NewTool("get_import_graph",
		mcp.WithDescription("Get import/dependency graph between modules"),
		mcp.WithNumber("limit", mcp.Description("Maximum import edges to return")),
	), s.handleGetImportGraph)

	// Setup Wizard tools

	// detect_environment - Detect available providers and project info
	mcpServer.AddTool(mcp.NewTool("detect_environment",
		mcp.WithDescription("Detect available embedding providers, project structure, and get configuration recommendations"),
	), s.handleDetectEnvironment)

	// validate_config - Validate current configuration
	mcpServer.AddTool(mcp.NewTool("validate_config",
		mcp.WithDescription("Validate the current configuration and test provider connections"),
	), s.handleValidateConfig)

	// get_config - Get current configuration
	mcpServer.AddTool(mcp.NewTool("get_config",
		mcp.WithDescription("Get the current configuration settings"),
	), s.handleGetConfig)

	// apply_recommendation - Apply a configuration recommendation
	mcpServer.AddTool(mcp.NewTool("apply_recommendation",
		mcp.WithDescription("Apply a configuration recommendation (primary, low_memory, high_quality, fast_index)"),
		mcp.WithString("preset", mcp.Required(), mcp.Description("Preset to apply: primary, low_memory, high_quality, fast_index")),
	), s.handleApplyRecommendation)

	// Analysis tools

	// get_blame - Get git blame information
	mcpServer.AddTool(mcp.NewTool("get_blame",
		mcp.WithDescription("Get git blame information for a file or line range"),
		mcp.WithString("file", mcp.Required(), mcp.Description("File path (relative to project root)")),
		mcp.WithNumber("start_line", mcp.Description("Start line (optional, for range)")),
		mcp.WithNumber("end_line", mcp.Description("End line (optional, for range)")),
	), s.handleGetBlame)

	// get_dead_code - Find potentially unused code
	mcpServer.AddTool(mcp.NewTool("get_dead_code",
		mcp.WithDescription("Find potentially unused/dead code in the codebase"),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 20)")),
		mcp.WithString("type", mcp.Description("Type to search: all, functions, types (default: functions)")),
	), s.handleGetDeadCode)

	// get_complexity - Get code complexity metrics
	mcpServer.AddTool(mcp.NewTool("get_complexity",
		mcp.WithDescription("Get code complexity metrics for a file or function"),
		mcp.WithString("file", mcp.Required(), mcp.Description("File path (relative to project root)")),
		mcp.WithNumber("start_line", mcp.Description("Start line (optional, for range)")),
		mcp.WithNumber("end_line", mcp.Description("End line (optional, for range)")),
	), s.handleGetComplexity)

	// Git History tools - 3D temporal search

	// search_history - Semantic search across git history
	mcpServer.AddTool(mcp.NewTool("search_history",
		mcp.WithDescription("Semantic search across git history - find changes by description, time range, author, or affected code"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query (e.g., 'authentication bug fix', 'memory leak')")),
		mcp.WithString("time_from", mcp.Description("Start date (ISO format: 2024-01-01) or commit hash")),
		mcp.WithString("time_to", mcp.Description("End date (ISO format) or commit hash (default: now)")),
		mcp.WithArray("authors", mcp.Description("Filter by author emails")),
		mcp.WithArray("paths", mcp.Description("Filter by file paths (glob patterns, e.g., 'src/auth/*')")),
		mcp.WithArray("change_types", mcp.Description("Filter by change type: A (added), M (modified), D (deleted), R (renamed)")),
		mcp.WithArray("functions", mcp.Description("Filter by affected function names")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 10)")),
		mcp.WithBoolean("include_diff", mcp.Description("Include diff content in results")),
	), s.handleSearchHistory)

	// get_chunk_history - History of a specific chunk/function
	mcpServer.AddTool(mcp.NewTool("get_chunk_history",
		mcp.WithDescription("Get the change history of a specific code chunk or function"),
		mcp.WithString("chunk_id", mcp.Description("Chunk ID to get history for")),
		mcp.WithString("symbol", mcp.Description("Symbol/function name to get history for")),
		mcp.WithString("file", mcp.Description("File path (used with symbol)")),
		mcp.WithNumber("limit", mcp.Description("Maximum history entries (default 20)")),
	), s.handleGetChunkHistory)

	// get_code_evolution - How code evolved over time
	mcpServer.AddTool(mcp.NewTool("get_code_evolution",
		mcp.WithDescription("Get detailed evolution of a symbol/function over time"),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Symbol/function name")),
		mcp.WithString("file", mcp.Description("File path to narrow down search")),
	), s.handleGetCodeEvolution)

	// find_regression - Find potential regression-introducing commits
	mcpServer.AddTool(mcp.NewTool("find_regression",
		mcp.WithDescription("Find commits that might have introduced a bug or regression"),
		mcp.WithString("description", mcp.Required(), mcp.Description("Description of the bug/problem")),
		mcp.WithString("known_good", mcp.Description("Commit hash/tag where it worked (optional)")),
		mcp.WithString("known_bad", mcp.Description("Commit hash/tag where it's broken (default: HEAD)")),
		mcp.WithArray("paths", mcp.Description("Limit search to specific paths")),
		mcp.WithNumber("limit", mcp.Description("Maximum candidates to return (default 10)")),
	), s.handleFindRegression)

	// get_commit_context - Detailed view of a commit
	mcpServer.AddTool(mcp.NewTool("get_commit_context",
		mcp.WithDescription("Get detailed information about a commit and all its changes"),
		mcp.WithString("commit", mcp.Required(), mcp.Description("Commit hash (full or short)")),
		mcp.WithBoolean("include_diff", mcp.Description("Include diff content")),
	), s.handleGetCommitContext)

	// get_contributor_insights - Who knows this code
	mcpServer.AddTool(mcp.NewTool("get_contributor_insights",
		mcp.WithDescription("Get contributor expertise and activity for code areas"),
		mcp.WithArray("paths", mcp.Description("File paths/patterns to analyze")),
		mcp.WithString("symbol", mcp.Description("Symbol/function to analyze")),
	), s.handleGetContributorInsights)

	// index_git_history - Index git history for temporal search
	mcpServer.AddTool(mcp.NewTool("index_git_history",
		mcp.WithDescription("Index git history for semantic temporal search"),
		mcp.WithString("since", mcp.Description("Index commits since date (ISO format) or 'all'")),
		mcp.WithBoolean("force", mcp.Description("Force reindex all commits")),
	), s.handleIndexGitHistory)

	// get_git_history_status - Status of git history index
	mcpServer.AddTool(mcp.NewTool("get_git_history_status",
		mcp.WithDescription("Get status and statistics of indexed git history"),
	), s.handleGetGitHistoryStatus)

	// Memory tools - Persistent context and knowledge

	// memory_store - Store a memory
	mcpServer.AddTool(mcp.NewTool("memory_store",
		mcp.WithDescription("Store a memory/knowledge entry for persistent context"),
		mcp.WithString("content", mcp.Required(), mcp.Description("Memory content")),
		mcp.WithString("category", mcp.Description("Category: decision, context, fact, note, error, review")),
		mcp.WithArray("tags", mcp.Description("Tags for organization")),
		mcp.WithNumber("importance", mcp.Description("Importance score 0-1 (default 0.5)")),
		mcp.WithString("channel", mcp.Description("Channel/branch for context isolation")),
		mcp.WithNumber("ttl_days", mcp.Description("Time-to-live in days (optional)")),
	), s.handleMemoryStore)

	// memory_recall - Recall memories
	mcpServer.AddTool(mcp.NewTool("memory_recall",
		mcp.WithDescription("Recall memories using semantic search"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 10)")),
		mcp.WithString("channel", mcp.Description("Channel/branch filter")),
		mcp.WithArray("categories", mcp.Description("Filter by categories")),
		mcp.WithArray("tags", mcp.Description("Filter by tags")),
		mcp.WithNumber("min_importance", mcp.Description("Minimum importance score")),
	), s.handleMemoryRecall)

	// memory_forget - Delete a memory
	mcpServer.AddTool(mcp.NewTool("memory_forget",
		mcp.WithDescription("Delete a specific memory"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Memory ID to delete")),
	), s.handleMemoryForget)

	// memory_checkpoint - Create a checkpoint
	mcpServer.AddTool(mcp.NewTool("memory_checkpoint",
		mcp.WithDescription("Create a checkpoint/snapshot of current memories"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Checkpoint name")),
		mcp.WithString("description", mcp.Description("Checkpoint description")),
		mcp.WithString("channel", mcp.Description("Channel/branch")),
	), s.handleMemoryCheckpoint)

	// memory_restore - Restore from checkpoint
	mcpServer.AddTool(mcp.NewTool("memory_restore",
		mcp.WithDescription("Restore memories from a checkpoint"),
		mcp.WithString("checkpoint_id", mcp.Required(), mcp.Description("Checkpoint ID to restore")),
	), s.handleMemoryRestore)

	// memory_stats - Get memory statistics
	mcpServer.AddTool(mcp.NewTool("memory_stats",
		mcp.WithDescription("Get memory statistics"),
		mcp.WithString("channel", mcp.Description("Channel/branch filter")),
	), s.handleMemoryStats)

	// Todo/Task tools

	// todo_create - Create a todo
	mcpServer.AddTool(mcp.NewTool("todo_create",
		mcp.WithDescription("Create a new todo/task"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
		mcp.WithString("description", mcp.Description("Task description")),
		mcp.WithString("priority", mcp.Description("Priority: low, medium, high, urgent (default medium)")),
		mcp.WithArray("tags", mcp.Description("Tags for organization")),
		mcp.WithString("channel", mcp.Description("Channel/branch for context isolation")),
		mcp.WithString("parent_id", mcp.Description("Parent task ID for subtasks")),
		mcp.WithString("due_date", mcp.Description("Due date (ISO format: 2024-01-01)")),
		mcp.WithString("effort", mcp.Description("Effort estimate (e.g., '2h', '1d', 'S/M/L')")),
	), s.handleTodoCreate)

	// todo_list - List todos
	mcpServer.AddTool(mcp.NewTool("todo_list",
		mcp.WithDescription("List todos/tasks"),
		mcp.WithString("channel", mcp.Description("Channel/branch filter")),
		mcp.WithArray("statuses", mcp.Description("Filter by status: pending, in_progress, completed, cancelled, blocked")),
		mcp.WithArray("priorities", mcp.Description("Filter by priority")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 20)")),
		mcp.WithBoolean("include_completed", mcp.Description("Include completed tasks")),
	), s.handleTodoList)

	// todo_search - Search todos
	mcpServer.AddTool(mcp.NewTool("todo_search",
		mcp.WithDescription("Search todos using semantic similarity"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 10)")),
		mcp.WithString("channel", mcp.Description("Channel/branch filter")),
		mcp.WithBoolean("include_children", mcp.Description("Include subtasks")),
	), s.handleTodoSearch)

	// todo_update - Update a todo
	mcpServer.AddTool(mcp.NewTool("todo_update",
		mcp.WithDescription("Update a todo/task"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("description", mcp.Description("New description")),
		mcp.WithString("status", mcp.Description("New status: pending, in_progress, completed, cancelled, blocked")),
		mcp.WithString("priority", mcp.Description("New priority")),
		mcp.WithNumber("progress", mcp.Description("Progress percentage 0-100")),
	), s.handleTodoUpdate)

	// todo_complete - Mark todo as complete
	mcpServer.AddTool(mcp.NewTool("todo_complete",
		mcp.WithDescription("Mark a todo as completed"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Task ID to complete")),
	), s.handleTodoComplete)

	// todo_delete - Delete a todo
	mcpServer.AddTool(mcp.NewTool("todo_delete",
		mcp.WithDescription("Delete a todo/task"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Task ID to delete")),
	), s.handleTodoDelete)

	// todo_stats - Get todo statistics
	mcpServer.AddTool(mcp.NewTool("todo_stats",
		mcp.WithDescription("Get todo statistics"),
		mcp.WithString("channel", mcp.Description("Channel/branch filter")),
	), s.handleTodoStats)
}

// Tool handlers

func (s *Server) handleIndexCodebase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	force := req.GetBool("force", false)

	slog.Info("starting indexing", "force", force)

	// Create indexer
	indexer := index.New(index.Config{
		ProjectDir: s.projectDir,
		Config:     s.config,
		Store:      s.store,
		Embedding:  s.embedding,
		Chunker:    s.chunker,
		OnProgress: func(p types.IndexProgress) {
			slog.Debug("progress", "phase", p.Phase, "files", p.ProcessedFiles, "chunks", p.ProcessedChunks)
		},
	})

	if err := indexer.Index(ctx, force); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("indexing failed: %v", err)), nil
	}

	// Get stats
	stats, err := s.store.GetStats()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	result := map[string]any{
		"success":    true,
		"files":      stats.IndexedFiles,
		"chunks":     stats.TotalChunks,
		"symbols":    stats.TotalSymbols,
		"references": stats.TotalReferences,
		"db_size":    formatBytes(stats.DBSizeBytes),
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleSearchCode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	limit := req.GetInt("limit", 10)
	modeStr := req.GetString("mode", "hybrid")
	noRerank := req.GetBool("no_rerank", false)
	includeContext := req.GetBool("include_context", false)
	languages := req.GetStringSlice("languages", nil)

	mode := types.SearchModeHybrid
	switch modeStr {
	case "vector":
		mode = types.SearchModeVector
	case "bm25":
		mode = types.SearchModeBM25
	}

	searchReq := &types.SearchRequest{
		Query:            query,
		Limit:            limit,
		Mode:             mode,
		UseReranker:      !noRerank,
		RerankCandidates: 100,
		IncludeContext:   includeContext,
		ContextLines:     5,
	}

	// Add language filter
	if len(languages) > 0 {
		searchReq.Filters = &types.SearchFilters{
			Languages: languages,
		}
	}

	results, err := s.search.Search(ctx, searchReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for _, r := range results {
		entry := map[string]any{
			"id":         r.Chunk.ID,
			"file":       r.Chunk.FilePath,
			"start_line": r.Chunk.StartLine,
			"end_line":   r.Chunk.EndLine,
			"language":   r.Chunk.Language,
			"type":       r.Chunk.ChunkType,
			"name":       r.Chunk.Name,
			"score":      r.Score,
			"content":    r.Chunk.Content,
		}

		if includeContext {
			if r.ContextBefore != "" {
				entry["context_before"] = r.ContextBefore
			}
			if r.ContextAfter != "" {
				entry["context_after"] = r.ContextAfter
			}
		}

		formatted = append(formatted, entry)
	}

	jsonResult, _ := json.MarshalIndent(formatted, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetChunk(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chunkID := req.GetString("chunk_id", "")
	if chunkID == "" {
		return mcp.NewToolResultError("chunk_id is required"), nil
	}

	contextLines := req.GetInt("context_lines", 5)

	// Get chunk directly by ID
	chunk, err := s.store.GetChunk(chunkID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get chunk: %v", err)), nil
	}

	if chunk == nil {
		return mcp.NewToolResultError("chunk not found"), nil
	}

	// Get context from file
	contextBefore, contextAfter := getChunkContext(chunk.FilePath, chunk.StartLine, chunk.EndLine, contextLines)

	result := map[string]any{
		"id":             chunk.ID,
		"file":           chunk.FilePath,
		"start_line":     chunk.StartLine,
		"end_line":       chunk.EndLine,
		"language":       chunk.Language,
		"type":           chunk.ChunkType,
		"name":           chunk.Name,
		"content":        chunk.Content,
		"context_before": contextBefore,
		"context_after":  contextAfter,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := s.store.GetStats()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	meta, _ := s.store.GetMetadata()

	result := map[string]any{
		"indexed_files": stats.IndexedFiles,
		"total_chunks":  stats.TotalChunks,
		"total_symbols": stats.TotalSymbols,
		"total_refs":    stats.TotalReferences,
		"db_size":       formatBytes(stats.DBSizeBytes),
		"last_indexed":  stats.LastIndexed.Format("2006-01-02 15:04:05"),
	}

	if meta != nil {
		result["embedding_model"] = meta.EmbeddingModel
		result["chunking_strategy"] = meta.ChunkingStrategy
		result["tool_version"] = meta.ToolVersion
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleClearIndex(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get all file hashes and delete chunks
	hashes, err := s.store.GetAllFileHashes()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get files: %v", err)), nil
	}

	for filePath := range hashes {
		if err := s.store.DeleteChunksByFile(filePath); err != nil {
			slog.Warn("failed to delete chunks", "file", filePath, "error", err)
		}
		if err := s.store.DeleteFileCache(filePath); err != nil {
			slog.Warn("failed to delete cache", "file", filePath, "error", err)
		}
	}

	return mcp.NewToolResultText(`{"success": true, "message": "Index cleared"}`), nil
}

func (s *Server) handleGetCallers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbol := req.GetString("symbol", "")
	if symbol == "" {
		return mcp.NewToolResultError("symbol is required"), nil
	}

	limit := req.GetInt("limit", 50)

	refs, err := s.search.GetCallers(symbol, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get callers: %v", err)), nil
	}

	var formatted []map[string]any
	for _, r := range refs {
		formatted = append(formatted, map[string]any{
			"from":        r.FromSymbol,
			"file":        r.FilePath,
			"line":        r.Line,
			"kind":        r.Kind,
			"is_external": r.IsExternal,
		})
	}

	jsonResult, _ := json.MarshalIndent(formatted, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetCallees(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbol := req.GetString("symbol", "")
	if symbol == "" {
		return mcp.NewToolResultError("symbol is required"), nil
	}

	limit := req.GetInt("limit", 50)

	refs, err := s.search.GetCallees(symbol, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get callees: %v", err)), nil
	}

	var formatted []map[string]any
	for _, r := range refs {
		formatted = append(formatted, map[string]any{
			"to":          r.ToSymbol,
			"file":        r.FilePath,
			"line":        r.Line,
			"kind":        r.Kind,
			"is_external": r.IsExternal,
		})
	}

	jsonResult, _ := json.MarshalIndent(formatted, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetSymbols(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	kindStr := req.GetString("kind", "")
	limit := req.GetInt("limit", 50)

	kind := types.SymbolKind(kindStr)

	symbols, err := s.search.SearchSymbols(query, kind, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search symbols: %v", err)), nil
	}

	var formatted []map[string]any
	for _, sym := range symbols {
		formatted = append(formatted, map[string]any{
			"id":         sym.ID,
			"name":       sym.Name,
			"kind":       sym.Kind,
			"file":       sym.FilePath,
			"start_line": sym.StartLine,
			"end_line":   sym.EndLine,
			"signature":  sym.Signature,
			"visibility": sym.Visibility,
		})
	}

	jsonResult, _ := json.MarshalIndent(formatted, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetEntryPoints(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := req.GetInt("limit", 100)
	filterType := req.GetString("type", "")

	entryPoints, err := s.search.GetEntryPoints(limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get entry points: %v", err)), nil
	}

	// Filter by type if specified
	if filterType != "" {
		filtered := make([]*search.EntryPoint, 0)
		for _, ep := range entryPoints {
			if ep.Type == filterType {
				filtered = append(filtered, ep)
			}
		}
		entryPoints = filtered
	}

	var formatted []map[string]any
	for _, ep := range entryPoints {
		formatted = append(formatted, map[string]any{
			"type":       ep.Type,
			"confidence": ep.Confidence,
			"name":       ep.Symbol.Name,
			"file":       ep.Symbol.FilePath,
			"start_line": ep.Symbol.StartLine,
			"end_line":   ep.Symbol.EndLine,
			"signature":  ep.Symbol.Signature,
		})
	}

	jsonResult, _ := json.MarshalIndent(formatted, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetImportGraph(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := req.GetInt("limit", 1000)

	graph, err := s.search.GetImportGraph(limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get import graph: %v", err)), nil
	}

	// Build summary
	result := map[string]any{
		"file_count":       graph.FileCount,
		"total_imports":    len(graph.Edges),
		"unique_packages":  len(graph.Imports),
		"top_imports":      getTopImports(graph.Imports, 20),
		"import_details":   graph.Edges,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// getTopImports returns the most imported packages.
func getTopImports(imports map[string]int, limit int) []map[string]any {
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range imports {
		sorted = append(sorted, kv{k, v})
	}
	// Sort by count descending
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Value > sorted[i].Value {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	var result []map[string]any
	for _, kv := range sorted {
		result = append(result, map[string]any{
			"package": kv.Key,
			"count":   kv.Value,
		})
	}
	return result
}

// Wizard tool handlers

func (s *Server) handleDetectEnvironment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	w := wizard.New(s.projectDir)
	result, err := w.DetectEnvironment(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to detect environment: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleValidateConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Load current config
	cfg, _, err := config.Load(s.projectDir)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load config: %v", err)), nil
	}

	w := wizard.New(s.projectDir)
	result, err := w.ValidateConfig(ctx, cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("validation failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Load current config
	cfg, source, err := config.Load(s.projectDir)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load config: %v", err)), nil
	}

	result := map[string]any{
		"source": source,
		"config": cfg,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleApplyRecommendation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	preset := req.GetString("preset", "")
	if preset == "" {
		return mcp.NewToolResultError("preset is required"), nil
	}

	// Get environment detection first
	w := wizard.New(s.projectDir)
	env, err := w.DetectEnvironment(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to detect environment: %v", err)), nil
	}

	if env.Recommendations == nil {
		return mcp.NewToolResultError("no recommendations available"), nil
	}

	// Get the recommendation set
	var recSet *wizard.RecommendationSet
	switch preset {
	case "primary":
		recSet = &env.Recommendations.Primary
	case "low_memory":
		recSet = env.Recommendations.LowMemory
	case "high_quality":
		recSet = env.Recommendations.HighQuality
	case "fast_index":
		recSet = env.Recommendations.FastIndex
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unknown preset: %s", preset)), nil
	}

	if recSet == nil {
		return mcp.NewToolResultError(fmt.Sprintf("preset %s not available", preset)), nil
	}

	// Build config from recommendation
	cfg := &config.Config{
		Embedding: config.EmbeddingConfig{
			Provider:  recSet.Embedding.Provider,
			Model:     recSet.Embedding.Model,
			Endpoint:  "http://localhost:11434",
			BatchSize: recSet.Embedding.BatchSize,
		},
		Reranker: config.RerankerConfig{
			Enabled:    recSet.Reranker.Enabled,
			Provider:   recSet.Reranker.Provider,
			Model:      recSet.Reranker.Model,
			Endpoint:   "http://localhost:11434",
			Candidates: recSet.Reranker.Candidates,
		},
		Chunking: config.ChunkingConfig{
			Strategy:     recSet.Chunking.Strategy,
			MaxChunkSize: recSet.Chunking.MaxChunkSize,
		},
		Index: config.IndexConfig{
			Include: []string{
				"**/*.go",
				"**/*.py",
				"**/*.js",
				"**/*.ts",
				"**/*.jsx",
				"**/*.tsx",
				"**/*.rs",
				"**/*.java",
				"**/*.c",
				"**/*.cpp",
				"**/*.h",
			},
			Exclude: []string{
				"**/node_modules/**",
				"**/.git/**",
				"**/vendor/**",
				"**/dist/**",
				"**/build/**",
			},
			UseGitIgnore: recSet.Indexing.UseGitIgnore,
		},
		Search: config.SearchConfig{
			Mode:         "hybrid",
			VectorWeight: 0.7,
			BM25Weight:   0.3,
			DefaultLimit: 10,
		},
		VectorStore: config.VectorStoreConfig{
			Provider: "sqlitevec",
		},
		Limits: config.LimitsConfig{
			MaxFileSize:   "1MB",
			MaxFiles:      50000,
			MaxChunkTokens: 2000,
			Workers:       recSet.Indexing.Workers,
		},
		Analysis: config.AnalysisConfig{
			ExtractSymbols:    true,
			ExtractReferences: true,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Write config
	if err := config.Save(s.projectDir, cfg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save config: %v", err)), nil
	}

	result := map[string]any{
		"success":     true,
		"preset":      preset,
		"config_path": config.ConfigPath(s.projectDir),
		"config":      cfg,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// ServeStdio starts the MCP server using stdio transport.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// formatBytes formats bytes to human readable string.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// getChunkContext reads context lines before and after the chunk from the source file.
func getChunkContext(filePath string, startLine, endLine, contextLines int) (before, after string) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", ""
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", ""
	}

	// Get context before (1-indexed to 0-indexed)
	beforeStart := startLine - 1 - contextLines
	if beforeStart < 0 {
		beforeStart = 0
	}
	beforeEnd := startLine - 1
	if beforeEnd > len(lines) {
		beforeEnd = len(lines)
	}
	if beforeStart < beforeEnd {
		before = strings.Join(lines[beforeStart:beforeEnd], "\n")
	}

	// Get context after
	afterStart := endLine // endLine is 1-indexed, so this is the line after
	if afterStart < 0 {
		afterStart = 0
	}
	afterEnd := endLine + contextLines
	if afterEnd > len(lines) {
		afterEnd = len(lines)
	}
	if afterStart < afterEnd {
		after = strings.Join(lines[afterStart:afterEnd], "\n")
	}

	return before, after
}

// Analysis handlers

func (s *Server) handleGetBlame(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := req.GetString("file", "")
	if file == "" {
		return mcp.NewToolResultError("file is required"), nil
	}

	startLine := req.GetInt("start_line", 0)
	endLine := req.GetInt("end_line", 0)

	// Make absolute path
	absPath := filepath.Join(s.projectDir, file)

	// Create blame analyzer
	blameAnalyzer := analysis.NewBlameAnalyzer(s.projectDir)

	// Check if git repo
	if !blameAnalyzer.IsGitRepo() {
		return mcp.NewToolResultError("not a git repository"), nil
	}

	var result any
	var err error

	if startLine > 0 && endLine > 0 {
		// Get range blame
		result, err = blameAnalyzer.GetRangeBlame(absPath, startLine, endLine)
	} else {
		// Get full file blame
		result, err = blameAnalyzer.GetFileBlame(absPath)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("git blame failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetDeadCode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := req.GetInt("limit", 20)
	typeStr := req.GetString("type", "functions")

	deadCodeAnalyzer := analysis.NewDeadCodeAnalyzer(s.store)

	var results []*analysis.DeadCodeResult
	var err error

	switch typeStr {
	case "functions":
		results, err = deadCodeAnalyzer.FindDeadFunctions(limit)
	case "types":
		results, err = deadCodeAnalyzer.FindUnusedTypes(limit)
	case "all":
		results, err = deadCodeAnalyzer.FindDeadCode(limit)
	default:
		results, err = deadCodeAnalyzer.FindDeadFunctions(limit)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("dead code analysis failed: %v", err)), nil
	}

	response := map[string]any{
		"count":   len(results),
		"type":    typeStr,
		"results": results,
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetComplexity(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := req.GetString("file", "")
	if file == "" {
		return mcp.NewToolResultError("file is required"), nil
	}

	startLine := req.GetInt("start_line", 0)
	endLine := req.GetInt("end_line", 0)

	// Make absolute path
	absPath := filepath.Join(s.projectDir, file)

	// Create complexity analyzer
	complexityAnalyzer := analysis.NewComplexityAnalyzer()

	var result any
	var err error

	if startLine > 0 && endLine > 0 {
		// Get range complexity
		result, err = complexityAnalyzer.AnalyzeRange(absPath, startLine, endLine)
	} else {
		// Get full file complexity
		result, err = complexityAnalyzer.AnalyzeFile(absPath)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("complexity analysis failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Git History handlers

func (s *Server) handleSearchHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// Parse time filters
	var timeFrom, timeTo *time.Time
	if tf := req.GetString("time_from", ""); tf != "" {
		if t, err := time.Parse("2006-01-02", tf); err == nil {
			timeFrom = &t
		}
	}
	if tt := req.GetString("time_to", ""); tt != "" {
		if t, err := time.Parse("2006-01-02", tt); err == nil {
			timeTo = &t
		}
	}

	// Parse array filters
	authors := req.GetStringSlice("authors", nil)
	paths := req.GetStringSlice("paths", nil)
	changeTypesStr := req.GetStringSlice("change_types", nil)
	functions := req.GetStringSlice("functions", nil)

	var changeTypes []types.ChangeType
	for _, ct := range changeTypesStr {
		changeTypes = append(changeTypes, types.ChangeType(ct))
	}

	limit := req.GetInt("limit", 10)
	includeDiff := req.GetBool("include_diff", false)

	// Generate query embedding
	var queryVec []float32
	if s.embedding != nil {
		embeddings, err := s.embedding.Embed(ctx, []string{query})
		if err == nil && len(embeddings) > 0 {
			queryVec = embeddings[0]
		}
	}

	// Build search request
	searchReq := &types.HistorySearchRequest{
		Query:           query,
		QueryVec:        queryVec,
		TimeFrom:        timeFrom,
		TimeTo:          timeTo,
		Authors:         authors,
		Paths:           paths,
		ChangeTypes:     changeTypes,
		Functions:       functions,
		Limit:           limit,
		IncludeDiff:     includeDiff,
		IncludeContext:  true,
		TimeDecayFactor: 0.5,
	}

	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		SearchHistory(ctx context.Context, req *types.HistorySearchRequest) ([]*types.HistorySearchResult, error)
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	results, err := historyStore.SearchHistory(ctx, searchReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("history search failed: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for _, r := range results {
		entry := map[string]any{
			"score":       r.Score,
			"commit_hash": r.Commit.ShortHash,
			"commit_date": r.Commit.Date.Format("2006-01-02 15:04"),
			"author":      r.Commit.Author,
			"message":     r.Commit.Message,
			"file":        r.Change.FilePath,
			"change_type": r.Change.ChangeType,
			"additions":   r.Change.Additions,
			"deletions":   r.Change.Deletions,
		}

		if len(r.Change.AffectedFunctions) > 0 {
			entry["affected_functions"] = r.Change.AffectedFunctions
		}

		if includeDiff && r.Change.DiffContent != "" {
			// Truncate long diffs
			diff := r.Change.DiffContent
			if len(diff) > 2000 {
				diff = diff[:2000] + "\n... (truncated)"
			}
			entry["diff"] = diff
		}

		if len(r.RelatedChanges) > 0 {
			var related []string
			for _, rc := range r.RelatedChanges {
				related = append(related, rc.FilePath)
			}
			entry["related_files"] = related
		}

		formatted = append(formatted, entry)
	}

	response := map[string]any{
		"query":   query,
		"count":   len(formatted),
		"results": formatted,
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetChunkHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chunkID := req.GetString("chunk_id", "")
	symbol := req.GetString("symbol", "")
	file := req.GetString("file", "")
	limit := req.GetInt("limit", 20)

	if chunkID == "" && symbol == "" {
		return mcp.NewToolResultError("either chunk_id or symbol is required"), nil
	}

	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		GetChunkHistory(chunkID string, limit int) ([]*types.ChunkHistoryEntry, error)
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	// If symbol provided, find the chunk ID first
	if chunkID == "" && symbol != "" {
		symbols, err := s.store.FindSymbols(symbol, "", 10)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to find symbol: %v", err)), nil
		}

		// Filter by file if provided
		for _, sym := range symbols {
			if file == "" || strings.Contains(sym.FilePath, file) {
				// Construct chunk ID from symbol
				chunkID = fmt.Sprintf("%s:%d:", sym.FilePath, sym.StartLine)
				break
			}
		}

		if chunkID == "" {
			return mcp.NewToolResultError("symbol not found"), nil
		}
	}

	history, err := historyStore.GetChunkHistory(chunkID, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get chunk history: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for _, h := range history {
		formatted = append(formatted, map[string]any{
			"commit":      h.CommitHash[:8],
			"date":        h.Date.Format("2006-01-02 15:04"),
			"author":      h.Author,
			"change_type": h.ChangeType,
			"summary":     h.DiffSummary,
		})
	}

	response := map[string]any{
		"chunk_id": chunkID,
		"count":    len(formatted),
		"history":  formatted,
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetCodeEvolution(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbol := req.GetString("symbol", "")
	if symbol == "" {
		return mcp.NewToolResultError("symbol is required"), nil
	}
	file := req.GetString("file", "")

	// Find symbol first
	symbols, err := s.store.FindSymbols(symbol, "", 10)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find symbol: %v", err)), nil
	}

	var targetSymbol *types.Symbol
	for _, sym := range symbols {
		if file == "" || strings.Contains(sym.FilePath, file) {
			targetSymbol = sym
			break
		}
	}

	if targetSymbol == nil {
		return mcp.NewToolResultError("symbol not found"), nil
	}

	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		GetChunkEvolution(chunkID string) (*types.ChunkEvolution, error)
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	// Try to get chunk ID
	chunkID := fmt.Sprintf("%s:%d:", targetSymbol.FilePath, targetSymbol.StartLine)

	evolution, err := historyStore.GetChunkEvolution(chunkID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get evolution: %v", err)), nil
	}

	// Format result
	result := map[string]any{
		"symbol":        symbol,
		"file":          targetSymbol.FilePath,
		"created_at":    evolution.CreatedAt.Format("2006-01-02"),
		"created_by":    evolution.CreatedBy,
		"last_modified": evolution.LastModified.Format("2006-01-02"),
		"modify_count":  evolution.ModifyCount,
		"authors":       evolution.Authors,
	}

	if len(evolution.History) > 0 {
		var history []map[string]any
		for _, h := range evolution.History {
			history = append(history, map[string]any{
				"commit":      h.CommitHash[:8],
				"date":        h.Date.Format("2006-01-02"),
				"author":      h.Author,
				"change_type": h.ChangeType,
				"summary":     h.DiffSummary,
			})
		}
		result["history"] = history
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleFindRegression(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	description := req.GetString("description", "")
	if description == "" {
		return mcp.NewToolResultError("description is required"), nil
	}

	knownGood := req.GetString("known_good", "")
	knownBad := req.GetString("known_bad", "HEAD")
	limit := req.GetInt("limit", 10)

	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		FindRegressionCandidates(ctx context.Context, description string, knownGood, knownBad string, limit int) ([]*types.RegressionCandidate, error)
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	candidates, err := historyStore.FindRegressionCandidates(ctx, description, knownGood, knownBad, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("regression search failed: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for i, c := range candidates {
		severity := "LOW"
		if c.Score > 0.7 {
			severity = "HIGH"
		} else if c.Score > 0.4 {
			severity = "MEDIUM"
		}

		entry := map[string]any{
			"rank":        i + 1,
			"severity":    severity,
			"score":       c.Score,
			"commit":      c.Commit.ShortHash,
			"date":        c.Commit.Date.Format("2006-01-02 15:04"),
			"author":      c.Commit.Author,
			"message":     c.Commit.Message,
			"reasoning":   c.Reasoning,
			"files_changed": len(c.Changes),
		}

		if c.DiffPreview != "" {
			entry["diff_preview"] = c.DiffPreview
		}

		// List changed files
		var changedFiles []string
		for _, ch := range c.Changes {
			changedFiles = append(changedFiles, ch.FilePath)
		}
		entry["changed_files"] = changedFiles

		formatted = append(formatted, entry)
	}

	response := map[string]any{
		"description": description,
		"range":       fmt.Sprintf("%s..%s", knownGood, knownBad),
		"count":       len(formatted),
		"candidates":  formatted,
	}

	if len(candidates) > 0 {
		response["recommendation"] = fmt.Sprintf("Start investigating commit %s - %s",
			candidates[0].Commit.ShortHash, candidates[0].Commit.Message)
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetCommitContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	commitHash := req.GetString("commit", "")
	if commitHash == "" {
		return mcp.NewToolResultError("commit is required"), nil
	}
	includeDiff := req.GetBool("include_diff", false)

	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		GetCommit(hash string) (*types.Commit, error)
		GetChangesByCommit(commitHash string) ([]*types.Change, error)
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	commit, err := historyStore.GetCommit(commitHash)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get commit: %v", err)), nil
	}
	if commit == nil {
		return mcp.NewToolResultError("commit not found"), nil
	}

	changes, err := historyStore.GetChangesByCommit(commit.Hash)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get changes: %v", err)), nil
	}

	// Format changes
	var changesFormatted []map[string]any
	for _, ch := range changes {
		entry := map[string]any{
			"file":        ch.FilePath,
			"change_type": ch.ChangeType,
			"additions":   ch.Additions,
			"deletions":   ch.Deletions,
		}

		if len(ch.AffectedFunctions) > 0 {
			entry["affected_functions"] = ch.AffectedFunctions
		}

		if includeDiff && ch.DiffContent != "" {
			diff := ch.DiffContent
			if len(diff) > 2000 {
				diff = diff[:2000] + "\n... (truncated)"
			}
			entry["diff"] = diff
		}

		changesFormatted = append(changesFormatted, entry)
	}

	result := map[string]any{
		"hash":          commit.Hash,
		"short_hash":    commit.ShortHash,
		"author":        commit.Author,
		"author_email":  commit.AuthorEmail,
		"date":          commit.Date.Format("2006-01-02 15:04:05"),
		"message":       commit.Message,
		"parent":        commit.ParentHash,
		"is_merge":      commit.IsMerge,
		"files_changed": commit.FilesChanged,
		"insertions":    commit.Insertions,
		"deletions":     commit.Deletions,
		"changes":       changesFormatted,
	}

	if len(commit.Tags) > 0 {
		result["tags"] = commit.Tags
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetContributorInsights(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	paths := req.GetStringSlice("paths", nil)
	symbol := req.GetString("symbol", "")

	if len(paths) == 0 && symbol == "" {
		return mcp.NewToolResultError("either paths or symbol is required"), nil
	}

	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		GetContributorInsights(ctx context.Context, paths []string, symbol string) ([]*types.ContributorInsight, error)
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	insights, err := historyStore.GetContributorInsights(ctx, paths, symbol)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get insights: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for _, ins := range insights {
		formatted = append(formatted, map[string]any{
			"author":        ins.Author,
			"email":         ins.Email,
			"commit_count":  ins.CommitCount,
			"lines_changed": ins.LinesChanged,
			"last_active":   ins.LastActive.Format("2006-01-02"),
			"expertise":     fmt.Sprintf("%.0f%%", ins.Expertise*100),
		})
	}

	response := map[string]any{
		"query":        map[string]any{"paths": paths, "symbol": symbol},
		"contributors": formatted,
	}

	if len(insights) > 0 {
		response["top_expert"] = insights[0].Author
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleIndexGitHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	since := req.GetString("since", "30 days ago")
	force := req.GetBool("force", false)

	// Create git history analyzer
	gitAnalyzer := analysis.NewGitHistoryAnalyzer(s.projectDir, nil)

	// Check if git repo
	blameAnalyzer := analysis.NewBlameAnalyzer(s.projectDir)
	if !blameAnalyzer.IsGitRepo() {
		return mcp.NewToolResultError("not a git repository"), nil
	}

	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		GetLastIndexedCommit() (string, error)
		SetLastIndexedCommit(hash string) error
		StoreCommits(commits []*types.Commit) error
		StoreChanges(changes []*types.Change) error
		StoreChunkHistory(entries []*types.ChunkHistoryEntry) error
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	// Determine starting point
	var commits []*types.Commit
	var err error

	if force || since == "all" {
		commits, err = gitAnalyzer.GetCommits("", 0)
	} else {
		lastIndexed, _ := historyStore.GetLastIndexedCommit()
		if lastIndexed != "" && !force {
			commits, err = gitAnalyzer.GetCommitsSinceHash(lastIndexed)
		} else {
			commits, err = gitAnalyzer.GetCommits(since, 0)
		}
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get commits: %v", err)), nil
	}

	if len(commits) == 0 {
		return mcp.NewToolResultText(`{"message": "No new commits to index", "indexed": 0}`), nil
	}

	// Generate embeddings for commit messages
	if s.embedding != nil {
		var messages []string
		for _, c := range commits {
			messages = append(messages, c.Message)
		}

		embeddings, err := s.embedding.Embed(ctx, messages)
		if err == nil {
			for i, emb := range embeddings {
				if i < len(commits) {
					commits[i].MessageEmbedding = emb
				}
			}
		}
	}

	// Store commits
	if err := historyStore.StoreCommits(commits); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to store commits: %v", err)), nil
	}

	// Process changes for each commit
	totalChanges := 0
	for _, commit := range commits {
		changes, err := gitAnalyzer.GetChangesForCommit(commit.Hash)
		if err != nil {
			continue
		}

		// Identify affected functions
		for _, change := range changes {
			change.AffectedFunctions = gitAnalyzer.IdentifyAffectedFunctions(change)
		}

		// Store changes
		if err := historyStore.StoreChanges(changes); err != nil {
			continue
		}

		totalChanges += len(changes)
	}

	// Update last indexed commit
	if len(commits) > 0 {
		_ = historyStore.SetLastIndexedCommit(commits[0].Hash)
	}

	result := map[string]any{
		"success":        true,
		"commits_indexed": len(commits),
		"changes_indexed": totalChanges,
		"newest_commit":  commits[0].ShortHash,
		"oldest_commit":  commits[len(commits)-1].ShortHash,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetGitHistoryStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if store supports git history
	historyStore, ok := s.store.(interface {
		GetGitHistoryStats() (*types.GitHistoryStats, error)
		GetLastIndexedCommit() (string, error)
	})
	if !ok {
		return mcp.NewToolResultError("git history not supported by current store"), nil
	}

	stats, err := historyStore.GetGitHistoryStats()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	lastCommit, _ := historyStore.GetLastIndexedCommit()

	result := map[string]any{
		"total_commits":           stats.TotalCommits,
		"total_changes":           stats.TotalChanges,
		"commits_with_embeddings": stats.CommitsWithEmbeddings,
		"changes_with_embeddings": stats.ChangesWithEmbeddings,
		"unique_authors":          stats.UniqueAuthors,
		"last_indexed_commit":     lastCommit,
	}

	if !stats.OldestCommit.IsZero() {
		result["oldest_commit"] = stats.OldestCommit.Format("2006-01-02")
	}
	if !stats.NewestCommit.IsZero() {
		result["newest_commit"] = stats.NewestCommit.Format("2006-01-02")
	}
	if !stats.LastIndexed.IsZero() {
		result["last_indexed_at"] = stats.LastIndexed.Format("2006-01-02 15:04:05")
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Memory handlers

func (s *Server) handleMemoryStore(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	if content == "" {
		return mcp.NewToolResultError("content is required"), nil
	}

	// Check if store supports memory
	memoryStore, ok := s.store.(provider.MemoryStore)
	if !ok {
		return mcp.NewToolResultError("memory not supported by current store"), nil
	}

	// Parse parameters
	categoryStr := req.GetString("category", "note")
	tags := req.GetStringSlice("tags", nil)
	importance := float32(req.GetFloat("importance", 0.5))
	channel := req.GetString("channel", s.getCurrentBranch())
	ttlDays := req.GetInt("ttl_days", 0)

	// Parse category
	category := types.MemoryCategory(categoryStr)

	// Generate embedding
	var embedding []float32
	if s.embedding != nil {
		embeddings, err := s.embedding.Embed(ctx, []string{content})
		if err == nil && len(embeddings) > 0 {
			embedding = embeddings[0]
		}
	}

	// Create memory entry
	now := time.Now()
	memory := &types.MemoryEntry{
		ID:          fmt.Sprintf("mem_%d", now.UnixNano()),
		Content:     content,
		Embedding:   embedding,
		Category:    category,
		Tags:        tags,
		Importance:  importance,
		Confidence:  1.0,
		Channel:     channel,
		CreatedAt:   now,
		UpdatedAt:   now,
		AccessCount: 0,
	}

	// Set TTL if provided
	if ttlDays > 0 {
		expiresAt := now.AddDate(0, 0, ttlDays)
		memory.ExpiresAt = &expiresAt
	}

	if err := memoryStore.StoreMemory(memory); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to store memory: %v", err)), nil
	}

	result := map[string]any{
		"success": true,
		"id":      memory.ID,
		"channel": memory.Channel,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleMemoryRecall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// Check if store supports memory
	memoryStore, ok := s.store.(provider.MemoryStore)
	if !ok {
		return mcp.NewToolResultError("memory not supported by current store"), nil
	}

	limit := req.GetInt("limit", 10)
	channel := req.GetString("channel", s.getCurrentBranch())
	categoriesStr := req.GetStringSlice("categories", nil)
	tags := req.GetStringSlice("tags", nil)
	minImportance := float32(req.GetFloat("min_importance", 0.0))

	// Convert categories
	var categories []types.MemoryCategory
	for _, c := range categoriesStr {
		categories = append(categories, types.MemoryCategory(c))
	}

	// Generate query embedding
	var queryVec []float32
	if s.embedding != nil {
		embeddings, err := s.embedding.Embed(ctx, []string{query})
		if err == nil && len(embeddings) > 0 {
			queryVec = embeddings[0]
		}
	}

	searchReq := &types.MemorySearchRequest{
		Query:         query,
		QueryVec:      queryVec,
		Channel:       channel,
		Categories:    categories,
		Tags:          tags,
		MinImportance: minImportance,
		Limit:         limit,
	}

	results, err := memoryStore.SearchMemories(ctx, searchReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("memory recall failed: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for _, r := range results {
		entry := map[string]any{
			"id":         r.Memory.ID,
			"content":    r.Memory.Content,
			"category":   r.Memory.Category,
			"importance": r.Memory.Importance,
			"confidence": r.Memory.Confidence,
			"score":      r.Score,
			"created_at": r.Memory.CreatedAt.Format("2006-01-02 15:04"),
		}

		if len(r.Memory.Tags) > 0 {
			entry["tags"] = r.Memory.Tags
		}
		if r.Memory.Summary != "" {
			entry["summary"] = r.Memory.Summary
		}

		formatted = append(formatted, entry)
	}

	response := map[string]any{
		"query":    query,
		"channel":  channel,
		"count":    len(formatted),
		"memories": formatted,
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleMemoryForget(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	// Check if store supports memory
	memoryStore, ok := s.store.(provider.MemoryStore)
	if !ok {
		return mcp.NewToolResultError("memory not supported by current store"), nil
	}

	if err := memoryStore.DeleteMemory(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete memory: %v", err)), nil
	}

	result := map[string]any{
		"success": true,
		"id":      id,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleMemoryCheckpoint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	// Check if store supports memory
	memoryStore, ok := s.store.(provider.MemoryStore)
	if !ok {
		return mcp.NewToolResultError("memory not supported by current store"), nil
	}

	description := req.GetString("description", "")
	channel := req.GetString("channel", s.getCurrentBranch())

	now := time.Now()
	checkpoint := &types.MemoryCheckpoint{
		ID:          fmt.Sprintf("chk_%d", now.UnixNano()),
		Name:        name,
		Description: description,
		Channel:     channel,
		CreatedAt:   now,
	}

	if err := memoryStore.CreateCheckpoint(checkpoint); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create checkpoint: %v", err)), nil
	}

	result := map[string]any{
		"success":    true,
		"id":         checkpoint.ID,
		"name":       checkpoint.Name,
		"channel":    checkpoint.Channel,
		"created_at": checkpoint.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleMemoryRestore(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checkpointID := req.GetString("checkpoint_id", "")
	if checkpointID == "" {
		return mcp.NewToolResultError("checkpoint_id is required"), nil
	}

	// Check if store supports memory
	memoryStore, ok := s.store.(provider.MemoryStore)
	if !ok {
		return mcp.NewToolResultError("memory not supported by current store"), nil
	}

	if err := memoryStore.RestoreCheckpoint(checkpointID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to restore checkpoint: %v", err)), nil
	}

	result := map[string]any{
		"success":       true,
		"checkpoint_id": checkpointID,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleMemoryStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if store supports memory
	memoryStore, ok := s.store.(provider.MemoryStore)
	if !ok {
		return mcp.NewToolResultError("memory not supported by current store"), nil
	}

	channel := req.GetString("channel", "")

	stats, err := memoryStore.GetMemoryStats(channel)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	result := map[string]any{
		"total_memories":    stats.TotalMemories,
		"active_sessions":   stats.ActiveSessions,
		"total_checkpoints": stats.TotalCheckpoints,
		"expired_count":     stats.ExpiredCount,
	}

	if channel != "" {
		result["channel"] = channel
	}

	if len(stats.MemoriesByCategory) > 0 {
		result["by_category"] = stats.MemoriesByCategory
	}
	if len(stats.MemoriesByChannel) > 0 {
		result["by_channel"] = stats.MemoriesByChannel
	}

	if stats.OldestMemory != nil {
		result["oldest_memory"] = stats.OldestMemory.Format("2006-01-02")
	}
	if stats.NewestMemory != nil {
		result["newest_memory"] = stats.NewestMemory.Format("2006-01-02")
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Todo handlers

func (s *Server) handleTodoCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := req.GetString("title", "")
	if title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	// Check if store supports todos
	todoStore, ok := s.store.(provider.TodoStore)
	if !ok {
		return mcp.NewToolResultError("todos not supported by current store"), nil
	}

	description := req.GetString("description", "")
	priorityStr := req.GetString("priority", "medium")
	tags := req.GetStringSlice("tags", nil)
	channel := req.GetString("channel", s.getCurrentBranch())
	parentID := req.GetString("parent_id", "")
	dueDateStr := req.GetString("due_date", "")
	effort := req.GetString("effort", "")

	// Parse priority
	priority := types.TodoPriority(priorityStr)

	// Parse due date
	var dueDate *time.Time
	if dueDateStr != "" {
		if t, err := time.Parse("2006-01-02", dueDateStr); err == nil {
			dueDate = &t
		}
	}

	// Generate embedding for searchability
	var embedding []float32
	if s.embedding != nil {
		text := title
		if description != "" {
			text += " " + description
		}
		embeddings, err := s.embedding.Embed(ctx, []string{text})
		if err == nil && len(embeddings) > 0 {
			embedding = embeddings[0]
		}
	}

	now := time.Now()
	todo := &types.TodoItem{
		ID:          fmt.Sprintf("todo_%d", now.UnixNano()),
		Title:       title,
		Description: description,
		Status:      types.TodoStatusPending,
		Priority:    priority,
		Tags:        tags,
		Embedding:   embedding,
		Channel:     channel,
		ParentID:    parentID,
		DueDate:     dueDate,
		Effort:      effort,
		Progress:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := todoStore.StoreTodo(todo); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create todo: %v", err)), nil
	}

	result := map[string]any{
		"success":  true,
		"id":       todo.ID,
		"title":    todo.Title,
		"status":   todo.Status,
		"priority": todo.Priority,
		"channel":  todo.Channel,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleTodoList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if store supports todos
	todoStore, ok := s.store.(provider.TodoStore)
	if !ok {
		return mcp.NewToolResultError("todos not supported by current store"), nil
	}

	channel := req.GetString("channel", s.getCurrentBranch())
	statusesStr := req.GetStringSlice("statuses", nil)
	limit := req.GetInt("limit", 20)
	includeCompleted := req.GetBool("include_completed", false)

	// Parse statuses
	var statuses []types.TodoStatus
	if len(statusesStr) > 0 {
		for _, st := range statusesStr {
			statuses = append(statuses, types.TodoStatus(st))
		}
	} else if !includeCompleted {
		// Default: show non-completed
		statuses = []types.TodoStatus{
			types.TodoStatusPending,
			types.TodoStatusInProgress,
			types.TodoStatusBlocked,
		}
	}

	todos, err := todoStore.ListTodos(ctx, channel, statuses, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list todos: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for _, t := range todos {
		entry := map[string]any{
			"id":       t.ID,
			"title":    t.Title,
			"status":   t.Status,
			"priority": t.Priority,
			"progress": t.Progress,
		}

		if t.Description != "" {
			entry["description"] = t.Description
		}
		if len(t.Tags) > 0 {
			entry["tags"] = t.Tags
		}
		if t.DueDate != nil {
			entry["due_date"] = t.DueDate.Format("2006-01-02")
		}
		if t.ParentID != "" {
			entry["parent_id"] = t.ParentID
		}

		formatted = append(formatted, entry)
	}

	response := map[string]any{
		"channel": channel,
		"count":   len(formatted),
		"todos":   formatted,
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleTodoSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// Check if store supports todos
	todoStore, ok := s.store.(provider.TodoStore)
	if !ok {
		return mcp.NewToolResultError("todos not supported by current store"), nil
	}

	limit := req.GetInt("limit", 10)
	channel := req.GetString("channel", s.getCurrentBranch())
	includeChildren := req.GetBool("include_children", false)

	// Generate query embedding
	var queryVec []float32
	if s.embedding != nil {
		embeddings, err := s.embedding.Embed(ctx, []string{query})
		if err == nil && len(embeddings) > 0 {
			queryVec = embeddings[0]
		}
	}

	searchReq := &types.TodoSearchRequest{
		Query:           query,
		QueryVec:        queryVec,
		Channel:         channel,
		Limit:           limit,
		IncludeChildren: includeChildren,
	}

	results, err := todoStore.SearchTodos(ctx, searchReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("todo search failed: %v", err)), nil
	}

	// Format results
	var formatted []map[string]any
	for _, r := range results {
		entry := map[string]any{
			"id":       r.Todo.ID,
			"title":    r.Todo.Title,
			"status":   r.Todo.Status,
			"priority": r.Todo.Priority,
			"score":    r.Score,
		}

		if r.Todo.Description != "" {
			entry["description"] = r.Todo.Description
		}

		formatted = append(formatted, entry)
	}

	response := map[string]any{
		"query":   query,
		"channel": channel,
		"count":   len(formatted),
		"results": formatted,
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleTodoUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	// Check if store supports todos
	todoStore, ok := s.store.(provider.TodoStore)
	if !ok {
		return mcp.NewToolResultError("todos not supported by current store"), nil
	}

	// Get existing todo
	todo, err := todoStore.GetTodo(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get todo: %v", err)), nil
	}
	if todo == nil {
		return mcp.NewToolResultError("todo not found"), nil
	}

	// Update fields if provided
	if title := req.GetString("title", ""); title != "" {
		todo.Title = title
	}
	if description := req.GetString("description", ""); description != "" {
		todo.Description = description
	}
	if statusStr := req.GetString("status", ""); statusStr != "" {
		todo.Status = types.TodoStatus(statusStr)
		if todo.Status == types.TodoStatusCompleted && todo.CompletedAt == nil {
			now := time.Now()
			todo.CompletedAt = &now
		}
		if todo.Status == types.TodoStatusInProgress && todo.StartedAt == nil {
			now := time.Now()
			todo.StartedAt = &now
		}
	}
	if priorityStr := req.GetString("priority", ""); priorityStr != "" {
		todo.Priority = types.TodoPriority(priorityStr)
	}
	if progress := req.GetInt("progress", -1); progress >= 0 {
		todo.Progress = progress
	}

	todo.UpdatedAt = time.Now()

	// Re-generate embedding if title/description changed
	if s.embedding != nil && (req.GetString("title", "") != "" || req.GetString("description", "") != "") {
		text := todo.Title
		if todo.Description != "" {
			text += " " + todo.Description
		}
		embeddings, err := s.embedding.Embed(ctx, []string{text})
		if err == nil && len(embeddings) > 0 {
			todo.Embedding = embeddings[0]
		}
	}

	if err := todoStore.UpdateTodo(todo); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update todo: %v", err)), nil
	}

	result := map[string]any{
		"success":  true,
		"id":       todo.ID,
		"title":    todo.Title,
		"status":   todo.Status,
		"priority": todo.Priority,
		"progress": todo.Progress,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleTodoComplete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	// Check if store supports todos
	todoStore, ok := s.store.(provider.TodoStore)
	if !ok {
		return mcp.NewToolResultError("todos not supported by current store"), nil
	}

	if err := todoStore.CompleteTodo(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to complete todo: %v", err)), nil
	}

	result := map[string]any{
		"success": true,
		"id":      id,
		"status":  "completed",
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleTodoDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	// Check if store supports todos
	todoStore, ok := s.store.(provider.TodoStore)
	if !ok {
		return mcp.NewToolResultError("todos not supported by current store"), nil
	}

	if err := todoStore.DeleteTodo(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete todo: %v", err)), nil
	}

	result := map[string]any{
		"success": true,
		"id":      id,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleTodoStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if store supports todos
	todoStore, ok := s.store.(provider.TodoStore)
	if !ok {
		return mcp.NewToolResultError("todos not supported by current store"), nil
	}

	channel := req.GetString("channel", "")

	stats, err := todoStore.GetTodoStats(channel)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	result := map[string]any{
		"total":            stats.TotalTodos,
		"overdue":          stats.OverdueTodos,
		"completed_today":  stats.CompletedToday,
		"completed_week":   stats.CompletedThisWeek,
	}

	if channel != "" {
		result["channel"] = channel
	}

	if len(stats.TodosByStatus) > 0 {
		result["by_status"] = stats.TodosByStatus
	}

	if len(stats.TodosByPriority) > 0 {
		result["by_priority"] = stats.TodosByPriority
	}

	if len(stats.TodosByChannel) > 0 {
		result["by_channel"] = stats.TodosByChannel
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// getCurrentBranch returns the current git branch or "default" if not in a git repo.
func (s *Server) getCurrentBranch() string {
	// Try to read git HEAD
	headPath := filepath.Join(s.projectDir, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "default"
	}

	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}

	// Detached HEAD or other state
	if len(content) >= 8 {
		return content[:8] // Short hash
	}
	return "default"
}
