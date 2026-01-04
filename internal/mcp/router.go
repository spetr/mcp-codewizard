// Package mcp implements the MCP server for code search.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolCategory groups related tools.
type ToolCategory struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Tools       []ToolInfo `json:"tools"`
}

// ToolInfo contains metadata about a tool.
type ToolInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Required    []string `json:"required,omitempty"`
	Optional    []string `json:"optional,omitempty"`
}

// toolCategories defines the grouping of tools with detailed descriptions.
var toolCategories = []ToolCategory{
	{
		Name:        "search",
		Description: "Additional search tools beyond search_code and grep_code",
		Tools: []ToolInfo{
			{Name: "search_code", Description: "[DIRECT] Semantic search by meaning", Required: []string{"query"}, Optional: []string{"limit", "mode", "languages"}},
			{Name: "grep_code", Description: "[DIRECT] Exact text/regex search", Required: []string{"pattern"}, Optional: []string{"path", "context_lines", "max_results"}},
			{Name: "fuzzy_search", Description: "Find symbols with typos/partial names - e.g. 'hndlAuth' finds 'handleAuth'", Required: []string{"query"}, Optional: []string{"kind", "type", "limit"}},
			{Name: "get_chunk", Description: "[DIRECT] Get full code for chunk ID from search", Required: []string{"chunk_id"}, Optional: []string{"context_lines"}},
		},
	},
	{
		Name:        "analysis",
		Description: "Code quality, complexity, and dependency analysis",
		Tools: []ToolInfo{
			{Name: "get_callers", Description: "[DIRECT] Who calls this function?", Required: []string{"symbol"}, Optional: []string{"limit"}},
			{Name: "get_callees", Description: "[DIRECT] What does this function call?", Required: []string{"symbol"}, Optional: []string{"limit"}},
			{Name: "get_symbols", Description: "List all functions/types, filter by kind, sort by size", Optional: []string{"query", "kind", "min_lines", "sort_by", "limit"}},
			{Name: "get_complexity", Description: "Cyclomatic complexity metrics for a file", Required: []string{"file"}, Optional: []string{"start_line", "end_line"}},
			{Name: "get_dead_code", Description: "Find unused functions/types that can be deleted", Optional: []string{"type", "limit"}},
			{Name: "get_entry_points", Description: "Find main(), HTTP handlers, CLI commands", Optional: []string{"type", "limit"}},
			{Name: "get_import_graph", Description: "See which packages import which", Optional: []string{"limit"}},
			{Name: "get_refactoring_candidates", Description: "Find long/complex functions needing refactor", Optional: []string{"min_lines", "max_complexity", "max_nesting", "limit"}},
		},
	},
	{
		Name:        "project",
		Description: "Project structure, file info, and indexing",
		Tools: []ToolInfo{
			{Name: "get_status", Description: "[DIRECT] Is codebase indexed? How many files?", Required: nil, Optional: nil},
			{Name: "get_file_summary", Description: "What's in this file? Imports, exports, functions, complexity", Required: []string{"file"}, Optional: []string{"quick"}},
			{Name: "get_project_tree", Description: "Directory structure as tree (json/text/markdown)", Optional: []string{"path", "depth", "include_files", "format"}},
			{Name: "index_codebase", Description: "Index/reindex the codebase for search", Optional: []string{"force", "ignore_patterns", "custom_extensions"}},
			{Name: "clear_index", Description: "Delete the entire index", Required: nil, Optional: nil},
		},
	},
	{
		Name:        "git",
		Description: "Git history, blame, and change tracking. NOTE: Run index_git_history first!",
		Tools: []ToolInfo{
			{Name: "search_history", Description: "Search commits by description - REQUIRES index_git_history first!", Required: []string{"query"}, Optional: []string{"time_from", "time_to", "authors", "paths", "limit"}},
			{Name: "get_blame", Description: "Who wrote each line? When?", Required: []string{"file"}, Optional: []string{"start_line", "end_line"}},
			{Name: "get_chunk_history", Description: "How has this function changed over time?", Optional: []string{"chunk_id", "symbol", "file", "limit"}},
			{Name: "get_code_evolution", Description: "Full history of a symbol's changes", Required: []string{"symbol"}, Optional: []string{"file"}},
			{Name: "find_regression", Description: "Find commit that introduced a bug (like git bisect)", Required: []string{"description"}, Optional: []string{"known_good", "known_bad", "paths", "limit"}},
			{Name: "get_commit_context", Description: "Detailed info about a specific commit", Required: []string{"commit"}, Optional: []string{"include_diff"}},
			{Name: "get_contributor_insights", Description: "Who knows this code best?", Optional: []string{"paths", "symbol"}},
			{Name: "index_git_history", Description: "Index git commits - REQUIRED before search_history!", Optional: []string{"since", "force"}},
			{Name: "get_git_history_status", Description: "Is git history indexed?", Required: nil, Optional: nil},
		},
	},
	{
		Name:        "memory",
		Description: "Additional memory operations (store/recall are direct)",
		Tools: []ToolInfo{
			{Name: "memory_store", Description: "[DIRECT] Store knowledge for future sessions", Required: []string{"content"}, Optional: []string{"category", "tags", "importance", "channel", "ttl_days"}},
			{Name: "memory_recall", Description: "[DIRECT] Search stored memories", Required: []string{"query"}, Optional: []string{"limit", "channel", "categories", "tags", "min_importance"}},
			{Name: "memory_forget", Description: "Delete a specific memory by ID", Required: []string{"id"}},
			{Name: "memory_checkpoint", Description: "Save snapshot of all memories", Required: []string{"name"}, Optional: []string{"description", "channel"}},
			{Name: "memory_restore", Description: "Restore memories from checkpoint", Required: []string{"checkpoint_id"}},
			{Name: "memory_stats", Description: "How many memories? By category?", Optional: []string{"channel"}},
			{Name: "memory_merge", Description: "Merge memories after git branch merge", Required: []string{"base_commit"}, Optional: []string{"strategy"}},
		},
	},
	{
		Name:        "todo",
		Description: "Additional task operations (create/list are direct)",
		Tools: []ToolInfo{
			{Name: "todo_create", Description: "[DIRECT] Create a tracked task", Required: []string{"title"}, Optional: []string{"description", "priority", "tags", "parent_id", "effort", "due_date", "channel"}},
			{Name: "todo_list", Description: "[DIRECT] List tasks by status/priority", Optional: []string{"statuses", "priorities", "channel", "include_completed", "limit"}},
			{Name: "todo_search", Description: "Search tasks by description", Required: []string{"query"}, Optional: []string{"channel", "include_children", "limit"}},
			{Name: "todo_update", Description: "Change task status, priority, title", Required: []string{"id"}, Optional: []string{"title", "description", "status", "priority", "progress"}},
			{Name: "todo_complete", Description: "Mark task as done", Required: []string{"id"}},
			{Name: "todo_delete", Description: "Remove a task", Required: []string{"id"}},
			{Name: "todo_stats", Description: "Task counts by status/priority", Optional: []string{"channel"}},
		},
	},
	{
		Name:        "notes",
		Description: "Formal notes, architecture decisions (ADR), and issue tracking",
		Tools: []ToolInfo{
			{Name: "note_add", Description: "Add a project note (more formal than memory)", Required: []string{"title", "content"}, Optional: []string{"tags", "file", "line_start", "line_end", "author"}},
			{Name: "note_list", Description: "List all notes, filter by tags/file", Optional: []string{"tags", "file", "author", "search", "limit"}},
			{Name: "note_get", Description: "Get note content by ID", Required: []string{"id"}},
			{Name: "note_update", Description: "Update note content", Required: []string{"id"}, Optional: []string{"title", "content", "tags"}},
			{Name: "note_delete", Description: "Delete a note", Required: []string{"id"}},
			{Name: "decision_add", Description: "Record architecture decision (ADR format)", Required: []string{"title", "context", "decision"}, Optional: []string{"alternatives", "consequences", "status"}},
			{Name: "decision_list", Description: "List architecture decisions", Optional: []string{"status", "search", "limit"}},
			{Name: "decision_get", Description: "Get decision details, optionally as markdown", Required: []string{"id"}, Optional: []string{"markdown"}},
			{Name: "decision_update", Description: "Update decision status/content", Required: []string{"id"}, Optional: []string{"decision", "consequences", "status", "superseded_by"}},
			{Name: "issue_add", Description: "Track a known bug/issue", Required: []string{"title", "description"}, Optional: []string{"severity", "file", "line"}},
			{Name: "issue_list", Description: "List tracked issues", Optional: []string{"severity", "status", "file", "search", "limit"}},
			{Name: "issue_resolve", Description: "Mark issue as resolved with explanation", Required: []string{"id", "resolution"}},
			{Name: "issue_reopen", Description: "Reopen a resolved issue", Required: []string{"id"}},
		},
	},
	{
		Name:        "config",
		Description: "Server configuration and setup",
		Tools: []ToolInfo{
			{Name: "get_config", Description: "Current server configuration", Required: nil, Optional: nil},
			{Name: "validate_config", Description: "Check if config is valid", Required: nil, Optional: nil},
			{Name: "detect_environment", Description: "Detect available AI providers (Ollama, etc.)", Required: nil, Optional: nil},
			{Name: "apply_recommendation", Description: "Apply config preset: primary, low_memory, high_quality, fast_index", Required: []string{"preset"}},
			{Name: "init_project", Description: "Initialize mcp-codewizard in this project", Optional: []string{"preset", "selections", "start_indexing"}},
			{Name: "get_project_context", Description: "Get project info for AI session start", Required: nil, Optional: nil},
		},
	},
	{
		Name:        "session",
		Description: "Session continuity across conversations",
		Tools: []ToolInfo{
			{Name: "session_set_context", Description: "Record what you're working on", Optional: []string{"task", "details", "files"}},
			{Name: "session_end", Description: "End session with summary for next time", Required: []string{"summary"}, Optional: []string{"next_steps", "open_issues"}},
			{Name: "session_get_history", Description: "See previous session summaries", Optional: []string{"limit"}},
		},
	},
	{
		Name:        "github",
		Description: "GitHub integration (requires gh CLI)",
		Tools: []ToolInfo{
			{Name: "github_sync", Description: "Import GitHub issues/PRs into memory for search", Optional: []string{"limit", "labels", "assignee", "prs", "all"}},
		},
	},
}

// toolHandlerFunc is the type for tool handler functions.
type toolHandlerFunc func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

// getToolHandlers returns a map of all tool handlers.
func (s *Server) getToolHandlers() map[string]toolHandlerFunc {
	return map[string]toolHandlerFunc{
		// Search
		"search_code":  s.handleSearchCode,
		"grep_code":    s.handleGrepCode,
		"fuzzy_search": s.handleFuzzySearch,
		"get_chunk":    s.handleGetChunk,
		// Analysis
		"get_callers":               s.handleGetCallers,
		"get_callees":               s.handleGetCallees,
		"get_symbols":               s.handleGetSymbols,
		"get_complexity":            s.handleGetComplexity,
		"get_dead_code":             s.handleGetDeadCode,
		"get_entry_points":          s.handleGetEntryPoints,
		"get_import_graph":          s.handleGetImportGraph,
		"get_refactoring_candidates": s.handleGetRefactoringCandidates,
		// Project
		"get_status":       s.handleGetStatus,
		"get_file_summary": s.handleGetFileSummary,
		"get_project_tree": s.handleGetProjectTree,
		"index_codebase":   s.handleIndexCodebase,
		"clear_index":      s.handleClearIndex,
		// Git
		"search_history":         s.handleSearchHistory,
		"get_blame":              s.handleGetBlame,
		"get_chunk_history":      s.handleGetChunkHistory,
		"get_code_evolution":     s.handleGetCodeEvolution,
		"find_regression":        s.handleFindRegression,
		"get_commit_context":     s.handleGetCommitContext,
		"get_contributor_insights": s.handleGetContributorInsights,
		"index_git_history":      s.handleIndexGitHistory,
		"get_git_history_status": s.handleGetGitHistoryStatus,
		// Memory
		"memory_store":      s.handleMemoryStore,
		"memory_recall":     s.handleMemoryRecall,
		"memory_forget":     s.handleMemoryForget,
		"memory_checkpoint": s.handleMemoryCheckpoint,
		"memory_restore":    s.handleMemoryRestore,
		"memory_stats":      s.handleMemoryStats,
		"memory_merge":      s.handleMemoryMerge,
		// Todo
		"todo_create":   s.handleTodoCreate,
		"todo_list":     s.handleTodoList,
		"todo_search":   s.handleTodoSearch,
		"todo_update":   s.handleTodoUpdate,
		"todo_complete": s.handleTodoComplete,
		"todo_delete":   s.handleTodoDelete,
		"todo_stats":    s.handleTodoStats,
		// Notes
		"note_add":        s.handleNoteAdd,
		"note_list":       s.handleNoteList,
		"note_get":        s.handleNoteGet,
		"note_update":     s.handleNoteUpdate,
		"note_delete":     s.handleNoteDelete,
		"decision_add":    s.handleDecisionAdd,
		"decision_list":   s.handleDecisionList,
		"decision_get":    s.handleDecisionGet,
		"decision_update": s.handleDecisionUpdate,
		"issue_add":       s.handleIssueAdd,
		"issue_list":      s.handleIssueList,
		"issue_resolve":   s.handleIssueResolve,
		"issue_reopen":    s.handleIssueReopen,
		// Config
		"get_config":           s.handleGetConfig,
		"validate_config":      s.handleValidateConfig,
		"detect_environment":   s.handleDetectEnvironment,
		"apply_recommendation": s.handleApplyRecommendation,
		"init_project":         s.handleInitProject,
		"get_project_context":  s.handleGetProjectContext,
		// Session
		"session_set_context": s.handleSessionSetContext,
		"session_end":         s.handleSessionEnd,
		"session_get_history": s.handleSessionGetHistory,
		// GitHub
		"github_sync": s.handleGitHubSync,
	}
}

// handleRoute routes a request to the appropriate tool handler.
func (s *Server) handleRoute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	toolName := req.GetString("tool", "")
	if toolName == "" {
		return mcp.NewToolResultError("required parameter 'tool' is missing"), nil
	}

	handlers := s.getToolHandlers()
	handler, ok := handlers[toolName]
	if !ok {
		// Find similar tool names for helpful error
		suggestions := s.findSimilarTools(toolName)
		if len(suggestions) > 0 {
			return mcp.NewToolResultError(fmt.Sprintf("unknown tool: %s. Did you mean: %s?", toolName, strings.Join(suggestions, ", "))), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("unknown tool: %s. Use list_tools to see available tools", toolName)), nil
	}

	// Extract params - try to get as map
	var params map[string]interface{}
	if argsMap, ok := req.Params.Arguments.(map[string]interface{}); ok {
		if p, ok := argsMap["params"].(map[string]interface{}); ok {
			params = p
		}
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	newReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: params,
		},
	}

	return handler(ctx, newReq)
}

// handleListTools returns available tools, optionally filtered by category.
func (s *Server) handleListTools(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	category := req.GetString("category", "")
	verbose := req.GetBool("verbose", false)

	var result interface{}

	if category != "" {
		// Return specific category
		for _, cat := range toolCategories {
			if cat.Name == category {
				result = cat
				break
			}
		}
		if result == nil {
			// List available categories
			cats := make([]string, len(toolCategories))
			for i, cat := range toolCategories {
				cats[i] = cat.Name
			}
			return nil, fmt.Errorf("unknown category: %s. Available: %s", category, strings.Join(cats, ", "))
		}
	} else if verbose {
		// Return all with full details
		result = toolCategories
	} else {
		// Return compact summary
		summary := make([]map[string]interface{}, len(toolCategories))
		for i, cat := range toolCategories {
			toolNames := make([]string, len(cat.Tools))
			for j, t := range cat.Tools {
				toolNames[j] = t.Name
			}
			summary[i] = map[string]interface{}{
				"category":    cat.Name,
				"description": cat.Description,
				"tools":       toolNames,
			}
		}
		result = summary
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleSuggestTool suggests the best tool for a given intent.
func (s *Server) handleSuggestTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	intent := req.GetString("intent", "")
	if intent == "" {
		return mcp.NewToolResultError("required parameter 'intent' is missing"), nil
	}

	intent = strings.ToLower(intent)

	// Tool keywords for matching
	toolKeywords := map[string][]string{
		"search_code":              {"search", "find", "look", "semantic", "meaning", "code"},
		"grep_code":                {"grep", "pattern", "regex", "text", "literal", "exact"},
		"fuzzy_search":             {"fuzzy", "typo", "approximate", "symbol", "function name"},
		"get_chunk":                {"chunk", "context", "snippet", "view"},
		"get_callers":              {"caller", "who calls", "usage", "reference", "used by"},
		"get_callees":              {"callee", "calls", "depends", "uses", "invokes"},
		"get_symbols":              {"symbol", "function", "type", "variable", "list"},
		"get_complexity":           {"complexity", "metric", "cyclomatic", "complicated"},
		"get_dead_code":            {"dead", "unused", "orphan", "unreachable"},
		"get_entry_points":         {"entry", "main", "handler", "endpoint", "start"},
		"get_import_graph":         {"import", "dependency", "graph", "module"},
		"get_refactoring_candidates": {"refactor", "improve", "clean", "long function"},
		"get_status":               {"status", "index", "stats", "info"},
		"get_file_summary":         {"file", "summary", "overview", "structure"},
		"get_project_tree":         {"tree", "directory", "structure", "folder"},
		"index_codebase":           {"index", "reindex", "build index"},
		"search_history":           {"history", "git", "commit", "change", "when"},
		"get_blame":                {"blame", "who wrote", "author", "line history"},
		"find_regression":          {"regression", "bug", "broke", "bisect"},
		"get_code_evolution":       {"evolution", "history of", "changed over time"},
		"memory_store":             {"remember", "store", "save", "note"},
		"memory_recall":            {"recall", "what did", "remember", "knowledge"},
		"todo_create":              {"todo", "task", "create task", "add todo"},
		"todo_list":                {"list tasks", "show todos", "pending"},
		"note_add":                 {"note", "document", "write down"},
		"decision_add":             {"decision", "adr", "architecture"},
		"issue_add":                {"issue", "bug", "problem", "track"},
		"get_config":               {"config", "configuration", "settings"},
		"github_sync":              {"github", "issue", "pr", "pull request"},
	}

	// Score each tool
	type toolScore struct {
		name        string
		score       int
		description string
	}

	var scores []toolScore
	for tool, keywords := range toolKeywords {
		score := 0
		for _, kw := range keywords {
			if strings.Contains(intent, kw) {
				score += len(kw) // Weight by keyword length
			}
		}
		if score > 0 {
			// Find description
			desc := ""
			for _, cat := range toolCategories {
				for _, t := range cat.Tools {
					if t.Name == tool {
						desc = t.Description
						break
					}
				}
			}
			scores = append(scores, toolScore{tool, score, desc})
		}
	}

	// Sort by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Build result
	if len(scores) == 0 {
		// No matches - suggest using list_tools
		return mcp.NewToolResultText(`{
  "suggestions": [],
  "hint": "No matching tools found. Use list_tools to see all available tools by category."
}`), nil
	}

	// Take top 3
	limit := 3
	if len(scores) < limit {
		limit = len(scores)
	}

	suggestions := make([]map[string]interface{}, limit)
	for i := 0; i < limit; i++ {
		// Find tool info
		var toolInfo *ToolInfo
		for _, cat := range toolCategories {
			for _, t := range cat.Tools {
				if t.Name == scores[i].name {
					toolInfo = &t
					break
				}
			}
		}

		suggestions[i] = map[string]interface{}{
			"tool":        scores[i].name,
			"confidence":  float64(scores[i].score) / 20.0, // Normalize
			"description": scores[i].description,
		}
		if toolInfo != nil {
			if len(toolInfo.Required) > 0 {
				suggestions[i]["required_params"] = toolInfo.Required
			}
			if len(toolInfo.Optional) > 0 {
				suggestions[i]["optional_params"] = toolInfo.Optional
			}
		}
	}

	result := map[string]interface{}{
		"suggestions": suggestions,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// findSimilarTools finds tools with similar names.
func (s *Server) findSimilarTools(name string) []string {
	var similar []string
	name = strings.ToLower(name)

	for _, cat := range toolCategories {
		for _, tool := range cat.Tools {
			toolLower := strings.ToLower(tool.Name)
			// Check for common substrings
			if strings.Contains(toolLower, name) || strings.Contains(name, toolLower) {
				similar = append(similar, tool.Name)
			} else if levenshteinDistance(name, toolLower) <= 3 {
				similar = append(similar, tool.Name)
			}
		}
	}

	if len(similar) > 3 {
		similar = similar[:3]
	}
	return similar
}

// levenshteinDistance calculates edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				min(matrix[i][j-1]+1, matrix[i-1][j-1]+cost),
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// registerRouterTools registers the router-only tools.
func (s *Server) registerRouterTools(mcpServer *server.MCPServer) {
	// Meta-router tool - for accessing 50+ additional tools
	mcpServer.AddTool(mcp.NewTool("route",
		mcp.WithDescription(`Access 50+ additional tools beyond the core set. ALWAYS call list_tools first to see available tools and required parameters.

Usage: route(tool="<name>", params={...})

Common examples:
• route(tool="find_regression", params={"description": "login broken after last deploy"})
• route(tool="get_file_summary", params={"file": "src/auth.go"})
• route(tool="search_history", params={"query": "authentication changes", "limit": 10})`),
		mcp.WithString("tool", mcp.Required(), mcp.Description("Tool name from list_tools output")),
		mcp.WithObject("params", mcp.Description("Tool parameters - check list_tools(verbose=true) for required/optional params")),
	), s.handleRoute)

	// Tool discovery - organized by USE CASE not just category
	mcpServer.AddTool(mcp.NewTool("list_tools",
		mcp.WithDescription(`Discover 50+ additional tools. Use when you need functionality beyond search_code, grep_code, get_callers, get_callees, memory_*, todo_*.

BY USE CASE:
• "Find symbol by approximate name" → fuzzy_search
• "Understand file structure" → get_file_summary, get_project_tree
• "Find code quality issues" → get_complexity, get_dead_code, get_refactoring_candidates
• "Analyze dependencies" → get_import_graph, get_entry_points
• "Search git history" → search_history (requires index_git_history first!)
• "Find who changed code" → get_blame, get_contributor_insights
• "Find when bug was introduced" → find_regression
• "Track architecture decisions" → decision_add, decision_list
• "Track known issues" → issue_add, issue_list

Call with verbose=true to see required parameters for each tool.`),
		mcp.WithString("category", mcp.Description("Filter by: search|analysis|project|git|memory|todo|notes|config|session|github")),
		mcp.WithBoolean("verbose", mcp.Description("Include required/optional parameters (recommended)")),
	), s.handleListTools)

	// Intent-based suggestion
	mcpServer.AddTool(mcp.NewTool("suggest_tool",
		mcp.WithDescription(`Don't know which tool to use? Describe your goal in natural language.

Examples:
• "find who wrote this code" → get_blame
• "find unused functions" → get_dead_code
• "see how this function evolved" → get_code_evolution
• "find functions that are too complex" → get_refactoring_candidates
• "what imports does this file use" → get_file_summary
• "sync github issues to memory" → github_sync`),
		mcp.WithString("intent", mcp.Required(), mcp.Description("What you want to accomplish in plain language")),
	), s.handleSuggestTool)
}

// hybridEssentialTools defines tools that are directly exposed in hybrid mode.
var hybridEssentialTools = []string{
	"search_code",
	"grep_code",
	"get_status",
	"get_chunk",
	"get_callers",
	"get_callees",
	"memory_store",
	"memory_recall",
	"todo_create",
	"todo_list",
}

// registerHybridTools registers essential tools directly + router for others.
func (s *Server) registerHybridTools(mcpServer *server.MCPServer) {
	// Register router tools first (for accessing additional tools)
	s.registerRouterTools(mcpServer)

	// ============================================
	// PRIMARY SEARCH - use these for code discovery
	// ============================================

	mcpServer.AddTool(mcp.NewTool("search_code",
		mcp.WithDescription(`Semantic code search - finds code by MEANING, not exact text.

WHEN TO USE:
• "find authentication logic" • "error handling" • "database connections" • "API endpoints"

WHEN TO USE grep_code INSTEAD:
• Exact function name: grep_code(pattern="handleLogin")
• Specific string: grep_code(pattern="TODO:")
• Import statement: grep_code(pattern="import.*fmt")

RETURNS: Chunks with {id, file, lines, code, score}. Use get_chunk(chunk_id) to see more context.

TIP: For typo-tolerant symbol search, use route(tool="fuzzy_search", params={"query": "..."})`),
		mcp.WithString("query", mcp.Required(), mcp.Description("Describe what you're looking for")),
		mcp.WithNumber("limit", mcp.Description("Results (default 10)")),
		mcp.WithBoolean("include_context", mcp.Description("Include surrounding lines")),
		mcp.WithArray("languages", mcp.Description("e.g. [\"go\", \"python\"]")),
	), s.handleSearchCode)

	mcpServer.AddTool(mcp.NewTool("grep_code",
		mcp.WithDescription(`Fast exact text/regex search - finds literal patterns.

WHEN TO USE:
• Function/variable name: "handleAuth", "userID"
• Error message: "connection refused"
• Import: "import.*http"
• TODO/FIXME comments
• Specific syntax: "func.*Error"

WHEN TO USE search_code INSTEAD:
• Conceptual: "authentication flow" → search_code
• Behavior: "how errors are handled" → search_code

RETURNS: Matches with {file, line, text, context}. Supports regex unless literal=true.`),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Exact text or regex")),
		mcp.WithString("path", mcp.Description("Glob filter: \"**/*.go\", \"src/**\"")),
		mcp.WithNumber("context_lines", mcp.Description("Lines around match (default 0)")),
		mcp.WithNumber("max_results", mcp.Description("Limit (default 100)")),
		mcp.WithBoolean("literal", mcp.Description("Disable regex, match literally")),
	), s.handleGrepCode)

	// ============================================
	// CODE NAVIGATION - understand code structure
	// ============================================

	mcpServer.AddTool(mcp.NewTool("get_chunk",
		mcp.WithDescription(`Get full code for a chunk ID. Use AFTER search_code or grep_code.

WORKFLOW:
1. search_code("authentication") → returns chunks with IDs
2. get_chunk(chunk_id="src/auth.go:42:abc123") → see full code with context

The chunk_id format is "filepath:line:hash" from search results.`),
		mcp.WithString("chunk_id", mcp.Required(), mcp.Description("ID from search results")),
		mcp.WithNumber("context_lines", mcp.Description("Extra lines (default 5)")),
	), s.handleGetChunk)

	mcpServer.AddTool(mcp.NewTool("get_callers",
		mcp.WithDescription(`Find all code that CALLS this function. "Who uses this?"

USE FOR:
• Impact analysis before refactoring
• Understanding how a function is used
• Finding entry points to functionality

RETURNS: List of {caller_function, file, line}

SEE ALSO: get_callees for the reverse - what does this function call?`),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Function name")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 50)")),
	), s.handleGetCallers)

	mcpServer.AddTool(mcp.NewTool("get_callees",
		mcp.WithDescription(`Find all functions CALLED BY this function. "What does this use?"

USE FOR:
• Understanding dependencies
• Tracing execution flow
• Finding what a function relies on

RETURNS: List of {callee_function, file, line}

SEE ALSO: get_callers for the reverse - who calls this function?`),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Function name")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 50)")),
	), s.handleGetCallees)

	// ============================================
	// INDEX STATUS - check before searching
	// ============================================

	mcpServer.AddTool(mcp.NewTool("get_status",
		mcp.WithDescription(`Check if codebase is indexed. RUN THIS FIRST if search returns no results.

RETURNS: {indexed: bool, files: N, chunks: N, last_indexed: timestamp}

IF NOT INDEXED: Use route(tool="index_codebase") to index.
IF OUTDATED: Use route(tool="index_codebase", params={"force": true}) to reindex.`),
	), s.handleGetStatus)

	// ============================================
	// PERSISTENT MEMORY - survives across sessions
	// ============================================

	mcpServer.AddTool(mcp.NewTool("memory_store",
		mcp.WithDescription(`Store knowledge that PERSISTS across sessions. Use liberally!

WHEN TO USE:
• User told you something important → memory_store(content="User prefers...", category="context")
• Made a decision → memory_store(content="Chose X because...", category="decision")
• Found important info → memory_store(content="Auth uses JWT...", category="fact")
• Noticed an issue → memory_store(content="Function X is slow...", category="error")

CATEGORIES: decision, context, fact, note, error, review

SEE ALSO for formal tracking:
• Architecture decisions → route(tool="decision_add", ...)
• Known bugs → route(tool="issue_add", ...)`),
		mcp.WithString("content", mcp.Required(), mcp.Description("What to remember")),
		mcp.WithString("category", mcp.Description("decision|context|fact|note|error|review")),
		mcp.WithArray("tags", mcp.Description("For retrieval: [\"auth\"]")),
		mcp.WithNumber("importance", mcp.Description("0.0-1.0, higher = more important")),
	), s.handleMemoryStore)

	mcpServer.AddTool(mcp.NewTool("memory_recall",
		mcp.WithDescription(`Search your stored memories. Use at SESSION START to restore context.

WHEN TO USE:
• Start of conversation: memory_recall(query="project context")
• Before making decisions: memory_recall(query="previous decisions about auth")
• Checking past findings: memory_recall(query="issues found")

RETURNS: Matching memories ranked by relevance.`),
		mcp.WithString("query", mcp.Required(), mcp.Description("What to search for")),
		mcp.WithNumber("limit", mcp.Description("Results (default 10)")),
		mcp.WithArray("categories", mcp.Description("Filter: [\"decision\"]")),
	), s.handleMemoryRecall)

	// ============================================
	// TASK TRACKING - persistent TODOs
	// ============================================

	mcpServer.AddTool(mcp.NewTool("todo_create",
		mcp.WithDescription(`Create a tracked task. PERSISTS across sessions.

USE FOR:
• "Fix this later" → todo_create(title="Fix auth bug", priority="high")
• "Remember to test" → todo_create(title="Test edge case X")
• "User requested" → todo_create(title="Add feature Y", priority="medium")

SEE ALSO:
• todo_list - see existing tasks
• route(tool="todo_complete", params={"id": "..."}) - mark done
• route(tool="todo_search", params={"query": "..."}) - search tasks`),
		mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
		mcp.WithString("description", mcp.Description("Details")),
		mcp.WithString("priority", mcp.Description("low|medium|high|urgent")),
	), s.handleTodoCreate)

	mcpServer.AddTool(mcp.NewTool("todo_list",
		mcp.WithDescription(`List tracked tasks. Check at SESSION START for pending work.

RETURNS: Tasks with {id, title, status, priority, created}

NEXT ACTIONS:
• Complete: route(tool="todo_complete", params={"id": "..."})
• Update: route(tool="todo_update", params={"id": "...", "status": "in_progress"})`),
		mcp.WithArray("statuses", mcp.Description("[\"pending\", \"in_progress\"]")),
		mcp.WithArray("priorities", mcp.Description("[\"high\", \"urgent\"]")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
	), s.handleTodoList)
}
