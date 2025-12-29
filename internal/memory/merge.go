package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// MergeStrategy defines how to merge memory files.
type MergeStrategy int

const (
	// MergeStrategyAuto attempts automatic merge, marks conflicts.
	MergeStrategyAuto MergeStrategy = iota
	// MergeStrategyOurs takes our version on conflict.
	MergeStrategyOurs
	// MergeStrategyTheirs takes their version on conflict.
	MergeStrategyTheirs
)

// Merger handles three-way merging of memory files.
type Merger struct {
	store    *Store
	strategy MergeStrategy
}

// NewMerger creates a new merger with the specified strategy.
func NewMerger(store *Store, strategy MergeStrategy) *Merger {
	return &Merger{
		store:    store,
		strategy: strategy,
	}
}

// SetStrategy sets the merge strategy.
func (m *Merger) SetStrategy(strategy MergeStrategy) {
	m.strategy = strategy
}

// MergeFromGit performs a three-way merge using a git base commit.
// This is called after a git merge to reconcile memory files.
func (m *Merger) MergeFromGit(baseCommit string) (*MergeResult, error) {
	// First check if there are any conflicts
	conflicts, err := m.DetectMergeConflicts()
	if err != nil {
		return nil, fmt.Errorf("failed to detect conflicts: %w", err)
	}

	if len(conflicts) == 0 {
		// No conflicts in memory files, just reload and return
		if err := m.store.Load(); err != nil {
			return nil, err
		}
		return &MergeResult{Success: true, Merged: 0}, nil
	}

	// Perform the merge
	return m.MergeAll()
}

// DetectMergeConflicts checks if there are merge conflicts in memory files.
func (m *Merger) DetectMergeConflicts() ([]string, error) {
	// Check git for conflicted files
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = m.store.projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // Not in a git repo or no conflicts
	}

	var conflicts []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		file := scanner.Text()
		if strings.HasPrefix(file, ".mcp-codewizard/memory/") {
			conflicts = append(conflicts, file)
		}
	}

	return conflicts, nil
}

// MergeAll merges all memory files.
func (m *Merger) MergeAll() (*MergeResult, error) {
	result := &MergeResult{Success: true}

	// Check for conflicts
	conflicts, err := m.DetectMergeConflicts()
	if err != nil {
		return nil, err
	}

	if len(conflicts) == 0 {
		// No conflicts, just reload
		return result, m.store.Load()
	}

	// Handle each conflicted file
	for _, file := range conflicts {
		partialResult, err := m.mergeFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to merge %s: %w", file, err)
		}

		result.Added += partialResult.Added
		result.Modified += partialResult.Modified
		result.Deleted += partialResult.Deleted
		result.Conflicts = append(result.Conflicts, partialResult.Conflicts...)
	}

	if len(result.Conflicts) > 0 {
		result.Success = false
	}

	// Reload the store
	if err := m.store.Load(); err != nil {
		return nil, err
	}

	return result, nil
}

// mergeFile merges a specific memory file.
func (m *Merger) mergeFile(relativePath string) (*MergeResult, error) {
	fullPath := filepath.Join(m.store.projectDir, relativePath)

	switch {
	case strings.HasSuffix(relativePath, "notes.jsonl"):
		return m.mergeNotes(fullPath)
	case strings.HasSuffix(relativePath, "decisions.jsonl"):
		return m.mergeDecisions(fullPath)
	case strings.HasSuffix(relativePath, "todos.jsonl"):
		return m.mergeTodos(fullPath)
	case strings.HasSuffix(relativePath, "issues.jsonl"):
		return m.mergeIssues(fullPath)
	default:
		// For unknown files, use git's merge
		return &MergeResult{Success: true}, nil
	}
}

// getGitVersions gets base, ours, and theirs versions of a file.
func (m *Merger) getGitVersions(path string) (base, ours, theirs []byte, err error) {
	dir := filepath.Dir(path)
	file := filepath.Base(path)

	// Get base version (:1:)
	cmd := exec.Command("git", "show", ":1:"+file)
	cmd.Dir = dir
	base, _ = cmd.Output() // Might not exist

	// Get ours version (:2:)
	cmd = exec.Command("git", "show", ":2:"+file)
	cmd.Dir = dir
	ours, _ = cmd.Output()

	// Get theirs version (:3:)
	cmd = exec.Command("git", "show", ":3:"+file)
	cmd.Dir = dir
	theirs, _ = cmd.Output()

	return base, ours, theirs, nil
}

// mergeNotes merges notes.jsonl file.
func (m *Merger) mergeNotes(path string) (*MergeResult, error) {
	base, ours, theirs, err := m.getGitVersions(path)
	if err != nil {
		return nil, err
	}

	baseNotes := parseJSONL[Note](base)
	ourNotes := parseJSONL[Note](ours)
	theirNotes := parseJSONL[Note](theirs)

	result, merged := m.threeWayMergeNotes(baseNotes, ourNotes, theirNotes)

	// Write merged result
	if err := saveJSONL(path, merged); err != nil {
		return nil, err
	}

	// Stage the file
	cmd := exec.Command("git", "add", path)
	cmd.Dir = m.store.projectDir
	_ = cmd.Run()

	return result, nil
}

// threeWayMergeNotes performs three-way merge on notes.
func (m *Merger) threeWayMergeNotes(base, ours, theirs []Note) (*MergeResult, []Note) {
	result := &MergeResult{Success: true}

	// Build maps by ID
	baseMap := make(map[string]Note)
	for _, n := range base {
		baseMap[n.ID] = n
	}
	oursMap := make(map[string]Note)
	for _, n := range ours {
		oursMap[n.ID] = n
	}
	theirsMap := make(map[string]Note)
	for _, n := range theirs {
		theirsMap[n.ID] = n
	}

	// Collect all IDs
	allIDs := make(map[string]bool)
	for id := range baseMap {
		allIDs[id] = true
	}
	for id := range oursMap {
		allIDs[id] = true
	}
	for id := range theirsMap {
		allIDs[id] = true
	}

	var merged []Note

	for id := range allIDs {
		baseNote, inBase := baseMap[id]
		ourNote, inOurs := oursMap[id]
		theirNote, inTheirs := theirsMap[id]

		switch {
		// Added only by us
		case inOurs && !inTheirs && !inBase:
			merged = append(merged, ourNote)
			result.Added++

		// Added only by them
		case !inOurs && inTheirs && !inBase:
			merged = append(merged, theirNote)
			result.Added++

		// Added by both (potential conflict)
		case inOurs && inTheirs && !inBase:
			if notesEqual(ourNote, theirNote) {
				merged = append(merged, ourNote)
			} else {
				// Conflict: both added different notes with same ID
				conflict := m.resolveNoteConflict(nil, &ourNote, &theirNote)
				if conflict != nil {
					result.Conflicts = append(result.Conflicts, *conflict)
					result.Success = false
				}
				// Take ours for now, mark as conflict
				merged = append(merged, ourNote)
			}

		// Existed in base, modified
		case inBase && inOurs && inTheirs:
			if notesEqual(ourNote, theirNote) {
				// Both made same changes
				merged = append(merged, ourNote)
			} else if notesEqual(baseNote, ourNote) {
				// We didn't change, they did
				merged = append(merged, theirNote)
				result.Modified++
			} else if notesEqual(baseNote, theirNote) {
				// They didn't change, we did
				merged = append(merged, ourNote)
				result.Modified++
			} else {
				// Both changed differently - conflict
				conflict := m.resolveNoteConflict(&baseNote, &ourNote, &theirNote)
				if conflict != nil {
					result.Conflicts = append(result.Conflicts, *conflict)
					result.Success = false
				}
				// Merge content if possible
				mergedNote := m.mergeNoteContent(baseNote, ourNote, theirNote)
				merged = append(merged, mergedNote)
				result.Modified++
			}

		// Deleted by us
		case inBase && !inOurs && inTheirs:
			if notesEqual(baseNote, theirNote) {
				// They didn't change, we deleted
				result.Deleted++
			} else {
				// They changed, we deleted - conflict
				result.Conflicts = append(result.Conflicts, MergeConflict{
					ID:     id,
					Type:   "note",
					Base:   baseNote,
					Theirs: theirNote,
				})
				result.Success = false
				// Keep theirs
				merged = append(merged, theirNote)
			}

		// Deleted by them
		case inBase && inOurs && !inTheirs:
			if notesEqual(baseNote, ourNote) {
				// We didn't change, they deleted
				result.Deleted++
			} else {
				// We changed, they deleted - conflict
				result.Conflicts = append(result.Conflicts, MergeConflict{
					ID:   id,
					Type: "note",
					Base: baseNote,
					Ours: ourNote,
				})
				result.Success = false
				// Keep ours
				merged = append(merged, ourNote)
			}

		// Deleted by both
		case inBase && !inOurs && !inTheirs:
			result.Deleted++
		}
	}

	return result, merged
}

// resolveNoteConflict resolves a note conflict based on strategy.
func (m *Merger) resolveNoteConflict(base, ours, theirs *Note) *MergeConflict {
	switch m.strategy {
	case MergeStrategyOurs:
		return nil // Take ours, no conflict
	case MergeStrategyTheirs:
		return nil // Take theirs, no conflict
	default:
		// Auto: mark as conflict
		return &MergeConflict{
			ID:     ours.ID,
			Type:   "note",
			Base:   base,
			Ours:   ours,
			Theirs: theirs,
		}
	}
}

// mergeNoteContent attempts to merge note content.
func (m *Merger) mergeNoteContent(base, ours, theirs Note) Note {
	result := ours // Start with ours

	// If content differs, append theirs as a note
	if ours.Content != theirs.Content {
		result.Content = ours.Content + "\n\n---\n**[Merged from other branch]:**\n" + theirs.Content
	}

	// Merge tags
	tagSet := make(map[string]bool)
	for _, t := range ours.Tags {
		tagSet[t] = true
	}
	for _, t := range theirs.Tags {
		tagSet[t] = true
	}
	result.Tags = make([]string, 0, len(tagSet))
	for t := range tagSet {
		result.Tags = append(result.Tags, t)
	}

	return result
}

// Similar merge functions for decisions, todos, issues...

func (m *Merger) mergeDecisions(path string) (*MergeResult, error) {
	base, ours, theirs, err := m.getGitVersions(path)
	if err != nil {
		return nil, err
	}

	baseItems := parseJSONL[Decision](base)
	ourItems := parseJSONL[Decision](ours)
	theirItems := parseJSONL[Decision](theirs)

	result, merged := m.threeWayMergeDecisions(baseItems, ourItems, theirItems)

	if err := saveJSONL(path, merged); err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "add", path)
	cmd.Dir = m.store.projectDir
	_ = cmd.Run()

	return result, nil
}

func (m *Merger) threeWayMergeDecisions(base, ours, theirs []Decision) (*MergeResult, []Decision) {
	result := &MergeResult{Success: true}

	// Build maps
	baseMap := make(map[string]Decision)
	for _, d := range base {
		baseMap[d.ID] = d
	}
	oursMap := make(map[string]Decision)
	for _, d := range ours {
		oursMap[d.ID] = d
	}
	theirsMap := make(map[string]Decision)
	for _, d := range theirs {
		theirsMap[d.ID] = d
	}

	// Collect all IDs
	allIDs := make(map[string]bool)
	for id := range baseMap {
		allIDs[id] = true
	}
	for id := range oursMap {
		allIDs[id] = true
	}
	for id := range theirsMap {
		allIDs[id] = true
	}

	var merged []Decision

	for id := range allIDs {
		_, inBase := baseMap[id]
		ourItem, inOurs := oursMap[id]
		theirItem, inTheirs := theirsMap[id]

		switch {
		case inOurs && !inTheirs && !inBase:
			merged = append(merged, ourItem)
			result.Added++
		case !inOurs && inTheirs && !inBase:
			merged = append(merged, theirItem)
			result.Added++
		case inOurs && inTheirs:
			// Take the one with later UpdatedAt
			if ourItem.UpdatedAt.After(theirItem.UpdatedAt) {
				merged = append(merged, ourItem)
			} else {
				merged = append(merged, theirItem)
			}
			if inBase {
				result.Modified++
			} else {
				result.Added++
			}
		case inBase && inOurs && !inTheirs:
			result.Deleted++
		case inBase && !inOurs && inTheirs:
			result.Deleted++
		}
	}

	return result, merged
}

func (m *Merger) mergeTodos(path string) (*MergeResult, error) {
	base, ours, theirs, err := m.getGitVersions(path)
	if err != nil {
		return nil, err
	}

	baseItems := parseJSONL[Todo](base)
	ourItems := parseJSONL[Todo](ours)
	theirItems := parseJSONL[Todo](theirs)

	result, merged := m.threeWayMergeTodos(baseItems, ourItems, theirItems)

	if err := saveJSONL(path, merged); err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "add", path)
	cmd.Dir = m.store.projectDir
	_ = cmd.Run()

	return result, nil
}

func (m *Merger) threeWayMergeTodos(base, ours, theirs []Todo) (*MergeResult, []Todo) {
	result := &MergeResult{Success: true}

	// Build maps
	baseMap := make(map[string]Todo)
	for _, t := range base {
		baseMap[t.ID] = t
	}
	oursMap := make(map[string]Todo)
	for _, t := range ours {
		oursMap[t.ID] = t
	}
	theirsMap := make(map[string]Todo)
	for _, t := range theirs {
		theirsMap[t.ID] = t
	}

	// Collect all IDs
	allIDs := make(map[string]bool)
	for id := range baseMap {
		allIDs[id] = true
	}
	for id := range oursMap {
		allIDs[id] = true
	}
	for id := range theirsMap {
		allIDs[id] = true
	}

	var merged []Todo

	for id := range allIDs {
		_, inBase := baseMap[id]
		ourItem, inOurs := oursMap[id]
		theirItem, inTheirs := theirsMap[id]

		switch {
		case inOurs && !inTheirs && !inBase:
			merged = append(merged, ourItem)
			result.Added++
		case !inOurs && inTheirs && !inBase:
			merged = append(merged, theirItem)
			result.Added++
		case inOurs && inTheirs:
			// Prefer completed status
			if ourItem.Status == TodoStatusDone || theirItem.Status == TodoStatusDone {
				if ourItem.Status == TodoStatusDone {
					merged = append(merged, ourItem)
				} else {
					merged = append(merged, theirItem)
				}
			} else if ourItem.UpdatedAt.After(theirItem.UpdatedAt) {
				merged = append(merged, ourItem)
			} else {
				merged = append(merged, theirItem)
			}
			if inBase {
				result.Modified++
			} else {
				result.Added++
			}
		case inBase && inOurs && !inTheirs:
			result.Deleted++
		case inBase && !inOurs && inTheirs:
			result.Deleted++
		}
	}

	return result, merged
}

func (m *Merger) mergeIssues(path string) (*MergeResult, error) {
	base, ours, theirs, err := m.getGitVersions(path)
	if err != nil {
		return nil, err
	}

	baseItems := parseJSONL[Issue](base)
	ourItems := parseJSONL[Issue](ours)
	theirItems := parseJSONL[Issue](theirs)

	result, merged := m.threeWayMergeIssues(baseItems, ourItems, theirItems)

	if err := saveJSONL(path, merged); err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "add", path)
	cmd.Dir = m.store.projectDir
	_ = cmd.Run()

	return result, nil
}

func (m *Merger) threeWayMergeIssues(base, ours, theirs []Issue) (*MergeResult, []Issue) {
	result := &MergeResult{Success: true}

	// Build maps
	baseMap := make(map[string]Issue)
	for _, i := range base {
		baseMap[i.ID] = i
	}
	oursMap := make(map[string]Issue)
	for _, i := range ours {
		oursMap[i.ID] = i
	}
	theirsMap := make(map[string]Issue)
	for _, i := range theirs {
		theirsMap[i.ID] = i
	}

	// Collect all IDs
	allIDs := make(map[string]bool)
	for id := range baseMap {
		allIDs[id] = true
	}
	for id := range oursMap {
		allIDs[id] = true
	}
	for id := range theirsMap {
		allIDs[id] = true
	}

	var merged []Issue

	for id := range allIDs {
		_, inBase := baseMap[id]
		ourItem, inOurs := oursMap[id]
		theirItem, inTheirs := theirsMap[id]

		switch {
		case inOurs && !inTheirs && !inBase:
			merged = append(merged, ourItem)
			result.Added++
		case !inOurs && inTheirs && !inBase:
			merged = append(merged, theirItem)
			result.Added++
		case inOurs && inTheirs:
			// Prefer resolved status
			if ourItem.Status == "resolved" || theirItem.Status == "resolved" {
				if ourItem.Status == "resolved" {
					merged = append(merged, ourItem)
				} else {
					merged = append(merged, theirItem)
				}
			} else if ourItem.UpdatedAt.After(theirItem.UpdatedAt) {
				merged = append(merged, ourItem)
			} else {
				merged = append(merged, theirItem)
			}
			if inBase {
				result.Modified++
			} else {
				result.Added++
			}
		case inBase && inOurs && !inTheirs:
			result.Deleted++
		case inBase && !inOurs && inTheirs:
			result.Deleted++
		}
	}

	return result, merged
}

// Helper functions

func parseJSONL[T any](data []byte) []T {
	var result []T
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item T
		if err := json.Unmarshal(line, &item); err == nil {
			result = append(result, item)
		}
	}
	return result
}

func notesEqual(a, b Note) bool {
	return a.Title == b.Title &&
		a.Content == b.Content &&
		a.FilePath == b.FilePath
}

// ResolveConflict resolves a specific conflict.
func (m *Merger) ResolveConflict(conflict MergeConflict, takeOurs bool) error {
	// This would update the merged file with the chosen resolution
	// For now, just reload
	return m.store.Load()
}
