// Package mcp implements the MCP server for code search.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/spetr/mcp-codewizard/internal/memory"
)

// registerProjectMemoryTools registers project memory and session tools.
func (s *Server) registerProjectMemoryTools(mcpServer *server.MCPServer) {
	// get_project_context - Get initialization data for new session
	mcpServer.AddTool(mcp.NewTool("get_project_context",
		mcp.WithDescription("Get project context and initialization data for a new AI session. Returns project info, git state, index status, memory summary, and suggested actions. Call this at the start of each session."),
	), s.handleGetProjectContext)

	// === Notes ===

	// note_add - Add a new note
	mcpServer.AddTool(mcp.NewTool("note_add",
		mcp.WithDescription("Add a note about code, architecture, or findings. Notes are tracked in git."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Note title")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Note content (markdown)")),
		mcp.WithArray("tags", mcp.Description("Tags for organization")),
		mcp.WithString("file", mcp.Description("Related file path")),
		mcp.WithNumber("line_start", mcp.Description("Start line in file")),
		mcp.WithNumber("line_end", mcp.Description("End line in file")),
		mcp.WithString("author", mcp.Description("Author: 'human', 'ai:claude', 'ai:gemini'")),
	), s.handleNoteAdd)

	// note_list - List notes
	mcpServer.AddTool(mcp.NewTool("note_list",
		mcp.WithDescription("List notes with optional filters"),
		mcp.WithArray("tags", mcp.Description("Filter by tags")),
		mcp.WithString("file", mcp.Description("Filter by file path")),
		mcp.WithString("search", mcp.Description("Search in title and content")),
		mcp.WithString("author", mcp.Description("Filter by author")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 20)")),
	), s.handleNoteList)

	// note_get - Get a specific note
	mcpServer.AddTool(mcp.NewTool("note_get",
		mcp.WithDescription("Get a specific note by ID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note ID")),
	), s.handleNoteGet)

	// note_update - Update a note
	mcpServer.AddTool(mcp.NewTool("note_update",
		mcp.WithDescription("Update an existing note"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note ID")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("content", mcp.Description("New content")),
		mcp.WithArray("tags", mcp.Description("New tags")),
	), s.handleNoteUpdate)

	// note_delete - Delete a note
	mcpServer.AddTool(mcp.NewTool("note_delete",
		mcp.WithDescription("Delete a note"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note ID to delete")),
	), s.handleNoteDelete)

	// === Decisions (ADR) ===

	// decision_add - Add an architecture decision
	mcpServer.AddTool(mcp.NewTool("decision_add",
		mcp.WithDescription("Add an architecture decision record (ADR). Tracked in git."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Decision title")),
		mcp.WithString("context", mcp.Required(), mcp.Description("Why we're deciding (problem/situation)")),
		mcp.WithString("decision", mcp.Required(), mcp.Description("What we decided")),
		mcp.WithString("consequences", mcp.Description("What this means (tradeoffs)")),
		mcp.WithArray("alternatives", mcp.Description("Alternatives considered")),
		mcp.WithString("status", mcp.Description("Status: proposed, accepted (default: proposed)")),
	), s.handleDecisionAdd)

	// decision_list - List decisions
	mcpServer.AddTool(mcp.NewTool("decision_list",
		mcp.WithDescription("List architecture decisions"),
		mcp.WithArray("status", mcp.Description("Filter by status: proposed, accepted, deprecated, superseded")),
		mcp.WithString("search", mcp.Description("Search in title, context, decision")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 20)")),
	), s.handleDecisionList)

	// decision_get - Get a specific decision
	mcpServer.AddTool(mcp.NewTool("decision_get",
		mcp.WithDescription("Get a specific decision by ID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Decision ID")),
		mcp.WithBoolean("markdown", mcp.Description("Export as markdown (ADR format)")),
	), s.handleDecisionGet)

	// decision_update - Update a decision
	mcpServer.AddTool(mcp.NewTool("decision_update",
		mcp.WithDescription("Update an architecture decision"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Decision ID")),
		mcp.WithString("status", mcp.Description("New status: proposed, accepted, deprecated")),
		mcp.WithString("decision", mcp.Description("New decision text")),
		mcp.WithString("consequences", mcp.Description("New consequences")),
		mcp.WithString("superseded_by", mcp.Description("ID of decision that supersedes this one")),
	), s.handleDecisionUpdate)

	// === Issues ===

	// issue_add - Add a known issue
	mcpServer.AddTool(mcp.NewTool("issue_add",
		mcp.WithDescription("Add a known issue or bug to track. Tracked in git."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Issue title")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Issue description")),
		mcp.WithString("severity", mcp.Description("Severity: critical, major, minor (default: minor)")),
		mcp.WithString("file", mcp.Description("Related file path")),
		mcp.WithNumber("line", mcp.Description("Line number in file")),
	), s.handleIssueAdd)

	// issue_list - List issues
	mcpServer.AddTool(mcp.NewTool("issue_list",
		mcp.WithDescription("List known issues"),
		mcp.WithArray("status", mcp.Description("Filter by status: open, investigating, resolved")),
		mcp.WithArray("severity", mcp.Description("Filter by severity: critical, major, minor")),
		mcp.WithString("file", mcp.Description("Filter by file path")),
		mcp.WithString("search", mcp.Description("Search in title and description")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 20)")),
	), s.handleIssueList)

	// issue_resolve - Resolve an issue
	mcpServer.AddTool(mcp.NewTool("issue_resolve",
		mcp.WithDescription("Mark an issue as resolved"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Issue ID")),
		mcp.WithString("resolution", mcp.Required(), mcp.Description("How the issue was resolved")),
	), s.handleIssueResolve)

	// issue_reopen - Reopen an issue
	mcpServer.AddTool(mcp.NewTool("issue_reopen",
		mcp.WithDescription("Reopen a resolved issue"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Issue ID")),
	), s.handleIssueReopen)

	// === Session Management ===

	// session_set_context - Set working context
	mcpServer.AddTool(mcp.NewTool("session_set_context",
		mcp.WithDescription("Set the current working context for session continuity"),
		mcp.WithString("task", mcp.Description("Current task description")),
		mcp.WithString("details", mcp.Description("Task details")),
		mcp.WithArray("files", mcp.Description("Relevant files")),
	), s.handleSessionSetContext)

	// session_end - End session with summary
	mcpServer.AddTool(mcp.NewTool("session_end",
		mcp.WithDescription("End the current session and save a summary for the next session"),
		mcp.WithString("summary", mcp.Required(), mcp.Description("What was accomplished")),
		mcp.WithArray("next_steps", mcp.Description("Suggested next steps")),
		mcp.WithArray("open_issues", mcp.Description("Unresolved problems")),
	), s.handleSessionEnd)

	// session_get_history - Get session history
	mcpServer.AddTool(mcp.NewTool("session_get_history",
		mcp.WithDescription("Get previous session summaries"),
		mcp.WithNumber("limit", mcp.Description("Maximum sessions to return (default 5)")),
	), s.handleSessionGetHistory)

	// === Memory Merge ===

	// memory_merge - Perform three-way merge after git merge
	mcpServer.AddTool(mcp.NewTool("memory_merge",
		mcp.WithDescription("Perform three-way merge of memory after git merge. Call this after resolving git merge conflicts."),
		mcp.WithString("base_commit", mcp.Required(), mcp.Description("Base commit hash (merge base)")),
		mcp.WithString("strategy", mcp.Description("Merge strategy: auto, ours, theirs (default: auto)")),
	), s.handleMemoryMerge)
}

// Handler implementations

func (s *Server) handleGetProjectContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	// Get git integration
	git := memory.NewGitIntegration(store)

	// Build project info
	projectInfo := memory.ProjectInfo{
		ID:   git.GetProjectID(),
		Name: filepath.Base(s.projectDir),
		Path: s.projectDir,
	}

	// Get git state
	gitState := git.GetGitState()

	// Get index state
	indexState := s.getIndexState()

	// Get memory summary
	memorySummary := store.GetSummary()

	// Get last session
	var lastSession *memory.SessionSummary
	if history, err := store.GetSessionHistory(1); err == nil && len(history) > 0 {
		lastSession = &history[0]
	}

	// Generate suggested actions
	suggestedActions := s.generateSuggestedActions(indexState, gitState, memorySummary)

	// Build response
	initData := memory.ProjectInitData{
		Project:          projectInfo,
		Git:              gitState,
		Index:            indexState,
		Memory:           memorySummary,
		SuggestedActions: suggestedActions,
		LastSession:      lastSession,
	}

	jsonResult, _ := json.MarshalIndent(initData, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) getIndexState() memory.IndexState {
	state := memory.IndexState{
		Status: "unknown",
	}

	stats, err := s.store.GetStats()
	if err != nil {
		state.Status = "missing"
		return state
	}

	state.TotalFiles = stats.IndexedFiles
	state.TotalChunks = stats.TotalChunks
	state.TotalSymbols = stats.TotalSymbols
	state.LastIndexed = stats.LastIndexed
	state.IndexSize = formatBytes(stats.DBSizeBytes)

	if stats.TotalChunks == 0 {
		state.Status = "missing"
	} else if time.Since(stats.LastIndexed) > 24*time.Hour {
		state.Status = "stale"
	} else {
		state.Status = "ready"
	}

	return state
}

func (s *Server) generateSuggestedActions(index memory.IndexState, git memory.GitState, mem memory.MemorySummary) []memory.SuggestedAction {
	var actions []memory.SuggestedAction

	// Check if reindex is needed
	if index.Status == "missing" {
		actions = append(actions, memory.SuggestedAction{
			Priority:    "high",
			Type:        "reindex",
			Title:       "Index the codebase",
			Description: "No search index found. Run index_codebase to enable semantic search.",
			Tool:        "index_codebase",
		})
	} else if index.Status == "stale" {
		actions = append(actions, memory.SuggestedAction{
			Priority:    "medium",
			Type:        "reindex",
			Title:       "Update search index",
			Description: fmt.Sprintf("Index is %s old. Consider reindexing.", formatMemoryDuration(time.Since(index.LastIndexed))),
			Tool:        "index_codebase",
		})
	}

	// Check for merge conflicts
	if git.MergeInProgress && len(git.ConflictFiles) > 0 {
		actions = append(actions, memory.SuggestedAction{
			Priority:    "critical",
			Type:        "fix",
			Title:       "Resolve merge conflicts",
			Description: fmt.Sprintf("%d files have merge conflicts that need resolution.", len(git.ConflictFiles)),
		})
	}

	// Check for open critical issues
	if mem.OpenIssues > 0 {
		actions = append(actions, memory.SuggestedAction{
			Priority:    "high",
			Type:        "review",
			Title:       "Review open issues",
			Description: fmt.Sprintf("%d open issues need attention.", mem.OpenIssues),
			Tool:        "issue_list",
			Args:        map[string]any{"status": []string{"open"}},
		})
	}

	// Check for pending todos
	if mem.ActiveTodos > 0 {
		actions = append(actions, memory.SuggestedAction{
			Priority:    "medium",
			Type:        "info",
			Title:       "Pending tasks",
			Description: fmt.Sprintf("%d todos are pending.", mem.ActiveTodos),
			Tool:        "todo_list",
			Args:        map[string]any{"statuses": []string{"pending", "in_progress"}},
		})
	}

	// Check for dirty working directory
	if git.IsDirty && len(git.DirtyFiles) > 5 {
		actions = append(actions, memory.SuggestedAction{
			Priority:    "low",
			Type:        "info",
			Title:       "Uncommitted changes",
			Description: fmt.Sprintf("%d files have uncommitted changes.", len(git.DirtyFiles)),
		})
	}

	return actions
}

// === Notes handlers ===

func (s *Server) handleNoteAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	noteReq := memory.AddNoteRequest{
		Title:     req.GetString("title", ""),
		Content:   req.GetString("content", ""),
		Tags:      req.GetStringSlice("tags", nil),
		FilePath:  req.GetString("file", ""),
		LineStart: req.GetInt("line_start", 0),
		LineEnd:   req.GetInt("line_end", 0),
		Author:    req.GetString("author", "ai"),
	}

	note, err := store.AddNote(noteReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add note: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(note, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleNoteList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	listReq := memory.ListNotesRequest{
		Tags:     req.GetStringSlice("tags", nil),
		FilePath: req.GetString("file", ""),
		Search:   req.GetString("search", ""),
		Author:   req.GetString("author", ""),
		Limit:    req.GetInt("limit", 20),
	}

	notes := store.ListNotes(listReq)

	result := map[string]any{
		"total": len(notes),
		"notes": notes,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleNoteGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	note, err := store.GetNote(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("note not found: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(note, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleNoteUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	updateReq := memory.UpdateNoteRequest{ID: id}

	if title := req.GetString("title", ""); title != "" {
		updateReq.Title = &title
	}
	if content := req.GetString("content", ""); content != "" {
		updateReq.Content = &content
	}
	if tags := req.GetStringSlice("tags", nil); tags != nil {
		updateReq.Tags = tags
	}

	note, err := store.UpdateNote(updateReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update note: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(note, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleNoteDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	if err := store.DeleteNote(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete note: %v", err)), nil
	}

	return mcp.NewToolResultText(`{"success": true}`), nil
}

// === Decision handlers ===

func (s *Server) handleDecisionAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	decisionReq := memory.AddDecisionRequest{
		Title:        req.GetString("title", ""),
		Context:      req.GetString("context", ""),
		Decision:     req.GetString("decision", ""),
		Consequences: req.GetString("consequences", ""),
		Alternatives: req.GetStringSlice("alternatives", nil),
		Status:       req.GetString("status", "proposed"),
		Author:       "ai",
	}

	decision, err := store.AddDecision(decisionReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add decision: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(decision, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleDecisionList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	listReq := memory.ListDecisionsRequest{
		Status: req.GetStringSlice("status", nil),
		Search: req.GetString("search", ""),
		Limit:  req.GetInt("limit", 20),
	}

	decisions := store.ListDecisions(listReq)

	result := map[string]any{
		"total":     len(decisions),
		"decisions": decisions,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleDecisionGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	asMarkdown := req.GetBool("markdown", false)

	if asMarkdown {
		markdown, err := store.ExportDecisionAsMarkdown(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("decision not found: %v", err)), nil
		}
		return mcp.NewToolResultText(markdown), nil
	}

	decision, err := store.GetDecision(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("decision not found: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(decision, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleDecisionUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	updateReq := memory.UpdateDecisionRequest{ID: id}

	if status := req.GetString("status", ""); status != "" {
		updateReq.Status = &status
	}
	if decision := req.GetString("decision", ""); decision != "" {
		updateReq.Decision = &decision
	}
	if consequences := req.GetString("consequences", ""); consequences != "" {
		updateReq.Consequences = &consequences
	}
	if supersededBy := req.GetString("superseded_by", ""); supersededBy != "" {
		updateReq.SupersededBy = &supersededBy
	}

	decision, err := store.UpdateDecision(updateReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update decision: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(decision, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// === Issue handlers ===

func (s *Server) handleIssueAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	issueReq := memory.AddIssueRequest{
		Title:       req.GetString("title", ""),
		Description: req.GetString("description", ""),
		Severity:    req.GetString("severity", "minor"),
		FilePath:    req.GetString("file", ""),
		LineNumber:  req.GetInt("line", 0),
		Author:      "ai",
	}

	issue, err := store.AddIssue(issueReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add issue: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(issue, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleIssueList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	listReq := memory.ListIssuesRequest{
		Status:   req.GetStringSlice("status", nil),
		Severity: req.GetStringSlice("severity", nil),
		FilePath: req.GetString("file", ""),
		Search:   req.GetString("search", ""),
		Limit:    req.GetInt("limit", 20),
	}

	issues := store.ListIssues(listReq)

	result := map[string]any{
		"total":  len(issues),
		"issues": issues,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleIssueResolve(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	resolution := req.GetString("resolution", "")
	if resolution == "" {
		return mcp.NewToolResultError("resolution is required"), nil
	}

	issue, err := store.ResolveIssue(id, resolution)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to resolve issue: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(issue, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleIssueReopen(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	issue, err := store.ReopenIssue(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to reopen issue: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(issue, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// === Session handlers ===

func (s *Server) handleSessionSetContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	ctxReq := memory.SetWorkingContextRequest{
		CurrentTask:   req.GetString("task", ""),
		TaskDetails:   req.GetString("details", ""),
		RelevantFiles: req.GetStringSlice("files", nil),
	}

	if err := store.SetWorkingContext(ctxReq); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set context: %v", err)), nil
	}

	return mcp.NewToolResultText(`{"success": true}`), nil
}

func (s *Server) handleSessionEnd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	endReq := memory.EndSessionRequest{
		Summary:    req.GetString("summary", ""),
		NextSteps:  req.GetStringSlice("next_steps", nil),
		OpenIssues: req.GetStringSlice("open_issues", nil),
		Agent:      "ai",
	}

	if err := store.EndSession(endReq); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to end session: %v", err)), nil
	}

	return mcp.NewToolResultText(`{"success": true, "message": "Session ended and summary saved"}`), nil
}

func (s *Server) handleSessionGetHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	limit := req.GetInt("limit", 5)

	sessions, err := store.GetSessionHistory(limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get session history: %v", err)), nil
	}

	result := map[string]any{
		"total":    len(sessions),
		"sessions": sessions,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// === Memory Merge handler ===

func (s *Server) handleMemoryMerge(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	baseCommit := req.GetString("base_commit", "")
	if baseCommit == "" {
		return mcp.NewToolResultError("base_commit is required"), nil
	}

	strategyStr := req.GetString("strategy", "auto")
	var strategy memory.MergeStrategy
	switch strategyStr {
	case "ours":
		strategy = memory.MergeStrategyOurs
	case "theirs":
		strategy = memory.MergeStrategyTheirs
	default:
		strategy = memory.MergeStrategyAuto
	}

	merger := memory.NewMerger(store, strategy)
	result, err := merger.MergeFromGit(baseCommit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("merge failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Helper functions

func (s *Server) getProjectMemoryStore() (*memory.Store, error) {
	return memory.NewStore(memory.StoreConfig{
		ProjectDir: s.projectDir,
	})
}

func formatMemoryDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}
