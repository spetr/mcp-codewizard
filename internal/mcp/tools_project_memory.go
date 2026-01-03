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

	"github.com/spetr/mcp-codewizard/internal/github"
	"github.com/spetr/mcp-codewizard/internal/memory"
)

// registerProjectMemoryTools registers project memory and session tools.
func (s *Server) registerProjectMemoryTools(mcpServer *server.MCPServer) {
	// Session
	mcpServer.AddTool(mcp.NewTool("get_project_context",
		mcp.WithDescription("Session init context"),
	), s.handleGetProjectContext)

	// Notes
	mcpServer.AddTool(mcp.NewTool("note_add",
		mcp.WithDescription("Add note"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Title")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content")),
		mcp.WithArray("tags", mcp.Description("Tags")),
		mcp.WithString("file", mcp.Description("File path")),
		mcp.WithNumber("line_start", mcp.Description("Start line")),
		mcp.WithNumber("line_end", mcp.Description("End line")),
		mcp.WithString("author", mcp.Description("human|ai:claude|ai:gemini")),
	), s.handleNoteAdd)

	mcpServer.AddTool(mcp.NewTool("note_list",
		mcp.WithDescription("List notes"),
		mcp.WithArray("tags", mcp.Description("Tags")),
		mcp.WithString("file", mcp.Description("File")),
		mcp.WithString("search", mcp.Description("Search")),
		mcp.WithString("author", mcp.Description("Author")),
		mcp.WithNumber("limit", mcp.Description("Max results")),
	), s.handleNoteList)

	mcpServer.AddTool(mcp.NewTool("note_get",
		mcp.WithDescription("Get note"),
		mcp.WithString("id", mcp.Required(), mcp.Description("ID")),
	), s.handleNoteGet)

	mcpServer.AddTool(mcp.NewTool("note_update",
		mcp.WithDescription("Update note"),
		mcp.WithString("id", mcp.Required(), mcp.Description("ID")),
		mcp.WithString("title", mcp.Description("Title")),
		mcp.WithString("content", mcp.Description("Content")),
		mcp.WithArray("tags", mcp.Description("Tags")),
	), s.handleNoteUpdate)

	mcpServer.AddTool(mcp.NewTool("note_delete",
		mcp.WithDescription("Delete note"),
		mcp.WithString("id", mcp.Required(), mcp.Description("ID")),
	), s.handleNoteDelete)

	// Decisions (ADR)
	mcpServer.AddTool(mcp.NewTool("decision_add",
		mcp.WithDescription("Add decision (ADR)"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Title")),
		mcp.WithString("context", mcp.Required(), mcp.Description("Context")),
		mcp.WithString("decision", mcp.Required(), mcp.Description("Decision")),
		mcp.WithString("consequences", mcp.Description("Consequences")),
		mcp.WithArray("alternatives", mcp.Description("Alternatives")),
		mcp.WithString("status", mcp.Description("proposed|accepted")),
	), s.handleDecisionAdd)

	mcpServer.AddTool(mcp.NewTool("decision_list",
		mcp.WithDescription("List decisions"),
		mcp.WithArray("status", mcp.Description("proposed|accepted|deprecated|superseded")),
		mcp.WithString("search", mcp.Description("Search")),
		mcp.WithNumber("limit", mcp.Description("Max results")),
	), s.handleDecisionList)

	mcpServer.AddTool(mcp.NewTool("decision_get",
		mcp.WithDescription("Get decision"),
		mcp.WithString("id", mcp.Required(), mcp.Description("ID")),
		mcp.WithBoolean("markdown", mcp.Description("Export markdown")),
	), s.handleDecisionGet)

	mcpServer.AddTool(mcp.NewTool("decision_update",
		mcp.WithDescription("Update decision"),
		mcp.WithString("id", mcp.Required(), mcp.Description("ID")),
		mcp.WithString("status", mcp.Description("Status")),
		mcp.WithString("decision", mcp.Description("Decision")),
		mcp.WithString("consequences", mcp.Description("Consequences")),
		mcp.WithString("superseded_by", mcp.Description("Superseded by ID")),
	), s.handleDecisionUpdate)

	// Issues
	mcpServer.AddTool(mcp.NewTool("issue_add",
		mcp.WithDescription("Add issue"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Title")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Description")),
		mcp.WithString("severity", mcp.Description("critical|major|minor")),
		mcp.WithString("file", mcp.Description("File")),
		mcp.WithNumber("line", mcp.Description("Line")),
	), s.handleIssueAdd)

	mcpServer.AddTool(mcp.NewTool("issue_list",
		mcp.WithDescription("List issues"),
		mcp.WithArray("status", mcp.Description("open|investigating|resolved")),
		mcp.WithArray("severity", mcp.Description("critical|major|minor")),
		mcp.WithString("file", mcp.Description("File")),
		mcp.WithString("search", mcp.Description("Search")),
		mcp.WithNumber("limit", mcp.Description("Max results")),
	), s.handleIssueList)

	mcpServer.AddTool(mcp.NewTool("issue_resolve",
		mcp.WithDescription("Resolve issue"),
		mcp.WithString("id", mcp.Required(), mcp.Description("ID")),
		mcp.WithString("resolution", mcp.Required(), mcp.Description("Resolution")),
	), s.handleIssueResolve)

	mcpServer.AddTool(mcp.NewTool("issue_reopen",
		mcp.WithDescription("Reopen issue"),
		mcp.WithString("id", mcp.Required(), mcp.Description("ID")),
	), s.handleIssueReopen)

	mcpServer.AddTool(mcp.NewTool("github_sync",
		mcp.WithDescription("Sync from GitHub"),
		mcp.WithBoolean("all", mcp.Description("Include closed")),
		mcp.WithBoolean("prs", mcp.Description("Include PRs")),
		mcp.WithNumber("limit", mcp.Description("Max items")),
		mcp.WithArray("labels", mcp.Description("Labels")),
		mcp.WithString("assignee", mcp.Description("Assignee")),
	), s.handleGitHubSync)

	// Session
	mcpServer.AddTool(mcp.NewTool("session_set_context",
		mcp.WithDescription("Set session context"),
		mcp.WithString("task", mcp.Description("Task")),
		mcp.WithString("details", mcp.Description("Details")),
		mcp.WithArray("files", mcp.Description("Files")),
	), s.handleSessionSetContext)

	mcpServer.AddTool(mcp.NewTool("session_end",
		mcp.WithDescription("End session"),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Summary")),
		mcp.WithArray("next_steps", mcp.Description("Next steps")),
		mcp.WithArray("open_issues", mcp.Description("Open issues")),
	), s.handleSessionEnd)

	mcpServer.AddTool(mcp.NewTool("session_get_history",
		mcp.WithDescription("Session history"),
		mcp.WithNumber("limit", mcp.Description("Max sessions")),
	), s.handleSessionGetHistory)

	// Memory merge
	mcpServer.AddTool(mcp.NewTool("memory_merge",
		mcp.WithDescription("Merge memories (post-git-merge)"),
		mcp.WithString("base_commit", mcp.Required(), mcp.Description("Base commit")),
		mcp.WithString("strategy", mcp.Description("auto|ours|theirs")),
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

func (s *Server) handleGitHubSync(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store, err := s.getProjectMemoryStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory store: %v", err)), nil
	}

	// Initialize GitHub client
	client := github.New()
	if !client.IsAvailable(ctx) {
		return mcp.NewToolResultError("gh CLI not available or not authenticated. Run 'gh auth login' first."), nil
	}

	// Build sync config
	cfg := github.SyncConfig{
		Limit: req.GetInt("limit", 100),
	}

	if req.GetBool("all", false) {
		cfg.States = []string{"all"}
	} else {
		cfg.States = []string{"open"}
	}

	cfg.IncludePRs = req.GetBool("prs", false)
	cfg.Assignee = req.GetString("assignee", "")
	cfg.Labels = req.GetStringSlice("labels", nil)

	// Sync
	requests, result, err := client.Sync(ctx, cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("sync failed: %v", err)), nil
	}

	// Store issues
	stored := 0
	for _, req := range requests {
		if _, err := store.AddIssue(req); err == nil {
			stored++
		}
	}

	// Build response
	response := map[string]interface{}{
		"issues_synced": result.IssuesSynced,
		"prs_synced":    result.PRsSynced,
		"stored":        stored,
	}
	if len(result.Errors) > 0 {
		errStrings := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			errStrings[i] = e.Error()
		}
		response["errors"] = errStrings
	}

	jsonResult, _ := json.MarshalIndent(response, "", "  ")
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
