package memory

import (
	"fmt"
	"strings"
	"time"
)

// AddTodoRequest contains parameters for adding a todo.
type AddTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"` // high, medium, low
	FilePath    string `json:"file,omitempty"`
	LineNumber  int    `json:"line,omitempty"`
	AssignedTo  string `json:"assigned,omitempty"` // human, ai
	DueDate     string `json:"due,omitempty"`      // ISO date
	Author      string `json:"author,omitempty"`
}

// AddTodo adds a new todo item.
func (s *Store) AddTodo(req AddTodoRequest) (*Todo, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	priority := TodoPriority(req.Priority)
	if priority == "" {
		priority = TodoPriorityMedium
	}

	now := time.Now()
	todo := Todo{
		Title:       req.Title,
		Description: req.Description,
		Priority:    priority,
		Status:      TodoStatusPending,
		FilePath:    req.FilePath,
		LineNumber:  req.LineNumber,
		AssignedTo:  req.AssignedTo,
		DueDate:     req.DueDate,
		Author:      req.Author,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if todo.Author == "" {
		todo.Author = "ai"
	}

	// Generate ID
	todo.ID = todo.GenerateID()

	// Get current branch
	todo.Branch = s.getCurrentBranch()

	s.mu.Lock()
	s.todos = append(s.todos, todo)
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return nil, fmt.Errorf("failed to save todo: %w", err)
	}

	return &todo, nil
}

// GetTodo gets a todo by ID.
func (s *Store) GetTodo(id string) (*Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			return &s.todos[i], nil
		}
	}

	return nil, fmt.Errorf("todo not found: %s", id)
}

// UpdateTodoRequest contains parameters for updating a todo.
type UpdateTodoRequest struct {
	ID          string  `json:"id"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	Status      *string `json:"status,omitempty"`
	FilePath    *string `json:"file,omitempty"`
	LineNumber  *int    `json:"line,omitempty"`
	AssignedTo  *string `json:"assigned,omitempty"`
	DueDate     *string `json:"due,omitempty"`
}

// UpdateTodo updates an existing todo.
func (s *Store) UpdateTodo(req UpdateTodoRequest) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var todo *Todo
	for i := range s.todos {
		if s.todos[i].ID == req.ID {
			todo = &s.todos[i]
			break
		}
	}

	if todo == nil {
		return nil, fmt.Errorf("todo not found: %s", req.ID)
	}

	if req.Title != nil {
		todo.Title = *req.Title
	}
	if req.Description != nil {
		todo.Description = *req.Description
	}
	if req.Priority != nil {
		todo.Priority = TodoPriority(*req.Priority)
	}
	if req.Status != nil {
		newStatus := TodoStatus(*req.Status)
		oldStatus := todo.Status
		todo.Status = newStatus

		// Set completed time if transitioning to done
		if newStatus == TodoStatusDone && oldStatus != TodoStatusDone {
			now := time.Now()
			todo.CompletedAt = &now
		} else if newStatus != TodoStatusDone {
			todo.CompletedAt = nil
		}
	}
	if req.FilePath != nil {
		todo.FilePath = *req.FilePath
	}
	if req.LineNumber != nil {
		todo.LineNumber = *req.LineNumber
	}
	if req.AssignedTo != nil {
		todo.AssignedTo = *req.AssignedTo
	}
	if req.DueDate != nil {
		todo.DueDate = *req.DueDate
	}

	todo.UpdatedAt = time.Now()

	// Save after releasing lock
	s.mu.Unlock()
	if err := s.Save(); err != nil {
		s.mu.Lock()
		return nil, fmt.Errorf("failed to save todo: %w", err)
	}

	s.mu.Lock()
	return todo, nil
}

// SetTodoStatus sets the status of a todo.
func (s *Store) SetTodoStatus(id string, status TodoStatus) (*Todo, error) {
	statusStr := string(status)
	return s.UpdateTodo(UpdateTodoRequest{
		ID:     id,
		Status: &statusStr,
	})
}

// CompleteTodo marks a todo as done.
func (s *Store) CompleteTodo(id string) (*Todo, error) {
	return s.SetTodoStatus(id, TodoStatusDone)
}

// CancelTodo marks a todo as cancelled.
func (s *Store) CancelTodo(id string) (*Todo, error) {
	return s.SetTodoStatus(id, TodoStatusCancelled)
}

// StartTodo marks a todo as in progress.
func (s *Store) StartTodo(id string) (*Todo, error) {
	return s.SetTodoStatus(id, TodoStatusInProgress)
}

// DeleteTodo deletes a todo by ID.
func (s *Store) DeleteTodo(id string) error {
	s.mu.Lock()

	found := false
	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			found = true
			break
		}
	}
	s.mu.Unlock()

	if !found {
		return fmt.Errorf("todo not found: %s", id)
	}

	return s.Save()
}

// ListTodosRequest contains parameters for listing todos.
type ListTodosRequest struct {
	Status     []string `json:"status,omitempty"`
	Priority   []string `json:"priority,omitempty"`
	FilePath   string   `json:"file,omitempty"`
	AssignedTo string   `json:"assigned,omitempty"`
	Search     string   `json:"search,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

// ListTodos lists todos with optional filters.
func (s *Store) ListTodos(req ListTodosRequest) []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Todo

	for _, todo := range s.todos {
		// Filter by status
		if len(req.Status) > 0 {
			hasStatus := false
			for _, status := range req.Status {
				if string(todo.Status) == status {
					hasStatus = true
					break
				}
			}
			if !hasStatus {
				continue
			}
		}

		// Filter by priority
		if len(req.Priority) > 0 {
			hasPriority := false
			for _, priority := range req.Priority {
				if string(todo.Priority) == priority {
					hasPriority = true
					break
				}
			}
			if !hasPriority {
				continue
			}
		}

		// Filter by file path
		if req.FilePath != "" && !strings.Contains(todo.FilePath, req.FilePath) {
			continue
		}

		// Filter by assigned to
		if req.AssignedTo != "" && !strings.EqualFold(todo.AssignedTo, req.AssignedTo) {
			continue
		}

		// Search in title and description
		if req.Search != "" {
			searchLower := strings.ToLower(req.Search)
			if !strings.Contains(strings.ToLower(todo.Title), searchLower) &&
				!strings.Contains(strings.ToLower(todo.Description), searchLower) {
				continue
			}
		}

		result = append(result, todo)
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

// GetPendingTodos returns all pending and in-progress todos.
func (s *Store) GetPendingTodos() []Todo {
	return s.ListTodos(ListTodosRequest{
		Status: []string{string(TodoStatusPending), string(TodoStatusInProgress)},
	})
}

// GetHighPriorityTodos returns all high-priority pending todos.
func (s *Store) GetHighPriorityTodos() []Todo {
	return s.ListTodos(ListTodosRequest{
		Status:   []string{string(TodoStatusPending), string(TodoStatusInProgress)},
		Priority: []string{string(TodoPriorityHigh)},
	})
}

// GetTodosByFile returns all todos associated with a file.
func (s *Store) GetTodosByFile(filePath string) []Todo {
	return s.ListTodos(ListTodosRequest{FilePath: filePath})
}

// GetAllTodos returns all todos.
func (s *Store) GetAllTodos() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Todo, len(s.todos))
	copy(result, s.todos)
	return result
}

// CountTodos returns the total number of todos.
func (s *Store) CountTodos() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.todos)
}

// CountPendingTodos returns the number of pending and in-progress todos.
func (s *Store) CountPendingTodos() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, todo := range s.todos {
		if todo.Status == TodoStatusPending || todo.Status == TodoStatusInProgress {
			count++
		}
	}
	return count
}

// GetOverdueTodos returns todos that are past their due date.
func (s *Store) GetOverdueTodos() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Todo
	now := time.Now()

	for _, todo := range s.todos {
		if todo.DueDate == "" {
			continue
		}
		if todo.Status == TodoStatusDone || todo.Status == TodoStatusCancelled {
			continue
		}

		dueDate, err := time.Parse("2006-01-02", todo.DueDate)
		if err != nil {
			continue
		}

		if dueDate.Before(now) {
			result = append(result, todo)
		}
	}

	return result
}
