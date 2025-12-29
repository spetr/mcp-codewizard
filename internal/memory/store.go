package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store provides persistent storage for project memory.
type Store struct {
	projectDir string
	memoryDir  string
	localDir   string

	mu      sync.RWMutex
	notes   []Note
	decisions []Decision
	todos   []Todo
	issues  []Issue
	context WorkingContext

	// Project metadata
	projectID   string
	projectName string
	gitRemote   string
}

// StoreConfig contains configuration for the memory store.
type StoreConfig struct {
	ProjectDir string
}

// NewStore creates a new memory store.
func NewStore(cfg StoreConfig) (*Store, error) {
	memoryDir := filepath.Join(cfg.ProjectDir, ".mcp-codewizard", "memory")
	localDir := filepath.Join(cfg.ProjectDir, ".mcp-codewizard", "local")

	// Create directories
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local directory: %w", err)
	}

	// Create .gitignore for local directory
	gitignorePath := filepath.Join(cfg.ProjectDir, ".mcp-codewizard", "local", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, []byte("*\n"), 0644); err != nil {
			return nil, fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}

	s := &Store{
		projectDir:  cfg.ProjectDir,
		memoryDir:   memoryDir,
		localDir:    localDir,
		projectName: filepath.Base(cfg.ProjectDir),
	}

	// Load existing data
	if err := s.Load(); err != nil {
		return nil, fmt.Errorf("failed to load memory: %w", err)
	}

	return s, nil
}

// Load loads all memory data from disk.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load notes
	notes, err := loadJSONL[Note](filepath.Join(s.memoryDir, "notes.jsonl"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load notes: %w", err)
	}
	s.notes = notes

	// Load decisions
	decisions, err := loadJSONL[Decision](filepath.Join(s.memoryDir, "decisions.jsonl"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load decisions: %w", err)
	}
	s.decisions = decisions

	// Load todos
	todos, err := loadJSONL[Todo](filepath.Join(s.memoryDir, "todos.jsonl"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load todos: %w", err)
	}
	s.todos = todos

	// Load issues
	issues, err := loadJSONL[Issue](filepath.Join(s.memoryDir, "issues.jsonl"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load issues: %w", err)
	}
	s.issues = issues

	// Load context
	contextPath := filepath.Join(s.localDir, "context.json")
	if data, err := os.ReadFile(contextPath); err == nil {
		if err := json.Unmarshal(data, &s.context); err != nil {
			return fmt.Errorf("failed to parse context: %w", err)
		}
	}

	return nil
}

// Save saves all memory data to disk.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Save notes
	if err := saveJSONL(filepath.Join(s.memoryDir, "notes.jsonl"), s.notes); err != nil {
		return fmt.Errorf("failed to save notes: %w", err)
	}

	// Save decisions
	if err := saveJSONL(filepath.Join(s.memoryDir, "decisions.jsonl"), s.decisions); err != nil {
		return fmt.Errorf("failed to save decisions: %w", err)
	}

	// Save todos
	if err := saveJSONL(filepath.Join(s.memoryDir, "todos.jsonl"), s.todos); err != nil {
		return fmt.Errorf("failed to save todos: %w", err)
	}

	// Save issues
	if err := saveJSONL(filepath.Join(s.memoryDir, "issues.jsonl"), s.issues); err != nil {
		return fmt.Errorf("failed to save issues: %w", err)
	}

	// Save context (local only)
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

// loadJSONL loads items from a JSONL file.
func loadJSONL[T any](path string) ([]T, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []T
	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var item T
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		items = append(items, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// saveJSONL saves items to a JSONL file.
func saveJSONL[T any](path string, items []T) error {
	// Write to temp file first
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			file.Close()
			os.Remove(tmpPath)
			return err
		}
		if _, err := writer.Write(data); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return err
		}
		if _, err := writer.WriteString("\n"); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// appendJSONL appends a single item to a JSONL file.
func appendJSONL[T any](path string, item T) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		return err
	}
	if _, err := file.WriteString("\n"); err != nil {
		return err
	}

	return nil
}

// GetProjectDir returns the project directory.
func (s *Store) GetProjectDir() string {
	return s.projectDir
}

// GetMemoryDir returns the memory directory.
func (s *Store) GetMemoryDir() string {
	return s.memoryDir
}

// GetLocalDir returns the local (gitignored) directory.
func (s *Store) GetLocalDir() string {
	return s.localDir
}

// GetSummary returns a summary of the memory.
func (s *Store) GetSummary() MemorySummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := MemorySummary{
		NotesCount:     len(s.notes),
		DecisionsCount: len(s.decisions),
		CurrentTask:    s.context.CurrentTask,
		RelevantFiles:  s.context.RelevantFiles,
	}

	// Count active todos
	for _, todo := range s.todos {
		if todo.Status == TodoStatusPending || todo.Status == TodoStatusInProgress {
			summary.ActiveTodos++
		}
	}

	// Count open issues
	for _, issue := range s.issues {
		if issue.Status == "open" || issue.Status == "investigating" {
			summary.OpenIssues++
		}
	}

	// Recent notes (last 5)
	noteCount := len(s.notes)
	startIdx := 0
	if noteCount > 5 {
		startIdx = noteCount - 5
	}
	for i := noteCount - 1; i >= startIdx; i-- {
		n := s.notes[i]
		summary.RecentNotes = append(summary.RecentNotes, NoteSummary{
			ID:    n.ID,
			Title: n.Title,
			Tags:  n.Tags,
		})
	}

	// Pending high-priority todos
	for _, todo := range s.todos {
		if (todo.Status == TodoStatusPending || todo.Status == TodoStatusInProgress) &&
			todo.Priority == TodoPriorityHigh {
			summary.PendingTodos = append(summary.PendingTodos, TodoSummary{
				ID:       todo.ID,
				Title:    todo.Title,
				Priority: todo.Priority,
			})
		}
	}

	return summary
}

// GetContext returns the current working context.
func (s *Store) GetContext() WorkingContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context
}

// SetContext sets the working context.
func (s *Store) SetContext(ctx WorkingContext) error {
	s.mu.Lock()
	s.context = ctx
	s.context.UpdatedAt = time.Now()
	s.mu.Unlock()

	return s.Save()
}

// GetLastSession returns the last session summary.
func (s *Store) GetLastSession() *SessionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context.LastSession
}

// SaveSession saves a session summary.
func (s *Store) SaveSession(session SessionSummary) error {
	s.mu.Lock()
	session.EndedAt = time.Now()
	s.context.LastSession = &session
	s.context.UpdatedAt = time.Now()
	s.mu.Unlock()

	return s.Save()
}

// Clear removes all memory data.
func (s *Store) Clear() error {
	s.mu.Lock()
	s.notes = nil
	s.decisions = nil
	s.todos = nil
	s.issues = nil
	s.context = WorkingContext{}
	s.mu.Unlock()

	// Remove files
	files := []string{
		filepath.Join(s.memoryDir, "notes.jsonl"),
		filepath.Join(s.memoryDir, "decisions.jsonl"),
		filepath.Join(s.memoryDir, "todos.jsonl"),
		filepath.Join(s.memoryDir, "issues.jsonl"),
		filepath.Join(s.localDir, "context.json"),
	}

	for _, f := range files {
		os.Remove(f)
	}

	return nil
}

// Export exports all memory as a single JSON object.
func (s *Store) Export() (*ProjectMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &ProjectMemory{
		ProjectID:   s.projectID,
		ProjectName: s.projectName,
		ProjectPath: s.projectDir,
		GitRemote:   s.gitRemote,
		Notes:       s.notes,
		Decisions:   s.decisions,
		Todos:       s.todos,
		Issues:      s.issues,
		Context:     s.context,
		UpdatedAt:   time.Now(),
	}, nil
}

// Import imports memory from a ProjectMemory object.
func (s *Store) Import(pm *ProjectMemory) error {
	s.mu.Lock()
	s.notes = pm.Notes
	s.decisions = pm.Decisions
	s.todos = pm.Todos
	s.issues = pm.Issues
	s.context = pm.Context
	s.mu.Unlock()

	return s.Save()
}

// AddSearchToHistory adds a search query to the recent searches.
func (s *Store) AddSearchToHistory(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Keep last 20 searches
	s.context.RecentSearches = append([]string{query}, s.context.RecentSearches...)
	if len(s.context.RecentSearches) > 20 {
		s.context.RecentSearches = s.context.RecentSearches[:20]
	}
	s.context.UpdatedAt = time.Now()
}

// GetRecentSearches returns recent search queries.
func (s *Store) GetRecentSearches(limit int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.context.RecentSearches) {
		limit = len(s.context.RecentSearches)
	}
	return s.context.RecentSearches[:limit]
}
