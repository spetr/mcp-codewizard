// Package analysis provides code analysis tools.
package analysis

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

// FileSummary contains comprehensive information about a source file.
type FileSummary struct {
	// Basic info
	FilePath     string    `json:"file_path"`
	FileName     string    `json:"file_name"`
	Language     string    `json:"language"`
	Size         int64     `json:"size_bytes"`
	ModifiedAt   time.Time `json:"modified_at"`

	// Line counts
	TotalLines   int `json:"total_lines"`
	CodeLines    int `json:"code_lines"`
	CommentLines int `json:"comment_lines"`
	BlankLines   int `json:"blank_lines"`

	// Structure
	Imports     []ImportInfo  `json:"imports"`
	Exports     []ExportInfo  `json:"exports"`
	Functions   []FunctionInfo `json:"functions"`
	Types       []TypeInfo    `json:"types"`

	// Metrics
	Complexity      *ComplexityMetrics `json:"complexity,omitempty"`
	AverageFunction float64            `json:"average_function_lines"`
	LongestFunction int                `json:"longest_function_lines"`

	// Dependencies
	InternalDeps []string `json:"internal_deps,omitempty"` // Local imports
	ExternalDeps []string `json:"external_deps,omitempty"` // External imports
}

// ImportInfo represents an import statement.
type ImportInfo struct {
	Path    string `json:"path"`
	Alias   string `json:"alias,omitempty"`
	Line    int    `json:"line"`
	IsLocal bool   `json:"is_local"` // Local vs external import
}

// ExportInfo represents an exported symbol.
type ExportInfo struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"` // function, type, constant, variable
	Line      int    `json:"line"`
	Signature string `json:"signature,omitempty"`
}

// FunctionInfo contains function summary.
type FunctionInfo struct {
	Name       string `json:"name"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	LineCount  int    `json:"line_count"`
	Signature  string `json:"signature,omitempty"`
	IsExported bool   `json:"is_exported"`
	Complexity int    `json:"complexity"`
	Parameters int    `json:"parameters"`
}

// TypeInfo contains type/struct/class summary.
type TypeInfo struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"` // struct, interface, class, enum
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	IsExported bool     `json:"is_exported"`
	Fields     []string `json:"fields,omitempty"`
	Methods    []string `json:"methods,omitempty"`
}

// SummaryAnalyzer creates file summaries.
type SummaryAnalyzer struct {
	complexityAnalyzer *ComplexityAnalyzer
}

// NewSummaryAnalyzer creates a new summary analyzer.
func NewSummaryAnalyzer() *SummaryAnalyzer {
	return &SummaryAnalyzer{
		complexityAnalyzer: NewComplexityAnalyzer(),
	}
}

// AnalyzeFile creates a comprehensive summary of a source file.
func (s *SummaryAnalyzer) AnalyzeFile(filePath string) (*FileSummary, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	// Remove trailing empty line if file ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	language := detectLanguageFromPath(filePath)

	summary := &FileSummary{
		FilePath:   filePath,
		FileName:   filepath.Base(filePath),
		Language:   language,
		Size:       fileInfo.Size(),
		ModifiedAt: fileInfo.ModTime(),
		TotalLines: len(lines),
	}

	// Count line types
	s.countLines(lines, language, summary)

	// Extract imports
	summary.Imports = s.extractImports(lines, language)
	s.categorizeDeps(summary)

	// Extract types/structs
	summary.Types = s.extractTypes(lines, language)

	// Extract functions
	summary.Functions = s.extractFunctions(lines, language)

	// Calculate function stats
	if len(summary.Functions) > 0 {
		var totalLines int
		for _, fn := range summary.Functions {
			totalLines += fn.LineCount
			if fn.LineCount > summary.LongestFunction {
				summary.LongestFunction = fn.LineCount
			}
		}
		summary.AverageFunction = float64(totalLines) / float64(len(summary.Functions))
	}

	// Extract exports
	summary.Exports = s.extractExports(summary, language)

	// Get complexity metrics
	complexity, err := s.complexityAnalyzer.AnalyzeFile(filePath)
	if err == nil {
		summary.Complexity = complexity.Metrics
	}

	return summary, nil
}

// countLines counts code, comment, and blank lines.
func (s *SummaryAnalyzer) countLines(lines []string, language string, summary *FileSummary) {
	inMultiLineComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			summary.BlankLines++
			continue
		}

		isComment, newInComment := s.isCommentLine(trimmed, language, inMultiLineComment)
		inMultiLineComment = newInComment

		if isComment {
			summary.CommentLines++
		} else {
			summary.CodeLines++
		}
	}
}

// isCommentLine checks if a line is a comment.
func (s *SummaryAnalyzer) isCommentLine(line, language string, inMultiLine bool) (bool, bool) {
	switch language {
	case "go", "java", "javascript", "typescript", "c", "cpp", "rust", "swift", "kotlin", "scala", "dart":
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
		return strings.HasPrefix(line, "//"), false

	case "python", "ruby":
		if inMultiLine {
			if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, `'''`) {
				return true, false
			}
			return true, true
		}
		if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, `'''`) {
			rest := line[3:]
			if strings.Contains(rest, `"""`) || strings.Contains(rest, `'''`) {
				return true, false
			}
			return true, true
		}
		return strings.HasPrefix(line, "#"), false

	case "html", "xml":
		if inMultiLine {
			if strings.Contains(line, "-->") {
				return true, false
			}
			return true, true
		}
		if strings.HasPrefix(line, "<!--") {
			if strings.Contains(line, "-->") {
				return true, false
			}
			return true, true
		}
		return false, false

	default:
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#"), false
	}
}

// extractImports extracts import statements.
func (s *SummaryAnalyzer) extractImports(lines []string, language string) []ImportInfo {
	var imports []ImportInfo

	var importPattern *regexp.Regexp
	switch language {
	case "go":
		// Go: import "path" or import ( "path" )
		importPattern = regexp.MustCompile(`^\s*(?:import\s+)?(?:(\w+)\s+)?["']([^"']+)["']`)
	case "python":
		// Python: import x or from x import y
		importPattern = regexp.MustCompile(`^\s*(?:from\s+(\S+)\s+)?import\s+(.+)`)
	case "javascript", "typescript":
		// JS/TS: import x from 'path' or import 'path'
		importPattern = regexp.MustCompile(`^\s*import\s+(?:(?:(?:\{[^}]*\}|\*\s+as\s+\w+|\w+)\s*,?\s*)*from\s+)?['"]([^'"]+)['"]`)
	case "java":
		// Java: import com.example.Class
		importPattern = regexp.MustCompile(`^\s*import\s+(?:static\s+)?([^;]+);`)
	case "rust":
		// Rust: use crate::path or use external_crate
		importPattern = regexp.MustCompile(`^\s*use\s+([^;]+);`)
	default:
		return imports
	}

	inImportBlock := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Go import block handling
		if language == "go" {
			if strings.HasPrefix(trimmed, "import (") {
				inImportBlock = true
				continue
			}
			if inImportBlock {
				if trimmed == ")" {
					inImportBlock = false
					continue
				}
				// Parse single import in block
				if matches := importPattern.FindStringSubmatch(trimmed); matches != nil {
					alias := ""
					path := ""
					if len(matches) > 2 && matches[1] != "" {
						alias = matches[1]
						path = matches[2]
					} else if len(matches) > 2 {
						path = matches[2]
					}
					if path != "" {
						imports = append(imports, ImportInfo{
							Path:    path,
							Alias:   alias,
							Line:    i + 1,
							IsLocal: s.isLocalImport(path, language),
						})
					}
				}
				continue
			}
		}

		if matches := importPattern.FindStringSubmatch(trimmed); matches != nil {
			var path string
			var alias string

			switch language {
			case "go":
				if len(matches) > 2 {
					alias = matches[1]
					path = matches[2]
				}
			case "python":
				if matches[1] != "" {
					path = matches[1]
				} else {
					path = strings.Split(matches[2], ",")[0]
					path = strings.TrimSpace(path)
				}
			case "javascript", "typescript":
				path = matches[1]
			case "java", "rust":
				path = matches[1]
			}

			if path != "" {
				imports = append(imports, ImportInfo{
					Path:    path,
					Alias:   alias,
					Line:    i + 1,
					IsLocal: s.isLocalImport(path, language),
				})
			}
		}
	}

	return imports
}

// isLocalImport determines if an import is local or external.
func (s *SummaryAnalyzer) isLocalImport(path, language string) bool {
	switch language {
	case "go":
		// Go: local imports don't start with known domains
		return !strings.Contains(path, ".") || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
	case "python":
		// Python: relative imports start with .
		return strings.HasPrefix(path, ".")
	case "javascript", "typescript":
		// JS/TS: relative imports start with . or /
		return strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/")
	case "rust":
		// Rust: local imports start with crate::, super::, or self::
		return strings.HasPrefix(path, "crate::") || strings.HasPrefix(path, "super::") || strings.HasPrefix(path, "self::")
	default:
		return false
	}
}

// categorizeDeps categorizes imports into internal and external.
func (s *SummaryAnalyzer) categorizeDeps(summary *FileSummary) {
	for _, imp := range summary.Imports {
		if imp.IsLocal {
			summary.InternalDeps = append(summary.InternalDeps, imp.Path)
		} else {
			summary.ExternalDeps = append(summary.ExternalDeps, imp.Path)
		}
	}
}

// extractTypes extracts type/struct/class definitions.
func (s *SummaryAnalyzer) extractTypes(lines []string, language string) []TypeInfo {
	var typeInfos []TypeInfo

	var typePattern *regexp.Regexp
	switch language {
	case "go":
		typePattern = regexp.MustCompile(`^type\s+(\w+)\s+(struct|interface)\s*\{?`)
	case "python":
		typePattern = regexp.MustCompile(`^class\s+(\w+)(?:\([^)]*\))?:`)
	case "javascript", "typescript":
		typePattern = regexp.MustCompile(`^(?:export\s+)?(?:class|interface)\s+(\w+)`)
	case "java":
		typePattern = regexp.MustCompile(`^(?:public\s+)?(?:class|interface|enum)\s+(\w+)`)
	case "rust":
		typePattern = regexp.MustCompile(`^(?:pub\s+)?(?:struct|enum|trait)\s+(\w+)`)
	default:
		return typeInfos
	}

	braceCount := 0
	var currentType *TypeInfo

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		if currentType == nil {
			if matches := typePattern.FindStringSubmatch(trimmed); matches != nil {
				name := matches[1]
				kind := "type"
				if len(matches) > 2 {
					kind = matches[2]
				} else if strings.Contains(trimmed, "class") {
					kind = "class"
				} else if strings.Contains(trimmed, "interface") {
					kind = "interface"
				} else if strings.Contains(trimmed, "enum") {
					kind = "enum"
				} else if strings.Contains(trimmed, "struct") {
					kind = "struct"
				} else if strings.Contains(trimmed, "trait") {
					kind = "trait"
				}

				currentType = &TypeInfo{
					Name:       name,
					Kind:       kind,
					StartLine:  lineNum,
					IsExported: isExported(name, language),
				}
				braceCount = strings.Count(line, "{") - strings.Count(line, "}")

				// Single-line type
				if braceCount <= 0 && strings.Contains(line, "}") {
					currentType.EndLine = lineNum
					typeInfos = append(typeInfos, *currentType)
					currentType = nil
				}
			}
		} else {
			// Track braces to find type end
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Extract fields (simplified)
			if strings.Contains(trimmed, " ") && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "func") {
				parts := strings.Fields(trimmed)
				if len(parts) >= 1 {
					fieldName := strings.TrimSuffix(parts[0], ":")
					if fieldName != "{" && fieldName != "}" && !strings.HasPrefix(fieldName, "//") {
						currentType.Fields = append(currentType.Fields, fieldName)
					}
				}
			}

			if braceCount <= 0 {
				currentType.EndLine = lineNum
				typeInfos = append(typeInfos, *currentType)
				currentType = nil
			}
		}
	}

	// Handle unclosed type at end of file
	if currentType != nil {
		currentType.EndLine = len(lines)
		typeInfos = append(typeInfos, *currentType)
	}

	return typeInfos
}

// extractFunctions extracts function definitions with metrics.
func (s *SummaryAnalyzer) extractFunctions(lines []string, language string) []FunctionInfo {
	var functions []FunctionInfo

	var funcPattern *regexp.Regexp
	switch language {
	case "go":
		funcPattern = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(([^)]*)\)`)
	case "python":
		funcPattern = regexp.MustCompile(`^def\s+(\w+)\s*\(([^)]*)\)`)
	case "javascript", "typescript":
		funcPattern = regexp.MustCompile(`(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>))\s*\(?([^)]*)\)?`)
	case "java":
		funcPattern = regexp.MustCompile(`(?:public|private|protected)?\s*(?:static)?\s*\w+\s+(\w+)\s*\(([^)]*)\)`)
	case "rust":
		funcPattern = regexp.MustCompile(`^(?:pub\s+)?fn\s+(\w+)\s*\(([^)]*)\)`)
	default:
		return functions
	}

	braceCount := 0
	var currentFunc *FunctionInfo

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		if currentFunc == nil {
			if matches := funcPattern.FindStringSubmatch(trimmed); matches != nil {
				name := ""
				params := ""
				for j, m := range matches[1:] {
					if m != "" {
						if name == "" {
							name = m
						} else if j > 0 {
							params = m
						}
					}
				}

				if name != "" {
					// Count parameters
					paramCount := 0
					if params != "" {
						paramCount = len(strings.Split(params, ","))
					}

					currentFunc = &FunctionInfo{
						Name:       name,
						StartLine:  lineNum,
						Signature:  trimmed,
						IsExported: isExported(name, language),
						Parameters: paramCount,
					}
					braceCount = strings.Count(line, "{") - strings.Count(line, "}")

					// Python: no braces, use indentation
					if language == "python" {
						braceCount = 1 // Treat as open
					}
				}
			}
		} else {
			if language == "python" {
				// Python: function ends when indentation returns to base level
				if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
					currentFunc.EndLine = lineNum - 1
					currentFunc.LineCount = currentFunc.EndLine - currentFunc.StartLine + 1
					currentFunc.Complexity = estimateComplexity(lines[currentFunc.StartLine-1 : currentFunc.EndLine])
					functions = append(functions, *currentFunc)
					currentFunc = nil
					// Check if this line starts a new function
					if matches := funcPattern.FindStringSubmatch(trimmed); matches != nil {
						name := ""
						params := ""
						for j, m := range matches[1:] {
							if m != "" {
								if name == "" {
									name = m
								} else if j > 0 {
									params = m
								}
							}
						}
						if name != "" {
							paramCount := 0
							if params != "" {
								paramCount = len(strings.Split(params, ","))
							}
							currentFunc = &FunctionInfo{
								Name:       name,
								StartLine:  lineNum,
								Signature:  trimmed,
								IsExported: isExported(name, language),
								Parameters: paramCount,
							}
							braceCount = 1
						}
					}
					continue
				}
			} else {
				// Track braces
				braceCount += strings.Count(line, "{") - strings.Count(line, "}")
				if braceCount <= 0 {
					currentFunc.EndLine = lineNum
					currentFunc.LineCount = currentFunc.EndLine - currentFunc.StartLine + 1
					currentFunc.Complexity = estimateComplexity(lines[currentFunc.StartLine-1 : currentFunc.EndLine])
					functions = append(functions, *currentFunc)
					currentFunc = nil
				}
			}
		}
	}

	// Handle unclosed function
	if currentFunc != nil {
		currentFunc.EndLine = len(lines)
		currentFunc.LineCount = currentFunc.EndLine - currentFunc.StartLine + 1
		currentFunc.Complexity = estimateComplexity(lines[currentFunc.StartLine-1:])
		functions = append(functions, *currentFunc)
	}

	// Sort by line number
	sort.Slice(functions, func(i, j int) bool {
		return functions[i].StartLine < functions[j].StartLine
	})

	return functions
}

// estimateComplexity estimates cyclomatic complexity from code lines.
func estimateComplexity(lines []string) int {
	complexity := 1 // Base complexity

	decisionPatterns := []string{
		`\bif\b`, `\belse if\b`, `\belif\b`, `\bfor\b`, `\bwhile\b`,
		`\bcase\b`, `\bcatch\b`, `\bexcept\b`, `&&`, `\|\|`, `\?`,
	}

	for _, line := range lines {
		for _, pattern := range decisionPatterns {
			re := regexp.MustCompile(pattern)
			complexity += len(re.FindAllString(line, -1))
		}
	}

	return complexity
}

// isExported checks if a symbol is exported (public) based on language conventions.
func isExported(name, language string) bool {
	if name == "" {
		return false
	}
	switch language {
	case "go":
		// Go: exported if starts with uppercase
		return name[0] >= 'A' && name[0] <= 'Z'
	case "python":
		// Python: private if starts with _
		return !strings.HasPrefix(name, "_")
	case "javascript", "typescript":
		// JS/TS: typically explicit, assume exported unless starts with _
		return !strings.HasPrefix(name, "_")
	default:
		return true
	}
}

// extractExports extracts exported symbols from the summary.
func (s *SummaryAnalyzer) extractExports(summary *FileSummary, language string) []ExportInfo {
	var exports []ExportInfo

	// Add exported functions
	for _, fn := range summary.Functions {
		if fn.IsExported {
			exports = append(exports, ExportInfo{
				Name:      fn.Name,
				Kind:      "function",
				Line:      fn.StartLine,
				Signature: fn.Signature,
			})
		}
	}

	// Add exported types
	for _, t := range summary.Types {
		if t.IsExported {
			exports = append(exports, ExportInfo{
				Name: t.Name,
				Kind: t.Kind,
				Line: t.StartLine,
			})
		}
	}

	// Sort by line number
	sort.Slice(exports, func(i, j int) bool {
		return exports[i].Line < exports[j].Line
	})

	return exports
}

// AnalyzeFileFromChunks creates a summary using indexed chunks and symbols.
func (s *SummaryAnalyzer) AnalyzeFileFromChunks(filePath string, chunks []*types.Chunk, symbols []*types.Symbol) (*FileSummary, error) {
	// Start with basic file analysis
	summary, err := s.AnalyzeFile(filePath)
	if err != nil {
		return nil, err
	}

	// Enhance with indexed data
	if len(symbols) > 0 {
		// Clear extracted functions/types and use indexed symbols
		summary.Functions = nil
		summary.Types = nil

		for _, sym := range symbols {
			switch sym.Kind {
			case types.SymbolKindFunction, types.SymbolKindMethod:
				summary.Functions = append(summary.Functions, FunctionInfo{
					Name:       sym.Name,
					StartLine:  sym.StartLine,
					EndLine:    sym.EndLine,
					LineCount:  sym.LineCount,
					Signature:  sym.Signature,
					IsExported: sym.Visibility == "public",
				})
			case types.SymbolKindType, types.SymbolKindInterface:
				kind := string(sym.Kind)
				summary.Types = append(summary.Types, TypeInfo{
					Name:       sym.Name,
					Kind:       kind,
					StartLine:  sym.StartLine,
					EndLine:    sym.EndLine,
					IsExported: sym.Visibility == "public",
				})
			}
		}

		// Recalculate exports
		summary.Exports = s.extractExports(summary, summary.Language)
	}

	return summary, nil
}

// QuickSummary returns a quick summary without full parsing.
func (s *SummaryAnalyzer) QuickSummary(filePath string) (*FileSummary, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	language := detectLanguageFromPath(filePath)
	summary := &FileSummary{
		FilePath:   filePath,
		FileName:   filepath.Base(filePath),
		Language:   language,
		Size:       fileInfo.Size(),
		ModifiedAt: fileInfo.ModTime(),
	}

	// Quick line count
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		summary.TotalLines++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			summary.BlankLines++
		} else if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			summary.CommentLines++
		} else {
			summary.CodeLines++
		}
	}

	return summary, nil
}
