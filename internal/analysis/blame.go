// Package analysis provides code analysis tools like git blame, dead code detection, and complexity metrics.
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
)

// BlameInfo contains git blame information for a line or range.
type BlameInfo struct {
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Date      time.Time `json:"date"`
	CommitSHA string    `json:"commit_sha"`
	Summary   string    `json:"summary"`
}

// LineBlame contains blame info for a specific line.
type LineBlame struct {
	Line int       `json:"line"`
	Info BlameInfo `json:"info"`
}

// FileBlame contains blame info for an entire file.
type FileBlame struct {
	FilePath string       `json:"file_path"`
	Lines    []*LineBlame `json:"lines"`
}

// ChunkBlame contains aggregated blame info for a code chunk.
type ChunkBlame struct {
	FilePath   string               `json:"file_path"`
	StartLine  int                  `json:"start_line"`
	EndLine    int                  `json:"end_line"`
	Authors    map[string]int       `json:"authors"`      // author -> line count
	LastChange *BlameInfo           `json:"last_change"`  // most recent change
	FirstChange *BlameInfo          `json:"first_change"` // oldest change in range
	LineCount  int                  `json:"line_count"`
}

// BlameAnalyzer provides git blame functionality.
type BlameAnalyzer struct {
	projectDir string
}

// NewBlameAnalyzer creates a new blame analyzer.
func NewBlameAnalyzer(projectDir string) *BlameAnalyzer {
	return &BlameAnalyzer{projectDir: projectDir}
}

// IsGitRepo checks if the project directory is a git repository.
func (b *BlameAnalyzer) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = b.projectDir
	return cmd.Run() == nil
}

// GetFileBlame returns blame info for all lines in a file.
func (b *BlameAnalyzer) GetFileBlame(filePath string) (*FileBlame, error) {
	// Make path relative to project dir
	relPath, err := filepath.Rel(b.projectDir, filePath)
	if err != nil {
		relPath = filePath
	}

	// Run git blame with porcelain format for easy parsing
	cmd := exec.Command("git", "blame", "--porcelain", relPath)
	cmd.Dir = b.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git blame failed: %w", err)
	}

	return b.parsePorcelainBlame(filePath, string(output))
}

// GetRangeBlame returns aggregated blame info for a line range.
func (b *BlameAnalyzer) GetRangeBlame(filePath string, startLine, endLine int) (*ChunkBlame, error) {
	// Make path relative to project dir
	relPath, err := filepath.Rel(b.projectDir, filePath)
	if err != nil {
		relPath = filePath
	}

	// Run git blame for specific range
	rangeArg := fmt.Sprintf("-L%d,%d", startLine, endLine)
	cmd := exec.Command("git", "blame", "--porcelain", rangeArg, relPath)
	cmd.Dir = b.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git blame failed: %w", err)
	}

	fileBlame, err := b.parsePorcelainBlame(filePath, string(output))
	if err != nil {
		return nil, err
	}

	return b.aggregateBlame(filePath, startLine, endLine, fileBlame.Lines), nil
}

// parsePorcelainBlame parses git blame --porcelain output.
func (b *BlameAnalyzer) parsePorcelainBlame(filePath, output string) (*FileBlame, error) {
	result := &FileBlame{
		FilePath: filePath,
		Lines:    make([]*LineBlame, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex for the header line: <sha> <orig-line> <final-line> [<num-lines>]
	headerRegex := regexp.MustCompile(`^([0-9a-f]{40}) (\d+) (\d+)`)

	var currentInfo *BlameInfo
	var currentLine int
	commits := make(map[string]*BlameInfo) // cache commit info

	for scanner.Scan() {
		line := scanner.Text()

		// Check for header line (commit SHA)
		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			sha := matches[1]
			currentLine, _ = strconv.Atoi(matches[3])

			// Check cache first
			if cached, ok := commits[sha]; ok {
				currentInfo = cached
			} else {
				currentInfo = &BlameInfo{CommitSHA: sha}
				commits[sha] = currentInfo
			}
			continue
		}

		// Parse metadata lines
		if currentInfo != nil {
			switch {
			case strings.HasPrefix(line, "author "):
				currentInfo.Author = strings.TrimPrefix(line, "author ")
			case strings.HasPrefix(line, "author-mail "):
				email := strings.TrimPrefix(line, "author-mail ")
				currentInfo.Email = strings.Trim(email, "<>")
			case strings.HasPrefix(line, "author-time "):
				timestamp, _ := strconv.ParseInt(strings.TrimPrefix(line, "author-time "), 10, 64)
				currentInfo.Date = time.Unix(timestamp, 0)
			case strings.HasPrefix(line, "summary "):
				currentInfo.Summary = strings.TrimPrefix(line, "summary ")
			case strings.HasPrefix(line, "\t"):
				// This is the actual source line - we're done with this entry
				if currentInfo != nil && currentLine > 0 {
					result.Lines = append(result.Lines, &LineBlame{
						Line: currentLine,
						Info: *currentInfo,
					})
				}
			}
		}
	}

	return result, scanner.Err()
}

// aggregateBlame aggregates line blame info into chunk blame.
func (b *BlameAnalyzer) aggregateBlame(filePath string, startLine, endLine int, lines []*LineBlame) *ChunkBlame {
	result := &ChunkBlame{
		FilePath:  filePath,
		StartLine: startLine,
		EndLine:   endLine,
		Authors:   make(map[string]int),
		LineCount: endLine - startLine + 1,
	}

	var lastChange, firstChange *BlameInfo

	for _, lb := range lines {
		if lb.Line < startLine || lb.Line > endLine {
			continue
		}

		// Count by author
		result.Authors[lb.Info.Author]++

		// Track last change (most recent)
		if lastChange == nil || lb.Info.Date.After(lastChange.Date) {
			info := lb.Info // copy
			lastChange = &info
		}

		// Track first change (oldest)
		if firstChange == nil || lb.Info.Date.Before(firstChange.Date) {
			info := lb.Info // copy
			firstChange = &info
		}
	}

	result.LastChange = lastChange
	result.FirstChange = firstChange

	return result
}

// GetRecentChanges returns files changed within the specified duration.
func (b *BlameAnalyzer) GetRecentChanges(since time.Duration) ([]string, error) {
	sinceDate := time.Now().Add(-since).Format("2006-01-02")

	cmd := exec.Command("git", "log", "--since="+sinceDate, "--name-only", "--pretty=format:")
	cmd.Dir = b.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	// Deduplicate files
	seen := make(map[string]bool)
	var files []string

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file != "" && !seen[file] {
			seen[file] = true
			files = append(files, file)
		}
	}

	return files, nil
}

// GetFileHistory returns commit history for a file.
func (b *BlameAnalyzer) GetFileHistory(filePath string, limit int) ([]*BlameInfo, error) {
	relPath, err := filepath.Rel(b.projectDir, filePath)
	if err != nil {
		relPath = filePath
	}

	format := "--format=%H%n%an%n%ae%n%at%n%s%n---"
	cmd := exec.Command("git", "log", format, fmt.Sprintf("-n%d", limit), "--", relPath)
	cmd.Dir = b.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	var history []*BlameInfo
	lines := strings.Split(string(output), "\n")

	for i := 0; i+5 <= len(lines); i += 6 {
		if lines[i] == "" {
			continue
		}

		timestamp, _ := strconv.ParseInt(lines[i+3], 10, 64)
		history = append(history, &BlameInfo{
			CommitSHA: lines[i],
			Author:    lines[i+1],
			Email:     lines[i+2],
			Date:      time.Unix(timestamp, 0),
			Summary:   lines[i+4],
		})
	}

	return history, nil
}
