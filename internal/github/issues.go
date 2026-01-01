// Package github provides GitHub integration using the gh CLI.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/spetr/mcp-codewizard/internal/memory"
)

// Issue represents a GitHub issue.
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	Labels    []Label   `json:"labels"`
	Assignees []User    `json:"assignees"`
	Author    User      `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ClosedAt  time.Time `json:"closedAt"`
	URL       string    `json:"url"`
	Comments  int       `json:"comments"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
}

// Label represents a GitHub label.
type Label struct {
	Name string `json:"name"`
}

// User represents a GitHub user.
type User struct {
	Login string `json:"login"`
}

// SyncConfig contains configuration for GitHub sync.
type SyncConfig struct {
	States     []string // open, closed, all
	Limit      int      // max issues to fetch
	Labels     []string // filter by labels
	Assignee   string   // filter by assignee
	IncludePRs bool     // include pull requests
}

// Client provides GitHub operations using gh CLI.
type Client struct{}

// New creates a new GitHub client.
func New() *Client {
	return &Client{}
}

// IsAvailable checks if gh CLI is available and authenticated.
func (c *Client) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "gh", "auth", "status")
	return cmd.Run() == nil
}

// GetRepoInfo returns the current repository info (owner/repo).
func (c *Client) GetRepoInfo(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repo info: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// FetchIssues fetches issues from the current repository.
func (c *Client) FetchIssues(ctx context.Context, cfg SyncConfig) ([]Issue, error) {
	args := []string{"issue", "list", "--json",
		"number,title,body,state,labels,assignees,author,createdAt,updatedAt,closedAt,url,comments,milestone",
		"--limit", fmt.Sprintf("%d", cfg.Limit),
	}

	// Add state filter
	if len(cfg.States) > 0 {
		args = append(args, "--state", cfg.States[0])
	}

	// Add label filter
	for _, label := range cfg.Labels {
		args = append(args, "--label", label)
	}

	// Add assignee filter
	if cfg.Assignee != "" {
		args = append(args, "--assignee", cfg.Assignee)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh issue list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// FetchPullRequests fetches pull requests from the current repository.
func (c *Client) FetchPullRequests(ctx context.Context, cfg SyncConfig) ([]Issue, error) {
	args := []string{"pr", "list", "--json",
		"number,title,body,state,labels,assignees,author,createdAt,updatedAt,closedAt,url,comments",
		"--limit", fmt.Sprintf("%d", cfg.Limit),
	}

	// Add state filter
	if len(cfg.States) > 0 {
		state := cfg.States[0]
		if state == "open" || state == "closed" || state == "merged" || state == "all" {
			args = append(args, "--state", state)
		}
	}

	// Add label filter
	for _, label := range cfg.Labels {
		args = append(args, "--label", label)
	}

	// Add assignee filter
	if cfg.Assignee != "" {
		args = append(args, "--assignee", cfg.Assignee)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh pr list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to fetch PRs: %w", err)
	}

	var prs []Issue
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PRs: %w", err)
	}

	return prs, nil
}

// IssueToMemoryRequest converts a GitHub issue to a memory.AddIssueRequest.
func IssueToMemoryRequest(issue Issue, repo string, isPR bool) memory.AddIssueRequest {
	// Build description with GitHub details
	var desc strings.Builder

	typeLabel := "Issue"
	if isPR {
		typeLabel = "Pull Request"
	}

	desc.WriteString(fmt.Sprintf("GitHub %s #%d\n", typeLabel, issue.Number))
	desc.WriteString(fmt.Sprintf("URL: %s\n", issue.URL))

	if len(issue.Labels) > 0 {
		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.Name
		}
		desc.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(labels, ", ")))
	}

	if len(issue.Assignees) > 0 {
		assignees := make([]string, len(issue.Assignees))
		for i, a := range issue.Assignees {
			assignees[i] = a.Login
		}
		desc.WriteString(fmt.Sprintf("Assignees: %s\n", strings.Join(assignees, ", ")))
	}

	if issue.Milestone != nil && issue.Milestone.Title != "" {
		desc.WriteString(fmt.Sprintf("Milestone: %s\n", issue.Milestone.Title))
	}

	desc.WriteString(fmt.Sprintf("Author: %s\n", issue.Author.Login))
	desc.WriteString(fmt.Sprintf("Comments: %d\n", issue.Comments))

	if issue.Body != "" {
		desc.WriteString(fmt.Sprintf("\n---\n%s", issue.Body))
	}

	// Determine severity from labels
	severity := "minor"
	for _, l := range issue.Labels {
		name := strings.ToLower(l.Name)
		if strings.Contains(name, "critical") || strings.Contains(name, "urgent") || strings.Contains(name, "p0") {
			severity = "critical"
			break
		}
		if strings.Contains(name, "major") || strings.Contains(name, "high") || strings.Contains(name, "p1") {
			severity = "major"
		}
	}

	// Build title with number prefix
	title := fmt.Sprintf("[GitHub #%d] %s", issue.Number, issue.Title)
	if isPR {
		title = fmt.Sprintf("[PR #%d] %s", issue.Number, issue.Title)
	}

	return memory.AddIssueRequest{
		Title:       title,
		Description: desc.String(),
		Severity:    severity,
		Author:      "github:" + issue.Author.Login,
	}
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	IssuesSynced int
	PRsSynced    int
	Errors       []error
}

// Sync fetches issues and PRs and converts them to memory requests.
func (c *Client) Sync(ctx context.Context, cfg SyncConfig) ([]memory.AddIssueRequest, SyncResult, error) {
	result := SyncResult{}

	// Get repo info
	repo, err := c.GetRepoInfo(ctx)
	if err != nil {
		return nil, result, err
	}

	var requests []memory.AddIssueRequest

	// Fetch issues
	issues, err := c.FetchIssues(ctx, cfg)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("issues: %w", err))
		slog.Warn("failed to fetch issues", "error", err)
	} else {
		for _, issue := range issues {
			requests = append(requests, IssueToMemoryRequest(issue, repo, false))
			result.IssuesSynced++
		}
	}

	// Fetch PRs if requested
	if cfg.IncludePRs {
		prs, err := c.FetchPullRequests(ctx, cfg)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("prs: %w", err))
			slog.Warn("failed to fetch PRs", "error", err)
		} else {
			for _, pr := range prs {
				requests = append(requests, IssueToMemoryRequest(pr, repo, true))
				result.PRsSynced++
			}
		}
	}

	return requests, result, nil
}
