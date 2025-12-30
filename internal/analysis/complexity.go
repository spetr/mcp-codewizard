package analysis

import (
	"os"
	"regexp"
	"strings"
	"unicode"
)

// ComplexityMetrics contains various code complexity measurements.
type ComplexityMetrics struct {
	FilePath            string  `json:"file_path"`
	FunctionName        string  `json:"function_name,omitempty"`
	StartLine           int     `json:"start_line"`
	EndLine             int     `json:"end_line"`

	// Line metrics
	TotalLines          int     `json:"total_lines"`
	CodeLines           int     `json:"code_lines"`      // non-blank, non-comment
	CommentLines        int     `json:"comment_lines"`
	BlankLines          int     `json:"blank_lines"`

	// Complexity metrics
	CyclomaticComplexity int    `json:"cyclomatic_complexity"`
	CognitiveComplexity  int    `json:"cognitive_complexity"`

	// Other metrics
	MaxNestingDepth     int     `json:"max_nesting_depth"`
	ParameterCount      int     `json:"parameter_count,omitempty"`
	ReturnCount         int     `json:"return_count"`

	// Quality indicators
	ComplexityRating    string  `json:"complexity_rating"` // low, medium, high, very_high
	Suggestions         []string `json:"suggestions,omitempty"`
}

// FileComplexity contains complexity metrics for an entire file.
type FileComplexity struct {
	FilePath    string               `json:"file_path"`
	Language    string               `json:"language"`
	Metrics     *ComplexityMetrics   `json:"metrics"`
	Functions   []*ComplexityMetrics `json:"functions"`
	AverageComplexity float64        `json:"average_complexity"`
	MaxComplexity     int            `json:"max_complexity"`
}

// ComplexityAnalyzer calculates code complexity metrics.
type ComplexityAnalyzer struct{}

// NewComplexityAnalyzer creates a new complexity analyzer.
func NewComplexityAnalyzer() *ComplexityAnalyzer {
	return &ComplexityAnalyzer{}
}

// AnalyzeFile analyzes complexity for an entire file.
func (c *ComplexityAnalyzer) AnalyzeFile(filePath string) (*FileComplexity, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	language := detectLanguageFromPath(filePath)
	lines := strings.Split(string(content), "\n")
	// Remove trailing empty line if file ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	result := &FileComplexity{
		FilePath: filePath,
		Language: language,
		Metrics:  c.analyzeLines(lines, language, filePath, "", 1, len(lines)),
	}

	// Find and analyze individual functions
	functions := c.findFunctions(lines, language)
	for _, fn := range functions {
		fnLines := lines[fn.startLine-1 : fn.endLine]
		metrics := c.analyzeLines(fnLines, language, filePath, fn.name, fn.startLine, fn.endLine)
		result.Functions = append(result.Functions, metrics)
	}

	// Calculate averages
	if len(result.Functions) > 0 {
		var totalComplexity int
		for _, fn := range result.Functions {
			totalComplexity += fn.CyclomaticComplexity
			if fn.CyclomaticComplexity > result.MaxComplexity {
				result.MaxComplexity = fn.CyclomaticComplexity
			}
		}
		result.AverageComplexity = float64(totalComplexity) / float64(len(result.Functions))
	}

	return result, nil
}

// AnalyzeRange analyzes complexity for a specific line range.
func (c *ComplexityAnalyzer) AnalyzeRange(filePath string, startLine, endLine int) (*ComplexityMetrics, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	language := detectLanguageFromPath(filePath)
	allLines := strings.Split(string(content), "\n")
	// Remove trailing empty line if file ends with newline
	if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}

	// Adjust bounds
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	lines := allLines[startLine-1 : endLine]
	return c.analyzeLines(lines, language, filePath, "", startLine, endLine), nil
}

// analyzeLines calculates metrics for a slice of lines.
func (c *ComplexityAnalyzer) analyzeLines(lines []string, language, filePath, funcName string, startLine, endLine int) *ComplexityMetrics {
	metrics := &ComplexityMetrics{
		FilePath:     filePath,
		FunctionName: funcName,
		StartLine:    startLine,
		EndLine:      endLine,
		TotalLines:   len(lines),
	}

	inMultiLineComment := false
	currentNesting := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count line types
		if trimmed == "" {
			metrics.BlankLines++
			continue
		}

		// Check for comments (simplified)
		isComment, newInComment := c.isCommentLine(trimmed, language, inMultiLineComment)
		inMultiLineComment = newInComment

		if isComment {
			metrics.CommentLines++
			continue
		}

		metrics.CodeLines++

		// Calculate cyclomatic complexity
		metrics.CyclomaticComplexity += c.countDecisionPoints(trimmed, language)

		// Track nesting depth
		nesting := c.calculateNesting(trimmed, language, currentNesting)
		if nesting > metrics.MaxNestingDepth {
			metrics.MaxNestingDepth = nesting
		}
		currentNesting = nesting

		// Count returns
		if c.isReturn(trimmed, language) {
			metrics.ReturnCount++
		}
	}

	// Base complexity is 1
	metrics.CyclomaticComplexity++

	// Calculate cognitive complexity (simplified approximation)
	metrics.CognitiveComplexity = c.calculateCognitiveComplexity(metrics)

	// Rate complexity
	metrics.ComplexityRating = c.rateComplexity(metrics.CyclomaticComplexity)

	// Generate suggestions
	metrics.Suggestions = c.generateSuggestions(metrics)

	return metrics
}

// isCommentLine checks if a line is a comment.
func (c *ComplexityAnalyzer) isCommentLine(line, language string, inMultiLine bool) (bool, bool) {
	switch language {
	case "go", "java", "javascript", "typescript", "c", "cpp", "rust":
		// Multi-line comment handling
		if inMultiLine {
			if strings.Contains(line, "*/") {
				return true, false
			}
			return true, true
		}
		if strings.HasPrefix(line, "/*") {
			if strings.Contains(line, "*/") {
				return true, false
			}
			return true, true
		}
		// Single-line comment
		return strings.HasPrefix(line, "//"), false

	case "python":
		if inMultiLine {
			if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, `'''`) {
				return true, false
			}
			return true, true
		}
		if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, `'''`) {
			// Check if it closes on the same line
			rest := line[3:]
			if strings.Contains(rest, `"""`) || strings.Contains(rest, `'''`) {
				return true, false
			}
			return true, true
		}
		return strings.HasPrefix(line, "#"), false

	default:
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#"), false
	}
}

// countDecisionPoints counts cyclomatic complexity decision points.
func (c *ComplexityAnalyzer) countDecisionPoints(line, language string) int {
	count := 0

	// Common decision points
	decisionKeywords := []string{
		"if", "else if", "elif", "for", "while", "case", "catch",
		"&&", "||", "?", // ternary and boolean operators
	}

	lowerLine := strings.ToLower(line)

	for _, keyword := range decisionKeywords {
		// Count keyword occurrences (word boundary check for keywords)
		if keyword == "&&" || keyword == "||" || keyword == "?" {
			count += strings.Count(line, keyword)
		} else {
			// Check for word boundary
			pattern := regexp.MustCompile(`\b` + keyword + `\b`)
			count += len(pattern.FindAllString(lowerLine, -1))
		}
	}

	// Language-specific
	switch language {
	case "go":
		// Go select cases
		if strings.Contains(lowerLine, "select") {
			count++
		}
	case "rust":
		// Rust match arms
		if strings.Contains(line, "=>") {
			count++
		}
	}

	return count
}

// calculateNesting calculates current nesting depth.
func (c *ComplexityAnalyzer) calculateNesting(line, language string, current int) int {
	opens := strings.Count(line, "{")
	closes := strings.Count(line, "}")

	// Python uses indentation
	if language == "python" {
		// Count leading spaces/tabs
		indent := 0
		for _, ch := range line {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent += 4
			} else {
				break
			}
		}
		return indent / 4
	}

	return current + opens - closes
}

// isReturn checks if line contains a return statement.
func (c *ComplexityAnalyzer) isReturn(line, language string) bool {
	return strings.Contains(line, "return ")
}

// calculateCognitiveComplexity approximates cognitive complexity.
func (c *ComplexityAnalyzer) calculateCognitiveComplexity(m *ComplexityMetrics) int {
	// Simplified: cyclomatic + nesting penalty
	nestingPenalty := m.MaxNestingDepth * 2
	return m.CyclomaticComplexity + nestingPenalty
}

// rateComplexity rates the complexity level.
func (c *ComplexityAnalyzer) rateComplexity(cyclomatic int) string {
	switch {
	case cyclomatic <= 5:
		return "low"
	case cyclomatic <= 10:
		return "medium"
	case cyclomatic <= 20:
		return "high"
	default:
		return "very_high"
	}
}

// generateSuggestions generates improvement suggestions.
func (c *ComplexityAnalyzer) generateSuggestions(m *ComplexityMetrics) []string {
	var suggestions []string

	if m.CyclomaticComplexity > 10 {
		suggestions = append(suggestions, "Consider breaking this function into smaller functions")
	}

	if m.MaxNestingDepth > 4 {
		suggestions = append(suggestions, "Deep nesting detected - consider early returns or extracting logic")
	}

	if m.CodeLines > 50 {
		suggestions = append(suggestions, "Function is long - consider splitting into smaller units")
	}

	if m.ReturnCount > 5 {
		suggestions = append(suggestions, "Many return statements - consider consolidating exit points")
	}

	if float64(m.CommentLines)/float64(m.CodeLines) < 0.1 && m.CodeLines > 20 {
		suggestions = append(suggestions, "Consider adding more comments for complex logic")
	}

	return suggestions
}

// functionInfo holds information about a found function.
type functionInfo struct {
	name      string
	startLine int
	endLine   int
}

// findFunctions finds function definitions in code.
func (c *ComplexityAnalyzer) findFunctions(lines []string, language string) []functionInfo {
	var functions []functionInfo

	var funcPattern *regexp.Regexp
	switch language {
	case "go":
		funcPattern = regexp.MustCompile(`^func\s+(\w+)`)
	case "python":
		funcPattern = regexp.MustCompile(`^def\s+(\w+)`)
	case "javascript", "typescript":
		funcPattern = regexp.MustCompile(`(?:function\s+(\w+)|(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>))`)
	case "java":
		funcPattern = regexp.MustCompile(`(?:public|private|protected)?\s*(?:static)?\s*\w+\s+(\w+)\s*\(`)
	case "rust":
		funcPattern = regexp.MustCompile(`^(?:pub\s+)?fn\s+(\w+)`)
	case "c", "cpp":
		funcPattern = regexp.MustCompile(`^\w+\s+(\w+)\s*\([^)]*\)\s*\{?`)
	default:
		return functions
	}

	braceCount := 0
	var currentFunc *functionInfo

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Look for function start
		if currentFunc == nil {
			if matches := funcPattern.FindStringSubmatch(trimmed); matches != nil {
				name := ""
				for _, m := range matches[1:] {
					if m != "" {
						name = m
						break
					}
				}
				if name != "" {
					currentFunc = &functionInfo{
						name:      name,
						startLine: lineNum,
					}
					braceCount = strings.Count(line, "{") - strings.Count(line, "}")
				}
			}
		} else {
			// Track braces to find function end
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			if braceCount <= 0 || (language == "python" && trimmed != "" && !unicode.IsSpace(rune(line[0]))) {
				currentFunc.endLine = lineNum
				functions = append(functions, *currentFunc)
				currentFunc = nil
				braceCount = 0
			}
		}
	}

	// Handle unclosed function at end of file
	if currentFunc != nil {
		currentFunc.endLine = len(lines)
		functions = append(functions, *currentFunc)
	}

	return functions
}

// detectLanguageFromPath detects language from file extension.
func detectLanguageFromPath(path string) string {
	path = strings.ToLower(path)
	switch {
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".py"):
		return "python"
	case strings.HasSuffix(path, ".js"):
		return "javascript"
	case strings.HasSuffix(path, ".ts"):
		return "typescript"
	case strings.HasSuffix(path, ".jsx"):
		return "jsx"
	case strings.HasSuffix(path, ".tsx"):
		return "tsx"
	case strings.HasSuffix(path, ".rs"):
		return "rust"
	case strings.HasSuffix(path, ".java"):
		return "java"
	case strings.HasSuffix(path, ".c"), strings.HasSuffix(path, ".h"):
		return "c"
	case strings.HasSuffix(path, ".cpp"), strings.HasSuffix(path, ".cc"), strings.HasSuffix(path, ".hpp"):
		return "cpp"
	default:
		return "unknown"
	}
}
