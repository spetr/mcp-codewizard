// Package mcp implements the MCP server for code search.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// TreeNode represents a node in the project tree.
type TreeNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"` // "file" or "directory"
	Path     string      `json:"path"` // relative path from project root
	Size     int64       `json:"size,omitempty"`
	Language string      `json:"language,omitempty"`
	Indexed  bool        `json:"indexed,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`

	// Aggregated stats for directories
	FileCount    int `json:"file_count,omitempty"`
	IndexedCount int `json:"indexed_count,omitempty"`
}

// TreeResult represents the result of get_project_tree.
type TreeResult struct {
	Root       *TreeNode `json:"root"`
	TotalFiles int       `json:"total_files"`
	TotalDirs  int       `json:"total_dirs"`
	Indexed    int       `json:"indexed_files"`
	Truncated  bool      `json:"truncated,omitempty"`
}

// handleGetProjectTree handles the get_project_tree tool.
func (s *Server) handleGetProjectTree(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pathFilter := req.GetString("path", "")
	maxDepth := req.GetInt("depth", 5)
	includeFiles := req.GetBool("include_files", true)
	includeHidden := req.GetBool("include_hidden", false)
	showIndexed := req.GetBool("show_indexed", true)
	format := req.GetString("format", "json")

	// Determine root path
	rootPath := s.projectDir
	if pathFilter != "" {
		rootPath = filepath.Join(s.projectDir, pathFilter)
		if _, err := os.Stat(rootPath); os.IsNotExist(err) {
			return mcp.NewToolResultError(fmt.Sprintf("path not found: %s", pathFilter)), nil
		}
	}

	// Get indexed files set for quick lookup
	var indexedFiles map[string]bool
	if showIndexed && s.store != nil {
		indexedFiles = s.getIndexedFilesSet()
	}

	// Build tree
	root, stats, err := s.buildTree(ctx, rootPath, "", maxDepth, 0, includeFiles, includeHidden, indexedFiles)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build tree: %v", err)), nil
	}

	result := &TreeResult{
		Root:       root,
		TotalFiles: stats.files,
		TotalDirs:  stats.dirs,
		Indexed:    stats.indexed,
		Truncated:  stats.truncated,
	}

	// Format output
	switch format {
	case "text":
		text := s.formatTreeAsText(root, "", true)
		return mcp.NewToolResultText(text), nil
	case "markdown":
		md := s.formatTreeAsMarkdown(root, 0)
		return mcp.NewToolResultText(md), nil
	default: // json
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	}
}

type treeStats struct {
	files     int
	dirs      int
	indexed   int
	truncated bool
}

// buildTree recursively builds the tree structure.
func (s *Server) buildTree(ctx context.Context, path, relPath string, maxDepth, currentDepth int, includeFiles, includeHidden bool, indexedFiles map[string]bool) (*TreeNode, treeStats, error) {
	if ctx.Err() != nil {
		return nil, treeStats{}, ctx.Err()
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, treeStats{}, err
	}

	name := filepath.Base(path)
	if relPath == "" {
		name = filepath.Base(s.projectDir)
		relPath = "."
	}

	node := &TreeNode{
		Name: name,
		Path: relPath,
	}

	stats := treeStats{}

	if info.IsDir() {
		node.Type = "directory"
		stats.dirs++

		// Check depth limit
		if currentDepth >= maxDepth {
			stats.truncated = true
			return node, stats, nil
		}

		// Read directory entries
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, stats, nil // Skip unreadable directories
		}

		// Sort entries: directories first, then files
		sort.Slice(entries, func(i, j int) bool {
			iDir := entries[i].IsDir()
			jDir := entries[j].IsDir()
			if iDir != jDir {
				return iDir // directories first
			}
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			entryName := entry.Name()

			// Skip hidden files/dirs unless requested
			if !includeHidden && strings.HasPrefix(entryName, ".") {
				continue
			}

			// Skip excluded directories
			entryRelPath := filepath.Join(relPath, entryName)
			if entry.IsDir() && s.shouldExcludeDir(entryRelPath) {
				continue
			}

			// Skip files if not requested
			if !entry.IsDir() && !includeFiles {
				continue
			}

			// Skip non-indexed file types if we're filtering
			if !entry.IsDir() && s.config != nil && len(s.config.Index.Include) > 0 {
				if !s.matchesIncludePatterns(entryRelPath) {
					continue
				}
			}

			childPath := filepath.Join(path, entryName)
			child, childStats, err := s.buildTree(ctx, childPath, entryRelPath, maxDepth, currentDepth+1, includeFiles, includeHidden, indexedFiles)
			if err != nil {
				continue
			}

			node.Children = append(node.Children, child)
			stats.files += childStats.files
			stats.dirs += childStats.dirs
			stats.indexed += childStats.indexed
			if childStats.truncated {
				stats.truncated = true
			}
		}

		// Aggregate counts for directory
		node.FileCount = stats.files
		node.IndexedCount = stats.indexed

	} else {
		node.Type = "file"
		node.Size = info.Size()
		node.Language = detectLanguage(relPath)
		stats.files++

		// Check if indexed
		if indexedFiles != nil {
			absPath := filepath.Join(s.projectDir, relPath)
			if indexedFiles[absPath] {
				node.Indexed = true
				stats.indexed++
			}
		}
	}

	return node, stats, nil
}

// shouldExcludeDir checks if a directory should be excluded.
func (s *Server) shouldExcludeDir(relPath string) bool {
	if s.config == nil {
		return false
	}

	// Check common excludes
	base := filepath.Base(relPath)
	commonExcludes := []string{"node_modules", "vendor", ".git", "dist", "build", "target", "bin", "obj", "__pycache__", ".venv", "venv"}
	for _, exc := range commonExcludes {
		if base == exc {
			return true
		}
	}

	// Check config excludes
	for _, pattern := range s.config.Index.Exclude {
		if matchGlobPath(pattern, relPath+"/") {
			return true
		}
	}

	return false
}

// matchesIncludePatterns checks if a file matches any include pattern.
func (s *Server) matchesIncludePatterns(relPath string) bool {
	if s.config == nil || len(s.config.Index.Include) == 0 {
		return true
	}

	for _, pattern := range s.config.Index.Include {
		if matchGlobPath(pattern, relPath) {
			return true
		}
	}
	return false
}

// getIndexedFilesSet returns a set of indexed file paths.
func (s *Server) getIndexedFilesSet() map[string]bool {
	result := make(map[string]bool)

	// Use GetAllFileHashes which returns map[filePath]hash
	fileHashes, err := s.store.GetAllFileHashes()
	if err != nil {
		return result
	}

	for filePath := range fileHashes {
		result[filePath] = true
	}
	return result
}

// detectLanguage detects the programming language from file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".jsx":
		return "jsx"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "c-header"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala", ".sc":
		return "scala"
	case ".lua":
		return "lua"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "shell"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".vue":
		return "vue"
	case ".svelte":
		return "svelte"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	case ".xml":
		return "xml"
	case ".md":
		return "markdown"
	case ".r", ".R":
		return "r"
	case ".dart":
		return "dart"
	case ".vb":
		return "vbnet"
	case ".ps1", ".psm1", ".psd1":
		return "powershell"
	default:
		return ""
	}
}

// formatTreeAsText formats the tree as ASCII art.
func (s *Server) formatTreeAsText(node *TreeNode, prefix string, isLast bool) string {
	var sb strings.Builder

	// Determine connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	// Format node name
	name := node.Name
	if node.Type == "directory" {
		name += "/"
		if node.FileCount > 0 {
			name += fmt.Sprintf(" (%d files", node.FileCount)
			if node.IndexedCount > 0 {
				name += fmt.Sprintf(", %d indexed", node.IndexedCount)
			}
			name += ")"
		}
	} else {
		if node.Indexed {
			name += " ✓"
		}
	}

	sb.WriteString(prefix + connector + name + "\n")

	// Process children
	if len(node.Children) > 0 {
		childPrefix := prefix
		if prefix != "" {
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		}

		for i, child := range node.Children {
			isChildLast := i == len(node.Children)-1
			sb.WriteString(s.formatTreeAsText(child, childPrefix, isChildLast))
		}
	}

	return sb.String()
}

// formatTreeAsMarkdown formats the tree as Markdown.
func (s *Server) formatTreeAsMarkdown(node *TreeNode, depth int) string {
	var sb strings.Builder

	indent := strings.Repeat("  ", depth)

	if node.Type == "directory" {
		icon := "📁"
		sb.WriteString(fmt.Sprintf("%s- %s **%s/**", indent, icon, node.Name))
		if node.FileCount > 0 {
			sb.WriteString(fmt.Sprintf(" *(%d files)*", node.FileCount))
		}
		sb.WriteString("\n")

		for _, child := range node.Children {
			sb.WriteString(s.formatTreeAsMarkdown(child, depth+1))
		}
	} else {
		icon := "📄"
		if node.Language != "" {
			icon = getLanguageIcon(node.Language)
		}
		sb.WriteString(fmt.Sprintf("%s- %s %s", indent, icon, node.Name))
		if node.Indexed {
			sb.WriteString(" ✅")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// getLanguageIcon returns an emoji icon for a language.
func getLanguageIcon(lang string) string {
	switch lang {
	case "go":
		return "🐹"
	case "python":
		return "🐍"
	case "javascript", "typescript", "jsx", "tsx":
		return "📜"
	case "rust":
		return "🦀"
	case "java":
		return "☕"
	case "ruby":
		return "💎"
	case "c", "cpp", "c-header":
		return "⚙️"
	case "shell":
		return "🐚"
	case "html":
		return "🌐"
	case "css":
		return "🎨"
	case "markdown":
		return "📝"
	case "json", "yaml", "toml":
		return "📋"
	default:
		return "📄"
	}
}
