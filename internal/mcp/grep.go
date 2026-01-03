// Package mcp implements the MCP server for code search.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// GrepMatch represents a single match from grep_code.
type GrepMatch struct {
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Column     int      `json:"column,omitempty"`
	Content    string   `json:"content"`
	ContextBefore []string `json:"context_before,omitempty"`
	ContextAfter  []string `json:"context_after,omitempty"`
}

// GrepResult represents the result of grep_code.
type GrepResult struct {
	Matches    []GrepMatch `json:"matches"`
	TotalCount int         `json:"total_count"`
	Truncated  bool        `json:"truncated"`
	Backend    string      `json:"backend"` // "ripgrep" or "go-regexp"
}

// handleGrepCode handles the grep_code tool.
func (s *Server) handleGrepCode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern := req.GetString("pattern", "")
	if pattern == "" {
		return mcp.NewToolResultError("pattern is required"), nil
	}

	pathFilter := req.GetString("path", "")
	contextLines := req.GetInt("context_lines", 3)
	maxResults := req.GetInt("max_results", 50)
	caseSensitive := req.GetBool("case_sensitive", false)
	indexedOnly := req.GetBool("indexed_only", true)
	literal := req.GetBool("literal", false)

	// Try ripgrep first, fall back to Go implementation
	result, err := s.grepWithRipgrep(ctx, pattern, pathFilter, contextLines, maxResults, caseSensitive, indexedOnly, literal)
	if err != nil {
		// Fall back to Go regexp
		result, err = s.grepWithGoRegexp(ctx, pattern, pathFilter, contextLines, maxResults, caseSensitive, indexedOnly, literal)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("grep failed: %v", err)), nil
		}
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// grepWithRipgrep uses ripgrep (rg) for fast searching.
func (s *Server) grepWithRipgrep(ctx context.Context, pattern, pathFilter string, contextLines, maxResults int, caseSensitive, indexedOnly, literal bool) (*GrepResult, error) {
	// Check if rg is available
	rgPath, err := exec.LookPath("rg")
	if err != nil {
		return nil, fmt.Errorf("ripgrep not found: %w", err)
	}

	args := []string{
		"--json",
		"--max-count", strconv.Itoa(maxResults * 2), // Get more to account for context grouping
	}

	if contextLines > 0 {
		args = append(args, "-C", strconv.Itoa(contextLines))
	}

	if !caseSensitive {
		args = append(args, "-i")
	}

	if literal {
		args = append(args, "-F") // Fixed string (literal)
	}

	// Add path filter as glob
	if pathFilter != "" {
		args = append(args, "-g", pathFilter)
	}

	// Add include patterns from config if indexedOnly
	if indexedOnly && s.config != nil && len(s.config.Index.Include) > 0 {
		for _, inc := range s.config.Index.Include {
			args = append(args, "-g", inc)
		}
	}

	// Add exclude patterns from config
	if s.config != nil && len(s.config.Index.Exclude) > 0 {
		for _, exc := range s.config.Index.Exclude {
			args = append(args, "-g", "!"+exc)
		}
	}

	args = append(args, pattern, s.projectDir)

	cmd := exec.CommandContext(ctx, rgPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 1 means no matches, which is OK
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return &GrepResult{
				Matches:    []GrepMatch{},
				TotalCount: 0,
				Backend:    "ripgrep",
			}, nil
		}
		return nil, fmt.Errorf("ripgrep failed: %w", err)
	}

	return s.parseRipgrepOutput(output, maxResults)
}

// parseRipgrepOutput parses JSON output from ripgrep.
func (s *Server) parseRipgrepOutput(output []byte, maxResults int) (*GrepResult, error) {
	var matches []GrepMatch
	currentMatch := GrepMatch{}
	var contextBefore, contextAfter []string
	inContext := false
	lastMatchLine := 0

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		data, _ := msg["data"].(map[string]any)

		switch msgType {
		case "match":
			// Save previous match if exists
			if currentMatch.File != "" {
				currentMatch.ContextBefore = contextBefore
				currentMatch.ContextAfter = contextAfter
				matches = append(matches, currentMatch)
				if len(matches) >= maxResults {
					return &GrepResult{
						Matches:    matches,
						TotalCount: len(matches),
						Truncated:  true,
						Backend:    "ripgrep",
					}, nil
				}
			}

			path, _ := data["path"].(map[string]any)
			pathText, _ := path["text"].(string)
			lineNum, _ := data["line_number"].(float64)
			linesData, _ := data["lines"].(map[string]any)
			text, _ := linesData["text"].(string)

			// Get relative path
			relPath, _ := filepath.Rel(s.projectDir, pathText)

			currentMatch = GrepMatch{
				File:    relPath,
				Line:    int(lineNum),
				Content: strings.TrimRight(text, "\n\r"),
			}
			contextBefore = nil
			contextAfter = nil
			lastMatchLine = int(lineNum)
			inContext = true

		case "context":
			if !inContext {
				continue
			}

			lineNum, _ := data["line_number"].(float64)
			linesData, _ := data["lines"].(map[string]any)
			text, _ := linesData["text"].(string)

			if int(lineNum) < lastMatchLine {
				contextBefore = append(contextBefore, strings.TrimRight(text, "\n\r"))
			} else {
				contextAfter = append(contextAfter, strings.TrimRight(text, "\n\r"))
			}

		case "end":
			// Finalize current match
			if currentMatch.File != "" {
				currentMatch.ContextBefore = contextBefore
				currentMatch.ContextAfter = contextAfter
				matches = append(matches, currentMatch)
			}
			currentMatch = GrepMatch{}
			contextBefore = nil
			contextAfter = nil
			inContext = false
		}
	}

	// Handle last match
	if currentMatch.File != "" {
		currentMatch.ContextBefore = contextBefore
		currentMatch.ContextAfter = contextAfter
		matches = append(matches, currentMatch)
	}

	return &GrepResult{
		Matches:    matches,
		TotalCount: len(matches),
		Truncated:  len(matches) >= maxResults,
		Backend:    "ripgrep",
	}, nil
}

// grepWithGoRegexp uses Go's regexp for searching when ripgrep is not available.
func (s *Server) grepWithGoRegexp(ctx context.Context, pattern, pathFilter string, contextLines, maxResults int, caseSensitive, indexedOnly, literal bool) (*GrepResult, error) {
	// Prepare pattern
	if literal {
		pattern = regexp.QuoteMeta(pattern)
	}
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var matches []GrepMatch

	// Walk the project directory
	err = filepath.WalkDir(s.projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check if we have enough matches
		if len(matches) >= maxResults {
			return filepath.SkipAll
		}

		relPath, _ := filepath.Rel(s.projectDir, path)

		// Check exclude patterns
		if s.config != nil {
			for _, exc := range s.config.Index.Exclude {
				if matchGlobPath(exc, relPath) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

		// Check path filter
		if pathFilter != "" && !matchGlobPath(pathFilter, relPath) {
			return nil
		}

		// Check include patterns if indexedOnly
		if indexedOnly && s.config != nil && len(s.config.Index.Include) > 0 {
			included := false
			for _, inc := range s.config.Index.Include {
				if matchGlobPath(inc, relPath) {
					included = true
					break
				}
			}
			if !included {
				return nil
			}
		}

		// Read and search file
		fileMatches, err := s.searchFile(path, relPath, re, contextLines, maxResults-len(matches))
		if err != nil {
			return nil // Skip files we can't read
		}

		matches = append(matches, fileMatches...)
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	return &GrepResult{
		Matches:    matches,
		TotalCount: len(matches),
		Truncated:  len(matches) >= maxResults,
		Backend:    "go-regexp",
	}, nil
}

// searchFile searches a single file for matches.
func (s *Server) searchFile(path, relPath string, re *regexp.Regexp, contextLines, maxMatches int) ([]GrepMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []GrepMatch
	var lines []string
	scanner := bufio.NewScanner(file)

	// Read all lines first (for context)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for i, line := range lines {
		if len(matches) >= maxMatches {
			break
		}

		if re.MatchString(line) {
			match := GrepMatch{
				File:    relPath,
				Line:    i + 1, // 1-indexed
				Content: line,
			}

			// Add context before
			if contextLines > 0 {
				start := i - contextLines
				if start < 0 {
					start = 0
				}
				for j := start; j < i; j++ {
					match.ContextBefore = append(match.ContextBefore, lines[j])
				}
			}

			// Add context after
			if contextLines > 0 {
				end := i + contextLines + 1
				if end > len(lines) {
					end = len(lines)
				}
				for j := i + 1; j < end; j++ {
					match.ContextAfter = append(match.ContextAfter, lines[j])
				}
			}

			matches = append(matches, match)
		}
	}

	return matches, nil
}

// matchGlobPath matches a glob pattern against a path.
func matchGlobPath(pattern, path string) bool {
	// Handle ** for recursive matching
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")

		// Handle patterns like "**/dir/**" (match anywhere in path)
		if len(parts) == 3 && parts[0] == "" && parts[2] == "" {
			// Pattern is "**/something/**"
			middle := strings.Trim(parts[1], "/")
			return strings.Contains(path, middle+"/") || strings.Contains(path, "/"+middle)
		}

		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			if prefix != "" && !strings.HasPrefix(path, prefix) {
				return false
			}

			if suffix == "" {
				return true
			}

			// Handle suffix patterns like "/**" (matches any subdirectory)
			if suffix == "" || suffix == "/" {
				return true
			}

			// Check if suffix contains another **
			if strings.Contains(suffix, "**") {
				// Suffix is like "dir/**" - check if path contains dir
				dirPart := strings.Split(suffix, "**")[0]
				dirPart = strings.Trim(dirPart, "/")
				return strings.Contains(path, "/"+dirPart+"/") || strings.HasPrefix(path, dirPart+"/")
			}

			// Match suffix against basename or path segments
			base := filepath.Base(path)
			matched, _ := filepath.Match(suffix, base)
			if matched {
				return true
			}

			// Try matching against each path segment
			segments := strings.Split(path, "/")
			for _, seg := range segments {
				if m, _ := filepath.Match(suffix, seg); m {
					return true
				}
			}
			return false
		}
	}

	// Simple glob match
	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	// Try matching against basename
	matched, _ = filepath.Match(pattern, filepath.Base(path))
	return matched
}
