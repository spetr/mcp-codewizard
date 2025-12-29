package memory

import (
	"fmt"
	"strings"
	"time"
)

// AddNoteRequest contains parameters for adding a note.
type AddNoteRequest struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags,omitempty"`
	FilePath  string   `json:"file,omitempty"`
	LineStart int      `json:"line_start,omitempty"`
	LineEnd   int      `json:"line_end,omitempty"`
	Author    string   `json:"author,omitempty"`
}

// AddNote adds a new note.
func (s *Store) AddNote(req AddNoteRequest) (*Note, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	now := time.Now()
	note := Note{
		Title:     req.Title,
		Content:   req.Content,
		Tags:      req.Tags,
		FilePath:  req.FilePath,
		LineStart: req.LineStart,
		LineEnd:   req.LineEnd,
		Author:    req.Author,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if note.Author == "" {
		note.Author = "ai"
	}

	// Generate ID
	note.ID = note.GenerateID()

	// Get current branch
	note.Branch = s.getCurrentBranch()

	s.mu.Lock()
	s.notes = append(s.notes, note)
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return nil, fmt.Errorf("failed to save note: %w", err)
	}

	return &note, nil
}

// GetNote gets a note by ID.
func (s *Store) GetNote(id string) (*Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.notes {
		if s.notes[i].ID == id {
			return &s.notes[i], nil
		}
	}

	return nil, fmt.Errorf("note not found: %s", id)
}

// UpdateNoteRequest contains parameters for updating a note.
type UpdateNoteRequest struct {
	ID        string   `json:"id"`
	Title     *string  `json:"title,omitempty"`
	Content   *string  `json:"content,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	FilePath  *string  `json:"file,omitempty"`
	LineStart *int     `json:"line_start,omitempty"`
	LineEnd   *int     `json:"line_end,omitempty"`
}

// UpdateNote updates an existing note.
func (s *Store) UpdateNote(req UpdateNoteRequest) (*Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var note *Note
	for i := range s.notes {
		if s.notes[i].ID == req.ID {
			note = &s.notes[i]
			break
		}
	}

	if note == nil {
		return nil, fmt.Errorf("note not found: %s", req.ID)
	}

	if req.Title != nil {
		note.Title = *req.Title
	}
	if req.Content != nil {
		note.Content = *req.Content
	}
	if req.Tags != nil {
		note.Tags = req.Tags
	}
	if req.FilePath != nil {
		note.FilePath = *req.FilePath
	}
	if req.LineStart != nil {
		note.LineStart = *req.LineStart
	}
	if req.LineEnd != nil {
		note.LineEnd = *req.LineEnd
	}

	note.UpdatedAt = time.Now()

	// Save after releasing lock
	notesCopy := make([]Note, len(s.notes))
	copy(notesCopy, s.notes)
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		s.mu.Lock()
		return nil, fmt.Errorf("failed to save note: %w", err)
	}

	s.mu.Lock()
	return note, nil
}

// DeleteNote deletes a note by ID.
func (s *Store) DeleteNote(id string) error {
	s.mu.Lock()

	found := false
	for i := range s.notes {
		if s.notes[i].ID == id {
			s.notes = append(s.notes[:i], s.notes[i+1:]...)
			found = true
			break
		}
	}
	s.mu.Unlock()

	if !found {
		return fmt.Errorf("note not found: %s", id)
	}

	return s.Save()
}

// ListNotesRequest contains parameters for listing notes.
type ListNotesRequest struct {
	Tags     []string `json:"tags,omitempty"`
	FilePath string   `json:"file,omitempty"`
	Search   string   `json:"search,omitempty"`
	Author   string   `json:"author,omitempty"`
	Branch   string   `json:"branch,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}

// ListNotes lists notes with optional filters.
func (s *Store) ListNotes(req ListNotesRequest) []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Note

	for _, note := range s.notes {
		// Filter by tags
		if len(req.Tags) > 0 {
			hasTag := false
			for _, reqTag := range req.Tags {
				for _, noteTag := range note.Tags {
					if strings.EqualFold(noteTag, reqTag) {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Filter by file path
		if req.FilePath != "" && !strings.Contains(note.FilePath, req.FilePath) {
			continue
		}

		// Filter by author
		if req.Author != "" && !strings.EqualFold(note.Author, req.Author) {
			continue
		}

		// Filter by branch
		if req.Branch != "" && note.Branch != req.Branch {
			continue
		}

		// Search in title and content
		if req.Search != "" {
			searchLower := strings.ToLower(req.Search)
			if !strings.Contains(strings.ToLower(note.Title), searchLower) &&
				!strings.Contains(strings.ToLower(note.Content), searchLower) {
				continue
			}
		}

		result = append(result, note)
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

// GetNotesByFile returns all notes associated with a file.
func (s *Store) GetNotesByFile(filePath string) []Note {
	return s.ListNotes(ListNotesRequest{FilePath: filePath})
}

// GetNotesByTag returns all notes with a specific tag.
func (s *Store) GetNotesByTag(tag string) []Note {
	return s.ListNotes(ListNotesRequest{Tags: []string{tag}})
}

// GetAllNotes returns all notes.
func (s *Store) GetAllNotes() []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Note, len(s.notes))
	copy(result, s.notes)
	return result
}

// CountNotes returns the total number of notes.
func (s *Store) CountNotes() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.notes)
}

// getCurrentBranch gets the current git branch (helper).
func (s *Store) getCurrentBranch() string {
	git := NewGitIntegration(s)
	if git.IsGitRepo() {
		return git.GetCurrentBranch()
	}
	return ""
}
