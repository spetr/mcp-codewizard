package memory

import (
	"fmt"
	"strings"
	"time"
)

// AddIssueRequest contains parameters for adding an issue.
type AddIssueRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // critical, major, minor
	FilePath    string `json:"file,omitempty"`
	LineNumber  int    `json:"line,omitempty"`
	Author      string `json:"author,omitempty"`
}

// AddIssue adds a new known issue.
func (s *Store) AddIssue(req AddIssueRequest) (*Issue, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	severity := req.Severity
	if severity == "" {
		severity = "minor"
	}

	now := time.Now()
	issue := Issue{
		Title:       req.Title,
		Description: req.Description,
		Severity:    severity,
		FilePath:    req.FilePath,
		LineNumber:  req.LineNumber,
		Status:      "open",
		Author:      req.Author,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if issue.Author == "" {
		issue.Author = "ai"
	}

	// Generate ID
	issue.ID = issue.GenerateID()

	// Get current branch
	issue.Branch = s.getCurrentBranch()

	s.mu.Lock()
	s.issues = append(s.issues, issue)
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return nil, fmt.Errorf("failed to save issue: %w", err)
	}

	return &issue, nil
}

// GetIssue gets an issue by ID.
func (s *Store) GetIssue(id string) (*Issue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.issues {
		if s.issues[i].ID == id {
			return &s.issues[i], nil
		}
	}

	return nil, fmt.Errorf("issue not found: %s", id)
}

// UpdateIssueRequest contains parameters for updating an issue.
type UpdateIssueRequest struct {
	ID          string  `json:"id"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Severity    *string `json:"severity,omitempty"`
	Status      *string `json:"status,omitempty"`
	Resolution  *string `json:"resolution,omitempty"`
	FilePath    *string `json:"file,omitempty"`
	LineNumber  *int    `json:"line,omitempty"`
}

// UpdateIssue updates an existing issue.
func (s *Store) UpdateIssue(req UpdateIssueRequest) (*Issue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var issue *Issue
	for i := range s.issues {
		if s.issues[i].ID == req.ID {
			issue = &s.issues[i]
			break
		}
	}

	if issue == nil {
		return nil, fmt.Errorf("issue not found: %s", req.ID)
	}

	if req.Title != nil {
		issue.Title = *req.Title
	}
	if req.Description != nil {
		issue.Description = *req.Description
	}
	if req.Severity != nil {
		issue.Severity = *req.Severity
	}
	if req.Status != nil {
		oldStatus := issue.Status
		issue.Status = *req.Status

		// Set resolved time if transitioning to resolved
		if *req.Status == "resolved" && oldStatus != "resolved" {
			now := time.Now()
			issue.ResolvedAt = &now
		} else if *req.Status != "resolved" {
			issue.ResolvedAt = nil
		}
	}
	if req.Resolution != nil {
		issue.Resolution = *req.Resolution
	}
	if req.FilePath != nil {
		issue.FilePath = *req.FilePath
	}
	if req.LineNumber != nil {
		issue.LineNumber = *req.LineNumber
	}

	issue.UpdatedAt = time.Now()

	// Save after releasing lock
	s.mu.Unlock()
	if err := s.Save(); err != nil {
		s.mu.Lock()
		return nil, fmt.Errorf("failed to save issue: %w", err)
	}

	s.mu.Lock()
	return issue, nil
}

// ResolveIssue marks an issue as resolved.
func (s *Store) ResolveIssue(id, resolution string) (*Issue, error) {
	status := "resolved"
	return s.UpdateIssue(UpdateIssueRequest{
		ID:         id,
		Status:     &status,
		Resolution: &resolution,
	})
}

// ReopenIssue reopens a resolved issue.
func (s *Store) ReopenIssue(id string) (*Issue, error) {
	status := "open"
	return s.UpdateIssue(UpdateIssueRequest{
		ID:     id,
		Status: &status,
	})
}

// DeleteIssue deletes an issue by ID.
func (s *Store) DeleteIssue(id string) error {
	s.mu.Lock()

	found := false
	for i := range s.issues {
		if s.issues[i].ID == id {
			s.issues = append(s.issues[:i], s.issues[i+1:]...)
			found = true
			break
		}
	}
	s.mu.Unlock()

	if !found {
		return fmt.Errorf("issue not found: %s", id)
	}

	return s.Save()
}

// ListIssuesRequest contains parameters for listing issues.
type ListIssuesRequest struct {
	Status   []string `json:"status,omitempty"`
	Severity []string `json:"severity,omitempty"`
	FilePath string   `json:"file,omitempty"`
	Search   string   `json:"search,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}

// ListIssues lists issues with optional filters.
func (s *Store) ListIssues(req ListIssuesRequest) []Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Issue

	for _, issue := range s.issues {
		// Filter by status
		if len(req.Status) > 0 {
			hasStatus := false
			for _, status := range req.Status {
				if issue.Status == status {
					hasStatus = true
					break
				}
			}
			if !hasStatus {
				continue
			}
		}

		// Filter by severity
		if len(req.Severity) > 0 {
			hasSeverity := false
			for _, severity := range req.Severity {
				if issue.Severity == severity {
					hasSeverity = true
					break
				}
			}
			if !hasSeverity {
				continue
			}
		}

		// Filter by file path
		if req.FilePath != "" && !strings.Contains(issue.FilePath, req.FilePath) {
			continue
		}

		// Search in title and description
		if req.Search != "" {
			searchLower := strings.ToLower(req.Search)
			if !strings.Contains(strings.ToLower(issue.Title), searchLower) &&
				!strings.Contains(strings.ToLower(issue.Description), searchLower) {
				continue
			}
		}

		result = append(result, issue)
	}

	// Apply offset
	if req.Offset > 0 {
		if req.Offset >= len(result) {
			return nil
		}
		result = result[req.Offset:]
	}

	// Apply limit
	if req.Limit > 0 && req.Limit < len(result) {
		result = result[:req.Limit]
	}

	return result
}

// GetOpenIssues returns all open and investigating issues.
func (s *Store) GetOpenIssues() []Issue {
	return s.ListIssues(ListIssuesRequest{
		Status: []string{"open", "investigating"},
	})
}

// GetCriticalIssues returns all critical open issues.
func (s *Store) GetCriticalIssues() []Issue {
	return s.ListIssues(ListIssuesRequest{
		Status:   []string{"open", "investigating"},
		Severity: []string{"critical"},
	})
}

// GetIssuesByFile returns all issues associated with a file.
func (s *Store) GetIssuesByFile(filePath string) []Issue {
	return s.ListIssues(ListIssuesRequest{FilePath: filePath})
}

// GetAllIssues returns all issues.
func (s *Store) GetAllIssues() []Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Issue, len(s.issues))
	copy(result, s.issues)
	return result
}

// CountIssues returns the total number of issues.
func (s *Store) CountIssues() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.issues)
}

// CountOpenIssues returns the number of open issues.
func (s *Store) CountOpenIssues() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, issue := range s.issues {
		if issue.Status == "open" || issue.Status == "investigating" {
			count++
		}
	}
	return count
}
