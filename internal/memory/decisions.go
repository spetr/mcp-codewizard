package memory

import (
	"fmt"
	"strings"
	"time"
)

// AddDecisionRequest contains parameters for adding a decision.
type AddDecisionRequest struct {
	Title        string   `json:"title"`
	Context      string   `json:"context"`      // Why we're deciding
	Decision     string   `json:"decision"`     // What we decided
	Consequences string   `json:"consequences"` // What this means
	Alternatives []string `json:"alternatives,omitempty"`
	Status       string   `json:"status,omitempty"` // proposed, accepted
	Author       string   `json:"author,omitempty"`
}

// AddDecision adds a new architecture decision.
func (s *Store) AddDecision(req AddDecisionRequest) (*Decision, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Context == "" {
		return nil, fmt.Errorf("context is required")
	}
	if req.Decision == "" {
		return nil, fmt.Errorf("decision is required")
	}

	status := DecisionStatus(req.Status)
	if status == "" {
		status = DecisionStatusProposed
	}

	now := time.Now()
	decision := Decision{
		Title:        req.Title,
		Context:      req.Context,
		Decision:     req.Decision,
		Consequences: req.Consequences,
		Alternatives: req.Alternatives,
		Status:       status,
		Author:       req.Author,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if decision.Author == "" {
		decision.Author = "ai"
	}

	// Generate ID
	decision.ID = decision.GenerateID()

	// Get current branch
	decision.Branch = s.getCurrentBranch()

	s.mu.Lock()
	s.decisions = append(s.decisions, decision)
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return nil, fmt.Errorf("failed to save decision: %w", err)
	}

	return &decision, nil
}

// GetDecision gets a decision by ID.
func (s *Store) GetDecision(id string) (*Decision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.decisions {
		if s.decisions[i].ID == id {
			return &s.decisions[i], nil
		}
	}

	return nil, fmt.Errorf("decision not found: %s", id)
}

// UpdateDecisionRequest contains parameters for updating a decision.
type UpdateDecisionRequest struct {
	ID           string   `json:"id"`
	Title        *string  `json:"title,omitempty"`
	Context      *string  `json:"context,omitempty"`
	Decision     *string  `json:"decision,omitempty"`
	Consequences *string  `json:"consequences,omitempty"`
	Alternatives []string `json:"alternatives,omitempty"`
	Status       *string  `json:"status,omitempty"`
	SupersededBy *string  `json:"superseded_by,omitempty"`
}

// UpdateDecision updates an existing decision.
func (s *Store) UpdateDecision(req UpdateDecisionRequest) (*Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var decision *Decision
	for i := range s.decisions {
		if s.decisions[i].ID == req.ID {
			decision = &s.decisions[i]
			break
		}
	}

	if decision == nil {
		return nil, fmt.Errorf("decision not found: %s", req.ID)
	}

	if req.Title != nil {
		decision.Title = *req.Title
	}
	if req.Context != nil {
		decision.Context = *req.Context
	}
	if req.Decision != nil {
		decision.Decision = *req.Decision
	}
	if req.Consequences != nil {
		decision.Consequences = *req.Consequences
	}
	if req.Alternatives != nil {
		decision.Alternatives = req.Alternatives
	}
	if req.Status != nil {
		decision.Status = DecisionStatus(*req.Status)
	}
	if req.SupersededBy != nil {
		decision.SupersededBy = *req.SupersededBy
		decision.Status = DecisionStatusSuperseded
	}

	decision.UpdatedAt = time.Now()

	// Save after releasing lock
	s.mu.Unlock()
	if err := s.Save(); err != nil {
		s.mu.Lock()
		return nil, fmt.Errorf("failed to save decision: %w", err)
	}

	s.mu.Lock()
	return decision, nil
}

// AcceptDecision marks a decision as accepted.
func (s *Store) AcceptDecision(id string) (*Decision, error) {
	status := string(DecisionStatusAccepted)
	return s.UpdateDecision(UpdateDecisionRequest{
		ID:     id,
		Status: &status,
	})
}

// DeprecateDecision marks a decision as deprecated.
func (s *Store) DeprecateDecision(id string) (*Decision, error) {
	status := string(DecisionStatusDeprecated)
	return s.UpdateDecision(UpdateDecisionRequest{
		ID:     id,
		Status: &status,
	})
}

// SupersedeDecision marks a decision as superseded by another.
func (s *Store) SupersedeDecision(id, supersededByID string) (*Decision, error) {
	return s.UpdateDecision(UpdateDecisionRequest{
		ID:           id,
		SupersededBy: &supersededByID,
	})
}

// DeleteDecision deletes a decision by ID.
func (s *Store) DeleteDecision(id string) error {
	s.mu.Lock()

	found := false
	for i := range s.decisions {
		if s.decisions[i].ID == id {
			s.decisions = append(s.decisions[:i], s.decisions[i+1:]...)
			found = true
			break
		}
	}
	s.mu.Unlock()

	if !found {
		return fmt.Errorf("decision not found: %s", id)
	}

	return s.Save()
}

// ListDecisionsRequest contains parameters for listing decisions.
type ListDecisionsRequest struct {
	Status []string `json:"status,omitempty"` // Filter by status
	Search string   `json:"search,omitempty"`
	Author string   `json:"author,omitempty"`
	Limit  int      `json:"limit,omitempty"`
	Offset int      `json:"offset,omitempty"`
}

// ListDecisions lists decisions with optional filters.
func (s *Store) ListDecisions(req ListDecisionsRequest) []Decision {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Decision

	for _, decision := range s.decisions {
		// Filter by status
		if len(req.Status) > 0 {
			hasStatus := false
			for _, status := range req.Status {
				if string(decision.Status) == status {
					hasStatus = true
					break
				}
			}
			if !hasStatus {
				continue
			}
		}

		// Filter by author
		if req.Author != "" && !strings.EqualFold(decision.Author, req.Author) {
			continue
		}

		// Search in title, context, decision
		if req.Search != "" {
			searchLower := strings.ToLower(req.Search)
			if !strings.Contains(strings.ToLower(decision.Title), searchLower) &&
				!strings.Contains(strings.ToLower(decision.Context), searchLower) &&
				!strings.Contains(strings.ToLower(decision.Decision), searchLower) {
				continue
			}
		}

		result = append(result, decision)
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

// GetActiveDecisions returns all non-deprecated, non-superseded decisions.
func (s *Store) GetActiveDecisions() []Decision {
	return s.ListDecisions(ListDecisionsRequest{
		Status: []string{string(DecisionStatusProposed), string(DecisionStatusAccepted)},
	})
}

// GetAllDecisions returns all decisions.
func (s *Store) GetAllDecisions() []Decision {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Decision, len(s.decisions))
	copy(result, s.decisions)
	return result
}

// CountDecisions returns the total number of decisions.
func (s *Store) CountDecisions() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.decisions)
}

// ExportDecisionAsMarkdown exports a decision as markdown (ADR format).
func (s *Store) ExportDecisionAsMarkdown(id string) (string, error) {
	decision, err := s.GetDecision(id)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", decision.Title))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", decision.Status))
	sb.WriteString(fmt.Sprintf("**Date:** %s\n", decision.CreatedAt.Format("2006-01-02")))
	if decision.Author != "" {
		sb.WriteString(fmt.Sprintf("**Author:** %s\n", decision.Author))
	}
	sb.WriteString("\n")

	sb.WriteString("## Context\n\n")
	sb.WriteString(decision.Context)
	sb.WriteString("\n\n")

	sb.WriteString("## Decision\n\n")
	sb.WriteString(decision.Decision)
	sb.WriteString("\n\n")

	if decision.Consequences != "" {
		sb.WriteString("## Consequences\n\n")
		sb.WriteString(decision.Consequences)
		sb.WriteString("\n\n")
	}

	if len(decision.Alternatives) > 0 {
		sb.WriteString("## Alternatives Considered\n\n")
		for _, alt := range decision.Alternatives {
			sb.WriteString(fmt.Sprintf("- %s\n", alt))
		}
		sb.WriteString("\n")
	}

	if decision.SupersededBy != "" {
		sb.WriteString(fmt.Sprintf("*Superseded by: %s*\n", decision.SupersededBy))
	}

	return sb.String(), nil
}
