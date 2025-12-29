package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SetWorkingContextRequest contains parameters for setting working context.
type SetWorkingContextRequest struct {
	CurrentTask   string   `json:"current_task,omitempty"`
	TaskDetails   string   `json:"task_details,omitempty"`
	RelevantFiles []string `json:"relevant_files,omitempty"`
}

// SetWorkingContext sets the current working context.
func (s *Store) SetWorkingContext(req SetWorkingContextRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.CurrentTask != "" {
		s.context.CurrentTask = req.CurrentTask
	}
	if req.TaskDetails != "" {
		s.context.TaskDetails = req.TaskDetails
	}
	if req.RelevantFiles != nil {
		s.context.RelevantFiles = req.RelevantFiles
	}
	s.context.UpdatedAt = time.Now()

	// Save context
	return s.saveContextLocked()
}

// AddRelevantFile adds a file to the relevant files list.
func (s *Store) AddRelevantFile(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, f := range s.context.RelevantFiles {
		if f == filePath {
			return nil
		}
	}

	s.context.RelevantFiles = append(s.context.RelevantFiles, filePath)
	s.context.UpdatedAt = time.Now()

	return s.saveContextLocked()
}

// RemoveRelevantFile removes a file from the relevant files list.
func (s *Store) RemoveRelevantFile(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, f := range s.context.RelevantFiles {
		if f == filePath {
			s.context.RelevantFiles = append(
				s.context.RelevantFiles[:i],
				s.context.RelevantFiles[i+1:]...,
			)
			s.context.UpdatedAt = time.Now()
			return s.saveContextLocked()
		}
	}

	return nil
}

// ClearRelevantFiles clears the relevant files list.
func (s *Store) ClearRelevantFiles() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.context.RelevantFiles = nil
	s.context.UpdatedAt = time.Now()

	return s.saveContextLocked()
}

// SetPRInProgress sets the current PR in progress.
func (s *Store) SetPRInProgress(pr *PRInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.context.PRInProgress = pr
	s.context.UpdatedAt = time.Now()

	return s.saveContextLocked()
}

// ClearPRInProgress clears the PR in progress.
func (s *Store) ClearPRInProgress() error {
	return s.SetPRInProgress(nil)
}

// EndSessionRequest contains parameters for ending a session.
type EndSessionRequest struct {
	Summary    string   `json:"summary"`
	NextSteps  []string `json:"next_steps,omitempty"`
	OpenIssues []string `json:"open_issues,omitempty"`
	Agent      string   `json:"agent,omitempty"`
}

// EndSession ends the current session and saves a summary.
func (s *Store) EndSession(req EndSessionRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := SessionSummary{
		EndedAt:    time.Now(),
		Summary:    req.Summary,
		NextSteps:  req.NextSteps,
		OpenIssues: req.OpenIssues,
		Agent:      req.Agent,
	}

	s.context.LastSession = &session
	s.context.UpdatedAt = time.Now()

	// Also save to session history
	if err := s.appendSessionHistory(session); err != nil {
		// Non-fatal error, just log
		fmt.Printf("Warning: failed to save session history: %v\n", err)
	}

	return s.saveContextLocked()
}

// appendSessionHistory appends a session to the history file.
func (s *Store) appendSessionHistory(session SessionSummary) error {
	historyPath := filepath.Join(s.localDir, "session-history.jsonl")
	return appendJSONL(historyPath, session)
}

// GetSessionHistory returns session history.
func (s *Store) GetSessionHistory(limit int) ([]SessionSummary, error) {
	historyPath := filepath.Join(s.localDir, "session-history.jsonl")
	sessions, err := loadJSONL[SessionSummary](historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Return most recent first
	result := make([]SessionSummary, 0, len(sessions))
	for i := len(sessions) - 1; i >= 0; i-- {
		result = append(result, sessions[i])
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result, nil
}

// saveContextLocked saves the context (must be called with lock held).
func (s *Store) saveContextLocked() error {
	contextPath := filepath.Join(s.localDir, "context.json")
	data, err := json.MarshalIndent(s.context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}
	if err := os.WriteFile(contextPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}
	return nil
}

// UpdateBranch updates the active branch in context.
func (s *Store) UpdateBranch(branch string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.context.ActiveBranch = branch
	s.context.UpdatedAt = time.Now()

	return s.saveContextLocked()
}

// OnBranchSwitch handles branch switch event.
func (s *Store) OnBranchSwitch(fromBranch, toBranch string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Save current context as part of history
	if s.context.CurrentTask != "" {
		// Create a mini session summary for the branch switch
		session := SessionSummary{
			EndedAt: time.Now(),
			Summary: fmt.Sprintf("Branch switch: %s -> %s. Task: %s",
				fromBranch, toBranch, s.context.CurrentTask),
			Agent: "system",
		}
		_ = s.appendSessionHistory(session)
	}

	// Update branch
	s.context.ActiveBranch = toBranch
	s.context.UpdatedAt = time.Now()

	return s.saveContextLocked()
}

// GetWorkingContext returns the current working context.
func (s *Store) GetWorkingContext() WorkingContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context
}

// HasActiveTask returns true if there's an active task.
func (s *Store) HasActiveTask() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context.CurrentTask != ""
}

// ClearWorkingContext clears the current working context.
func (s *Store) ClearWorkingContext() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.context.CurrentTask = ""
	s.context.TaskDetails = ""
	s.context.RelevantFiles = nil
	s.context.PRInProgress = nil
	s.context.UpdatedAt = time.Now()

	return s.saveContextLocked()
}

// GetContextSummary returns a text summary of the current context.
func (s *Store) GetContextSummary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.context.CurrentTask == "" && s.context.LastSession == nil {
		return "No active context."
	}

	summary := ""

	if s.context.CurrentTask != "" {
		summary += fmt.Sprintf("Current task: %s\n", s.context.CurrentTask)
		if s.context.TaskDetails != "" {
			summary += fmt.Sprintf("Details: %s\n", s.context.TaskDetails)
		}
		if len(s.context.RelevantFiles) > 0 {
			summary += fmt.Sprintf("Relevant files: %v\n", s.context.RelevantFiles)
		}
	}

	if s.context.PRInProgress != nil {
		summary += fmt.Sprintf("PR in progress: #%d %s\n",
			s.context.PRInProgress.Number,
			s.context.PRInProgress.Title)
	}

	if s.context.LastSession != nil {
		summary += fmt.Sprintf("\nLast session (%s):\n",
			s.context.LastSession.EndedAt.Format("2006-01-02 15:04"))
		summary += fmt.Sprintf("  Summary: %s\n", s.context.LastSession.Summary)
		if len(s.context.LastSession.NextSteps) > 0 {
			summary += "  Next steps:\n"
			for _, step := range s.context.LastSession.NextSteps {
				summary += fmt.Sprintf("    - %s\n", step)
			}
		}
		if len(s.context.LastSession.OpenIssues) > 0 {
			summary += "  Open issues:\n"
			for _, issue := range s.context.LastSession.OpenIssues {
				summary += fmt.Sprintf("    - %s\n", issue)
			}
		}
	}

	return summary
}
