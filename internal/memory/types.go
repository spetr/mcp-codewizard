// Package memory provides persistent memory storage for AI coding assistants.
// Memory is stored in JSONL format for easy git merging and version control.
package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Note represents a note about code, architecture, or findings.
// Notes are tracked in git and shared with the team.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"` // Markdown
	Tags      []string  `json:"tags,omitempty"`
	FilePath  string    `json:"file,omitempty"`       // Related file
	LineStart int       `json:"line_start,omitempty"` // Start line in file
	LineEnd   int       `json:"line_end,omitempty"`   // End line in file
	Author    string    `json:"author"`               // "human", "ai:claude", "ai:gemini"
	Branch    string    `json:"branch,omitempty"`     // Where it was created
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GenerateID generates a unique ID for a note.
func (n *Note) GenerateID() string {
	data := fmt.Sprintf("%s:%s:%d", n.Title, n.CreatedAt.String(), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "n" + hex.EncodeToString(hash[:])[:7]
}

// DecisionStatus represents the status of an architecture decision.
type DecisionStatus string

const (
	DecisionStatusProposed   DecisionStatus = "proposed"
	DecisionStatusAccepted   DecisionStatus = "accepted"
	DecisionStatusDeprecated DecisionStatus = "deprecated"
	DecisionStatusSuperseded DecisionStatus = "superseded"
)

// Decision represents an architecture or design decision (ADR-style).
// Decisions are tracked in git for team visibility.
type Decision struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Status       DecisionStatus `json:"status"`
	Context      string         `json:"context"`      // Why we're deciding
	Decision     string         `json:"decision"`     // What we decided
	Consequences string         `json:"consequences"` // What this means
	Alternatives []string       `json:"alternatives,omitempty"`
	SupersededBy string         `json:"superseded_by,omitempty"`
	Author       string         `json:"author"`
	Branch       string         `json:"branch,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// GenerateID generates a unique ID for a decision.
func (d *Decision) GenerateID() string {
	data := fmt.Sprintf("%s:%s:%d", d.Title, d.CreatedAt.String(), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "d" + hex.EncodeToString(hash[:])[:7]
}

// TodoPriority represents the priority of a TODO item.
type TodoPriority string

const (
	TodoPriorityHigh   TodoPriority = "high"
	TodoPriorityMedium TodoPriority = "medium"
	TodoPriorityLow    TodoPriority = "low"
)

// TodoStatus represents the status of a TODO item.
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusDone       TodoStatus = "done"
	TodoStatusCancelled  TodoStatus = "cancelled"
)

// Todo represents a TODO item for tracking work.
type Todo struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Priority    TodoPriority `json:"priority"`
	Status      TodoStatus   `json:"status"`
	FilePath    string       `json:"file,omitempty"`
	LineNumber  int          `json:"line,omitempty"`
	AssignedTo  string       `json:"assigned,omitempty"` // "human", "ai"
	DueDate     string       `json:"due,omitempty"`      // ISO date string
	Author      string       `json:"author"`
	Branch      string       `json:"branch,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}

// GenerateID generates a unique ID for a todo.
func (t *Todo) GenerateID() string {
	data := fmt.Sprintf("%s:%s:%d", t.Title, t.CreatedAt.String(), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "t" + hex.EncodeToString(hash[:])[:7]
}

// Issue represents a known issue or bug.
type Issue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // critical, major, minor
	FilePath    string    `json:"file,omitempty"`
	LineNumber  int       `json:"line,omitempty"`
	Status      string    `json:"status"` // open, investigating, resolved
	Resolution  string    `json:"resolution,omitempty"`
	Author      string    `json:"author"`
	Branch      string    `json:"branch,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// GenerateID generates a unique ID for an issue.
func (i *Issue) GenerateID() string {
	data := fmt.Sprintf("%s:%s:%d", i.Title, i.CreatedAt.String(), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "i" + hex.EncodeToString(hash[:])[:7]
}

// SessionSummary represents a summary of a previous session.
type SessionSummary struct {
	EndedAt    time.Time `json:"ended_at"`
	Summary    string    `json:"summary"`               // What we did
	NextSteps  []string  `json:"next_steps,omitempty"`  // What to do next
	OpenIssues []string  `json:"open_issues,omitempty"` // Unresolved problems
	Agent      string    `json:"agent,omitempty"`       // Which AI agent
}

// WorkingContext represents the current working context for session continuity.
type WorkingContext struct {
	// What we're working on
	CurrentTask   string   `json:"current_task,omitempty"`
	TaskDetails   string   `json:"task_details,omitempty"`
	RelevantFiles []string `json:"relevant_files,omitempty"`

	// Git state
	ActiveBranch string  `json:"branch,omitempty"`
	PRInProgress *PRInfo `json:"pr,omitempty"`

	// Session continuity
	LastSession *SessionSummary `json:"last_session,omitempty"`

	// Recent activity
	RecentSearches []string `json:"recent_searches,omitempty"`

	// Timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// PRInfo represents information about a pull request in progress.
type PRInfo struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	BaseBranch  string `json:"base"`
	HeadBranch  string `json:"head"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

// ProjectMemory represents the complete memory for a project.
type ProjectMemory struct {
	// Project identification
	ProjectID   string `json:"project_id"`   // Hash of initial commit or UUID
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
	GitRemote   string `json:"git_remote,omitempty"`

	// Memory collections
	Notes     []Note     `json:"notes,omitempty"`
	Decisions []Decision `json:"decisions,omitempty"`
	Todos     []Todo     `json:"todos,omitempty"`
	Issues    []Issue    `json:"issues,omitempty"`

	// Working context
	Context WorkingContext `json:"context"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProjectInfo contains basic project information.
type ProjectInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// GitState represents the current git state of the project.
type GitState struct {
	Branch           string   `json:"branch"`
	Commit           string   `json:"commit"`
	CommitMessage    string   `json:"commit_message,omitempty"`
	IsDirty          bool     `json:"is_dirty"`
	DirtyFiles       []string `json:"dirty_files,omitempty"`
	Remote           string   `json:"remote,omitempty"`
	Ahead            int      `json:"ahead"`
	Behind           int      `json:"behind"`
	MergeInProgress  bool     `json:"merge_in_progress"`
	RebaseInProgress bool     `json:"rebase_in_progress"`
	ConflictFiles    []string `json:"conflict_files,omitempty"`
}

// IndexState represents the state of the code index.
type IndexState struct {
	Status       string    `json:"status"` // ready, stale, missing, indexing
	TotalFiles   int       `json:"total_files"`
	TotalChunks  int       `json:"total_chunks"`
	TotalSymbols int       `json:"total_symbols"`
	LastIndexed  time.Time `json:"last_indexed"`
	StaleFiles   int       `json:"stale_files"`
	IndexSize    string    `json:"index_size"` // Human readable
}

// MemorySummary represents a summary of the project memory.
type MemorySummary struct {
	NotesCount     int           `json:"notes_count"`
	DecisionsCount int           `json:"decisions_count"`
	ActiveTodos    int           `json:"active_todos"`
	OpenIssues     int           `json:"open_issues"`
	RecentNotes    []NoteSummary `json:"recent_notes,omitempty"`
	PendingTodos   []TodoSummary `json:"pending_todos,omitempty"`
	CurrentTask    string        `json:"current_task,omitempty"`
	RelevantFiles  []string      `json:"relevant_files,omitempty"`
}

// NoteSummary is a brief summary of a note.
type NoteSummary struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

// TodoSummary is a brief summary of a todo.
type TodoSummary struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Priority TodoPriority `json:"priority"`
}

// SuggestedAction represents an action suggested to the AI agent.
type SuggestedAction struct {
	Priority    string `json:"priority"` // critical, high, medium, low
	Type        string `json:"type"`     // reindex, review, fix, info
	Title       string `json:"title"`
	Description string `json:"description"`
	Tool        string `json:"tool,omitempty"`
	Args        any    `json:"args,omitempty"`
}

// ProjectInitData contains all initialization data for a new session.
type ProjectInitData struct {
	Project          ProjectInfo       `json:"project"`
	Git              GitState          `json:"git"`
	Index            IndexState        `json:"index"`
	Memory           MemorySummary     `json:"memory"`
	SuggestedActions []SuggestedAction `json:"suggested_actions,omitempty"`
	LastSession      *SessionSummary   `json:"last_session,omitempty"`
}

// MergeConflict represents a conflict during memory merge.
type MergeConflict struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // note, decision, todo, issue
	Field      string `json:"field,omitempty"`
	Base       any    `json:"base,omitempty"`
	Ours       any    `json:"ours"`
	Theirs     any    `json:"theirs"`
	Resolved   bool   `json:"resolved"`
	Resolution any    `json:"resolution,omitempty"`
}

// MergeResult represents the result of a memory merge operation.
type MergeResult struct {
	Success   bool            `json:"success"`
	Merged    int             `json:"merged"`
	Added     int             `json:"added"`
	Modified  int             `json:"modified"`
	Deleted   int             `json:"deleted"`
	Conflicts []MergeConflict `json:"conflicts,omitempty"`
}
