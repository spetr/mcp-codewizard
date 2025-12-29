// Package analysis provides git history analysis for temporal code search.
package analysis

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

// GitHistoryAnalyzer extracts and analyzes git history for semantic search.
type GitHistoryAnalyzer struct {
	projectDir string
	config     *types.GitIndexConfig
}

// NewGitHistoryAnalyzer creates a new git history analyzer.
func NewGitHistoryAnalyzer(projectDir string, config *types.GitIndexConfig) *GitHistoryAnalyzer {
	if config == nil {
		config = DefaultGitIndexConfig()
	}
	return &GitHistoryAnalyzer{
		projectDir: projectDir,
		config:     config,
	}
}

// DefaultGitIndexConfig returns sensible defaults for git indexing.
func DefaultGitIndexConfig() *types.GitIndexConfig {
	return &types.GitIndexConfig{
		FullHistoryDays:     30,
		SampledHistoryDays:  365,
		SampleRate:          0.2,
		OnlyTags:            true,
		EmbedCommitMessages: true,
		EmbedDiffs:          true,
		MinDiffLines:        10,
		MaxDiffLines:        500,
		MaxCommits:          10000,
	}
}

// GetCommits retrieves commits from git history.
func (g *GitHistoryAnalyzer) GetCommits(since string, limit int) ([]*types.Commit, error) {
	args := []string{
		"log",
		"--format=%H%x00%h%x00%an%x00%ae%x00%at%x00%s%x00%P%x00%d",
		"--numstat",
	}

	if since != "" {
		args = append(args, "--since="+since)
	}

	if limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", limit))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return g.parseCommitLog(string(output))
}

// GetCommitsSinceHash retrieves commits since a specific commit hash.
func (g *GitHistoryAnalyzer) GetCommitsSinceHash(sinceHash string) ([]*types.Commit, error) {
	args := []string{
		"log",
		"--format=%H%x00%h%x00%an%x00%ae%x00%at%x00%s%x00%P%x00%d",
		"--numstat",
	}

	if sinceHash != "" {
		args = append(args, sinceHash+"..HEAD")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return g.parseCommitLog(string(output))
}

// GetCommitsBetween retrieves commits between two hashes.
func (g *GitHistoryAnalyzer) GetCommitsBetween(fromHash, toHash string) ([]*types.Commit, error) {
	if toHash == "" {
		toHash = "HEAD"
	}

	args := []string{
		"log",
		"--format=%H%x00%h%x00%an%x00%ae%x00%at%x00%s%x00%P%x00%d",
		"--numstat",
	}

	if fromHash != "" {
		args = append(args, fromHash+".."+toHash)
	} else {
		args = append(args, toHash)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return g.parseCommitLog(string(output))
}

// parseCommitLog parses git log output into Commit structs.
func (g *GitHistoryAnalyzer) parseCommitLog(output string) ([]*types.Commit, error) {
	var commits []*types.Commit
	var currentCommit *types.Commit

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a commit line (contains null separators)
		if strings.Contains(line, "\x00") {
			// Save previous commit
			if currentCommit != nil {
				commits = append(commits, currentCommit)
			}

			// Parse new commit
			parts := strings.Split(line, "\x00")
			if len(parts) < 6 {
				continue
			}

			timestamp, _ := strconv.ParseInt(parts[4], 10, 64)
			parentHash := ""
			if len(parts) > 6 {
				parentHash = strings.Split(parts[6], " ")[0] // First parent
			}

			currentCommit = &types.Commit{
				Hash:        parts[0],
				ShortHash:   parts[1],
				Author:      parts[2],
				AuthorEmail: parts[3],
				Date:        time.Unix(timestamp, 0),
				Message:     parts[5],
				ParentHash:  parentHash,
			}

			// Parse refs (tags, branches)
			if len(parts) > 7 && parts[7] != "" {
				refs := strings.Trim(parts[7], " ()")
				for _, ref := range strings.Split(refs, ", ") {
					ref = strings.TrimSpace(ref)
					if strings.HasPrefix(ref, "tag: ") {
						currentCommit.Tags = append(currentCommit.Tags, strings.TrimPrefix(ref, "tag: "))
						currentCommit.IsTagged = true
					} else if strings.HasPrefix(ref, "HEAD -> ") {
						currentCommit.Branch = strings.TrimPrefix(ref, "HEAD -> ")
					}
				}
			}

			// Check if merge commit
			if strings.HasPrefix(currentCommit.Message, "Merge") {
				currentCommit.IsMerge = true
			}

		} else if currentCommit != nil {
			// This is a numstat line (additions deletions filepath)
			numstatParts := strings.Fields(line)
			if len(numstatParts) >= 3 {
				additions, _ := strconv.Atoi(numstatParts[0])
				deletions, _ := strconv.Atoi(numstatParts[1])
				currentCommit.Insertions += additions
				currentCommit.Deletions += deletions
				currentCommit.FilesChanged++
			}
		}
	}

	// Don't forget the last commit
	if currentCommit != nil {
		commits = append(commits, currentCommit)
	}

	return commits, nil
}

// GetChangesForCommit retrieves file changes for a specific commit.
func (g *GitHistoryAnalyzer) GetChangesForCommit(commitHash string) ([]*types.Change, error) {
	// Get diff for this commit
	cmd := exec.Command("git", "diff-tree", "-p", "--no-commit-id", "-r", commitHash)
	cmd.Dir = g.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree failed: %w", err)
	}

	// Also get stats
	statsCmd := exec.Command("git", "diff-tree", "--numstat", "--no-commit-id", "-r", commitHash)
	statsCmd.Dir = g.projectDir
	statsOutput, _ := statsCmd.Output()

	return g.parseChanges(commitHash, string(output), string(statsOutput))
}

// parseChanges parses git diff output into Change structs.
func (g *GitHistoryAnalyzer) parseChanges(commitHash string, diffOutput, statsOutput string) ([]*types.Change, error) {
	changes := make(map[string]*types.Change)

	// Parse stats first
	for _, line := range strings.Split(statsOutput, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			additions, _ := strconv.Atoi(parts[0])
			deletions, _ := strconv.Atoi(parts[1])
			filePath := parts[2]

			// Handle renames (old -> new)
			if strings.Contains(filePath, " => ") {
				pathParts := strings.Split(filePath, " => ")
				if len(pathParts) == 2 {
					filePath = pathParts[1]
				}
			}

			changes[filePath] = &types.Change{
				ID:         fmt.Sprintf("%s:%s", commitHash[:8], filePath),
				CommitHash: commitHash,
				FilePath:   filePath,
				ChangeType: types.ChangeTypeModified,
				Additions:  additions,
				Deletions:  deletions,
			}
		}
	}

	// Parse diff to get hunks and determine change types
	currentFile := ""
	var currentHunks []types.DiffHunk
	var currentDiff strings.Builder

	diffHeaderRegex := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@(.*)$`)
	newFileRegex := regexp.MustCompile(`^new file mode`)
	deletedFileRegex := regexp.MustCompile(`^deleted file mode`)
	renameFromRegex := regexp.MustCompile(`^rename from (.+)$`)

	for _, line := range strings.Split(diffOutput, "\n") {
		// New file diff
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous file's hunks
			if currentFile != "" && len(currentHunks) > 0 {
				if c, ok := changes[currentFile]; ok {
					c.Hunks = currentHunks
					c.DiffContent = currentDiff.String()
				}
			}

			currentFile = matches[2]
			currentHunks = nil
			currentDiff.Reset()

			// Create change if not exists
			if _, ok := changes[currentFile]; !ok {
				changes[currentFile] = &types.Change{
					ID:         fmt.Sprintf("%s:%s", commitHash[:8], currentFile),
					CommitHash: commitHash,
					FilePath:   currentFile,
					ChangeType: types.ChangeTypeModified,
				}
			}

			currentDiff.WriteString(line + "\n")
			continue
		}

		// Check for new/deleted/renamed file
		if newFileRegex.MatchString(line) {
			if c, ok := changes[currentFile]; ok {
				c.ChangeType = types.ChangeTypeAdded
			}
		} else if deletedFileRegex.MatchString(line) {
			if c, ok := changes[currentFile]; ok {
				c.ChangeType = types.ChangeTypeDeleted
			}
		} else if matches := renameFromRegex.FindStringSubmatch(line); matches != nil {
			if c, ok := changes[currentFile]; ok {
				c.ChangeType = types.ChangeTypeRenamed
				c.OldPath = matches[1]
			}
		}

		// Parse hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			oldStart, _ := strconv.Atoi(matches[1])
			oldLines := 1
			if matches[2] != "" {
				oldLines, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newLines := 1
			if matches[4] != "" {
				newLines, _ = strconv.Atoi(matches[4])
			}
			context := strings.TrimSpace(matches[5])

			currentHunks = append(currentHunks, types.DiffHunk{
				OldStart: oldStart,
				OldLines: oldLines,
				NewStart: newStart,
				NewLines: newLines,
				Context:  context,
			})
		}

		// Accumulate diff content
		if currentFile != "" {
			currentDiff.WriteString(line + "\n")
		}
	}

	// Save last file's hunks
	if currentFile != "" && len(currentHunks) > 0 {
		if c, ok := changes[currentFile]; ok {
			c.Hunks = currentHunks
			c.DiffContent = currentDiff.String()
		}
	}

	// Convert map to slice
	result := make([]*types.Change, 0, len(changes))
	for _, c := range changes {
		result = append(result, c)
	}

	return result, nil
}

// IdentifyAffectedFunctions parses diff hunks to identify affected functions.
func (g *GitHistoryAnalyzer) IdentifyAffectedFunctions(change *types.Change) []string {
	functions := make(map[string]bool)

	for _, hunk := range change.Hunks {
		// Use context from hunk header (git often includes function name)
		if hunk.Context != "" {
			// Clean up the context - extract function name
			fn := extractFunctionName(hunk.Context)
			if fn != "" {
				functions[fn] = true
			}
		}
	}

	// Also look for function definitions in diff content
	if change.DiffContent != "" {
		fns := extractFunctionsFromDiff(change.DiffContent, change.FilePath)
		for _, fn := range fns {
			functions[fn] = true
		}
	}

	result := make([]string, 0, len(functions))
	for fn := range functions {
		result = append(result, fn)
	}
	return result
}

// extractFunctionName extracts function name from hunk context.
func extractFunctionName(context string) string {
	context = strings.TrimSpace(context)

	// Go: func Name(
	goFuncRegex := regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	if matches := goFuncRegex.FindStringSubmatch(context); matches != nil {
		return matches[1]
	}

	// Python: def name(
	pyFuncRegex := regexp.MustCompile(`def\s+(\w+)\s*\(`)
	if matches := pyFuncRegex.FindStringSubmatch(context); matches != nil {
		return matches[1]
	}

	// JavaScript/TypeScript: function name( or name( or name = (
	jsFuncRegex := regexp.MustCompile(`(?:function\s+)?(\w+)\s*(?:=\s*)?\(`)
	if matches := jsFuncRegex.FindStringSubmatch(context); matches != nil {
		return matches[1]
	}

	// Class: class Name
	classRegex := regexp.MustCompile(`class\s+(\w+)`)
	if matches := classRegex.FindStringSubmatch(context); matches != nil {
		return matches[1]
	}

	return ""
}

// extractFunctionsFromDiff looks for function definitions in diff content.
func extractFunctionsFromDiff(diff, filePath string) []string {
	var functions []string
	seen := make(map[string]bool)

	// Determine language from file extension
	ext := strings.ToLower(filepath.Ext(filePath))

	var patterns []*regexp.Regexp

	switch ext {
	case ".go":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`[+-]\s*func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`),
		}
	case ".py":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`[+-]\s*def\s+(\w+)\s*\(`),
			regexp.MustCompile(`[+-]\s*class\s+(\w+)`),
		}
	case ".js", ".ts", ".jsx", ".tsx":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`[+-]\s*(?:function\s+)?(\w+)\s*(?:=\s*)?(?:async\s*)?\(`),
			regexp.MustCompile(`[+-]\s*class\s+(\w+)`),
		}
	case ".java", ".c", ".cpp", ".h", ".hpp":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`[+-]\s*(?:public|private|protected|static|\s)+\s+\w+\s+(\w+)\s*\(`),
			regexp.MustCompile(`[+-]\s*class\s+(\w+)`),
		}
	case ".rs":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`[+-]\s*(?:pub\s+)?fn\s+(\w+)`),
			regexp.MustCompile(`[+-]\s*(?:pub\s+)?struct\s+(\w+)`),
		}
	}

	for _, line := range strings.Split(diff, "\n") {
		for _, pattern := range patterns {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				fn := matches[1]
				if !seen[fn] {
					seen[fn] = true
					functions = append(functions, fn)
				}
			}
		}
	}

	return functions
}

// MapChangesToChunks maps file changes to existing chunk IDs.
func (g *GitHistoryAnalyzer) MapChangesToChunks(change *types.Change, chunks []*types.Chunk) []string {
	var chunkIDs []string

	for _, chunk := range chunks {
		if chunk.FilePath != change.FilePath {
			continue
		}

		// Check if any hunk overlaps with this chunk
		for _, hunk := range change.Hunks {
			if rangesOverlap(hunk.NewStart, hunk.NewStart+hunk.NewLines, chunk.StartLine, chunk.EndLine) {
				chunkIDs = append(chunkIDs, chunk.ID)
				break
			}
		}

		// Also check by function name
		if chunk.Name != "" {
			for _, fn := range change.AffectedFunctions {
				if chunk.Name == fn {
					// Avoid duplicates
					found := false
					for _, id := range chunkIDs {
						if id == chunk.ID {
							found = true
							break
						}
					}
					if !found {
						chunkIDs = append(chunkIDs, chunk.ID)
					}
					break
				}
			}
		}
	}

	return chunkIDs
}

// rangesOverlap checks if two line ranges overlap.
func rangesOverlap(start1, end1, start2, end2 int) bool {
	return start1 <= end2 && end1 >= start2
}

// GetHeadCommit returns the current HEAD commit hash.
func (g *GitHistoryAnalyzer) GetHeadCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = g.projectDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetTags returns all tags in the repository.
func (g *GitHistoryAnalyzer) GetTags() (map[string]string, error) {
	cmd := exec.Command("git", "show-ref", "--tags")
	cmd.Dir = g.projectDir

	output, err := cmd.Output()
	if err != nil {
		// No tags is not an error
		return make(map[string]string), nil
	}

	tags := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 2 {
			hash := parts[0]
			tag := strings.TrimPrefix(parts[1], "refs/tags/")
			tags[tag] = hash
		}
	}

	return tags, nil
}

// ShouldIndexCommit determines if a commit should be indexed based on config.
func (g *GitHistoryAnalyzer) ShouldIndexCommit(commit *types.Commit) bool {
	age := time.Since(commit.Date)
	ageDays := int(age.Hours() / 24)

	// Always index recent commits
	if ageDays <= g.config.FullHistoryDays {
		return true
	}

	// Sample older commits
	if ageDays <= g.config.SampledHistoryDays {
		// Index tagged commits
		if commit.IsTagged {
			return true
		}
		// Index merge commits
		if commit.IsMerge {
			return true
		}
		// Sample based on rate (simplified - uses hash for determinism)
		hashSum := 0
		for _, c := range commit.Hash {
			hashSum += int(c)
		}
		return float32(hashSum%100) < g.config.SampleRate*100
	}

	// Beyond sampled period - only tags
	if g.config.OnlyTags {
		return commit.IsTagged
	}

	return false
}

// ShouldEmbedDiff determines if a change's diff should be embedded.
func (g *GitHistoryAnalyzer) ShouldEmbedDiff(change *types.Change) bool {
	if !g.config.EmbedDiffs {
		return false
	}

	totalLines := change.Additions + change.Deletions
	return totalLines >= g.config.MinDiffLines && totalLines <= g.config.MaxDiffLines
}

// CreateChunkHistoryEntry creates a history entry linking a change to a chunk.
func (g *GitHistoryAnalyzer) CreateChunkHistoryEntry(commit *types.Commit, change *types.Change, chunkID string) *types.ChunkHistoryEntry {
	// Determine change type for chunk
	changeType := "modified"
	if change.ChangeType == types.ChangeTypeAdded {
		changeType = "created"
	} else if change.ChangeType == types.ChangeTypeDeleted {
		changeType = "deleted"
	}

	// Create diff summary
	var summary string
	if len(change.AffectedFunctions) > 0 {
		summary = fmt.Sprintf("Changed: %s (+%d/-%d)",
			strings.Join(change.AffectedFunctions, ", "),
			change.Additions, change.Deletions)
	} else {
		summary = fmt.Sprintf("+%d/-%d lines", change.Additions, change.Deletions)
	}

	return &types.ChunkHistoryEntry{
		ChunkID:     chunkID,
		CommitHash:  commit.Hash,
		ChangeType:  changeType,
		DiffSummary: summary,
		Date:        commit.Date,
		Author:      commit.Author,
	}
}
