// Package analysis provides comprehensive tests for language-specific parsing.
// Tests chunking, symbol extraction, reference extraction, and dead code detection.
package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spetr/mcp-codewizard/builtin/chunking/treesitter"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// TestLanguage defines test expectations for a language.
type TestLanguage struct {
	Name       string
	Extensions []string
	LangID     string // Language ID for treesitter
	Files      []string

	// Expected symbols - map of symbol name to expected kind
	ExpectedSymbols map[string]types.SymbolKind

	// Expected entry points
	EntryPoints []string

	// Expected exported symbols
	ExportedSymbols []string

	// Expected dead code (functions/types that should be detected as unused)
	DeadCodeSymbols []string

	// Expected test functions (should be excluded from dead code)
	TestFunctions []string

	// Expected references - map of caller to callees
	ExpectedReferences map[string][]string
}

// testDataDir returns the path to test data directory.
func testDataDir() string {
	// Find project root
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "testdata", "languages")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// detectLanguage detects language from file extension
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
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
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".cs":
		return "csharp"
	case ".c":
		return "c"
	case ".h":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".hpp":
		return "cpp"
	case ".pas", ".dpr", ".pp":
		return "pascal"
	case ".vb":
		return "vbnet"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".kt", ".kts":
		return "kotlin"
	case ".swift":
		return "swift"
	default:
		return ""
	}
}

// getTestLanguages returns all test language configurations.
func getTestLanguages() []TestLanguage {
	return []TestLanguage{
		{
			Name:       "Go",
			Extensions: []string{".go"},
			LangID:     "go",
			Files: []string{
				"go/main.go",
				"go/types.go",
				"go/utils.go",
				"go/complex.go",
				"go/utils_test.go",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":           types.SymbolKindFunction,
				"Initialize":     types.SymbolKindFunction,
				"LoadConfig":     types.SymbolKindFunction,
				"Config":         types.SymbolKindType,
				"Server":         types.SymbolKindType,
				"Logger":         types.SymbolKindType,
				"NewServer":      types.SymbolKindFunction,
				"NewLogger":      types.SymbolKindFunction,
				"ProcessData":    types.SymbolKindFunction,
				"Handler":        types.SymbolKindType, // interface parsed as type
				"SimpleFunction": types.SymbolKindFunction,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"Initialize", "LoadConfig", "Config", "Server", "Logger",
				"NewServer", "NewLogger", "ProcessData", "Handler",
			},
			DeadCodeSymbols: []string{
				"unusedEntryHelper", "anotherUnused",
				"deadChainStart", "deadChainMiddle", "deadChainEnd",
				"HashString", "FilterStrings", "NewCache",
				"UnusedType", "NewEchoHandler",
			},
			TestFunctions: []string{
				"TestProcessData", "TestHashString", "TestFilterStrings",
				"BenchmarkProcessData", "ExampleProcessData",
				"TestMain", "FuzzProcessData",
			},
			ExpectedReferences: map[string][]string{
				"main":       {"LoadConfig", "Initialize", "NewServer", "ProcessData"},
				"Initialize": {"setupLogging"},
			},
		},
		{
			Name:       "Python",
			Extensions: []string{".py"},
			LangID:     "python",
			Files: []string{
				"python/main.py",
				"python/models.py",
				"python/utils.py",
				"python/complex.py",
				"python/test_utils.py",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":           types.SymbolKindFunction,
				"load_config":    types.SymbolKindFunction,
				"initialize":     types.SymbolKindFunction,
				"Config":         types.SymbolKindType,
				"Server":         types.SymbolKindType,
				"Logger":         types.SymbolKindType,
				"process_data":   types.SymbolKindFunction,
				"simple_function": types.SymbolKindFunction,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"load_config", "initialize", "Config", "Server", "Logger",
				"process_data",
			},
			DeadCodeSymbols: []string{
				"unused_function", "another_unused",
				"dead_chain_start", "dead_chain_middle", "dead_chain_end",
				"hash_string", "filter_strings",
				"UnusedClass", "EchoHandler",
			},
			TestFunctions: []string{
				"test_process_data", "test_hash_string", "test_filter_strings",
				"TestProcessData", "TestHashString",
			},
			ExpectedReferences: map[string][]string{
				"main": {"load_config", "initialize", "process_data"},
			},
		},
		{
			Name:       "JavaScript",
			Extensions: []string{".js", ".mjs"},
			LangID:     "javascript",
			Files: []string{
				"javascript/index.js",
				"javascript/models.js",
				"javascript/utils.js",
				"javascript/utils.test.js",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":        types.SymbolKindFunction,
				"loadConfig":  types.SymbolKindFunction,
				"initialize":  types.SymbolKindFunction,
				"Server":      types.SymbolKindType,
				"Logger":      types.SymbolKindType,
				"processData": types.SymbolKindFunction,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"loadConfig", "initialize", "Server", "Logger",
				"processData", "formatOutput",
			},
			DeadCodeSymbols: []string{
				"unusedFunction", "anotherUnused",
				"deadChainStart", "deadChainMiddle", "deadChainEnd",
				"hashString", "filterItems",
				"UnusedClass", "EchoHandler",
			},
			TestFunctions: []string{
				// Jest/Mocha style tests
			},
			ExpectedReferences: map[string][]string{
				"main": {"loadConfig", "initialize", "processData"},
			},
		},
		{
			Name:       "TypeScript",
			Extensions: []string{".ts"},
			LangID:     "typescript",
			Files: []string{
				"typescript/index.ts",
				"typescript/types.ts",
				"typescript/models.ts",
				"typescript/utils.ts",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":        types.SymbolKindFunction,
				"loadConfig":  types.SymbolKindFunction,
				"initialize":  types.SymbolKindFunction,
				"processData": types.SymbolKindFunction,
				"formatOutput": types.SymbolKindFunction,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"loadConfig", "initialize",
				"processData", "formatOutput",
			},
			DeadCodeSymbols: []string{
				"unusedFunction", "anotherUnused",
				"deadChainStart", "deadChainMiddle", "deadChainEnd",
				"hashString", "filterItems",
				"UnusedClass", "EchoHandler",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"main": {"loadConfig", "initialize", "processData"},
			},
		},
		{
			Name:       "Java",
			Extensions: []string{".java"},
			LangID:     "java",
			Files: []string{
				"java/Main.java",
				"java/Models.java",
				"java/Utils.java",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"Main":        types.SymbolKindType,
				"Server":      types.SymbolKindType,
				"Logger":      types.SymbolKindType,
				"Utils":       types.SymbolKindType,
				"Handler":     types.SymbolKindInterface,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"Main", "Server", "Logger", "Utils",
			},
			DeadCodeSymbols: []string{
				"unusedMethod", "anotherUnused",
				"deadChainStart", "deadChainMiddle", "deadChainEnd",
				"hashString", "filterItems",
				"UnusedClass", "EchoHandler",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"main": {"loadConfig", "initialize", "processData"},
			},
		},
		{
			Name:       "Rust",
			Extensions: []string{".rs"},
			LangID:     "rust",
			Files: []string{
				"rust/main.rs",
				"rust/models.rs",
				"rust/utils.rs",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":         types.SymbolKindFunction,
				"load_config":  types.SymbolKindFunction,
				"initialize":   types.SymbolKindFunction,
				"Server":       types.SymbolKindType,
				"Handler":      types.SymbolKindInterface, // trait parsed as interface
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"load_config", "initialize", "Server",
				"Handler",
			},
			DeadCodeSymbols: []string{
				"unused_function", "another_unused",
				"dead_chain_start", "dead_chain_middle", "dead_chain_end",
				"hash_string", "filter_items",
				"UnusedStruct", "EchoHandler",
			},
			TestFunctions: []string{
				"test_process_data", "test_hash_string", "test_filter_strings",
			},
			ExpectedReferences: map[string][]string{
				"main": {"load_config", "initialize"},
			},
		},
		{
			Name:       "C#",
			Extensions: []string{".cs"},
			LangID:     "csharp",
			Files: []string{
				"csharp/Program.cs",
				"csharp/Models.cs",
				"csharp/Utils.cs",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"Program":     types.SymbolKindType,
				"Server":      types.SymbolKindType,
				"Utils":       types.SymbolKindType,
				"IHandler":    types.SymbolKindInterface,
			},
			EntryPoints: []string{"Main"},
			ExportedSymbols: []string{
				"Program", "Server", "Utils",
			},
			DeadCodeSymbols: []string{
				"UnusedMethod", "AnotherUnused",
				"DeadChainStart", "DeadChainMiddle", "DeadChainEnd",
				"HashString", "FilterItems",
				"UnusedClass", "EchoHandler",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"Main": {"LoadConfig", "Initialize", "ProcessData"},
			},
		},
		{
			Name:       "C",
			Extensions: []string{".c", ".h"},
			LangID:     "c",
			Files: []string{
				"c/main.c",
				"c/utils.h",
				"c/utils.c",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":         types.SymbolKindFunction,
				"load_config":  types.SymbolKindFunction,
				"initialize":   types.SymbolKindFunction,
				"Server":       types.SymbolKindType,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"load_config", "initialize", "Server",
			},
			DeadCodeSymbols: []string{
				"unused_function", "another_unused",
				"dead_chain_start", "dead_chain_middle", "dead_chain_end",
				"hash_string", "filter_items",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"main": {"load_config", "initialize", "process_data"},
			},
		},
		{
			Name:       "C++",
			Extensions: []string{".cpp", ".hpp", ".cc"},
			LangID:     "cpp",
			Files: []string{
				"cpp/main.cpp",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":          types.SymbolKindFunction,
				"loadConfig":    types.SymbolKindFunction,
				"initialize":    types.SymbolKindFunction,
				"processData":   types.SymbolKindFunction,
				"formatOutput":  types.SymbolKindFunction,
				"IHandler":      types.SymbolKindType,
				"Container":     types.SymbolKindType,
				"EchoHandler":   types.SymbolKindType,
				"UpperHandler":  types.SymbolKindType,
				"Cache":         types.SymbolKindType,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"loadConfig", "initialize",
				"processData", "formatOutput", "IHandler",
			},
			DeadCodeSymbols: []string{
				"unusedFunction", "anotherUnused",
				"deadChainStart", "deadChainMiddle", "deadChainEnd",
				"makeAdder", "applyTwice",
				"EchoHandler", "UpperHandler", "Cache", "Pair",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"main": {"loadConfig", "initialize", "processData", "runPipeline"},
			},
		},
		{
			Name:       "Pascal",
			Extensions: []string{".pas", ".dpr", ".pp"},
			LangID:     "pascal",
			Files: []string{
				"pascal/main.pas",
				"pascal/utils.pas",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"Main":         types.SymbolKindFunction,
				"Initialize":   types.SymbolKindFunction,
				"SetupLogging": types.SymbolKindFunction,
				"RunPipeline":  types.SymbolKindFunction,
			},
			EntryPoints: []string{"Main"},
			ExportedSymbols: []string{
				"LoadConfig", "ProcessData", "FormatOutput",
			},
			DeadCodeSymbols: []string{
				"UnusedProcedure", "AnotherUnused",
				"DeadChainStart", "DeadChainMiddle", "DeadChainEnd",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"Main": {"LoadConfig", "Initialize", "ProcessData"},
			},
		},
		{
			Name:       "VB.NET",
			Extensions: []string{".vb"},
			LangID:     "vbnet",
			Files: []string{
				"vbnet/Program.vb",
				"vbnet/Models.vb",
				"vbnet/Utils.vb",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"Main":         types.SymbolKindFunction,
				"LoadConfig":   types.SymbolKindFunction,
				"Initialize":   types.SymbolKindFunction,
				"SetupLogging": types.SymbolKindFunction,
				"RunPipeline":  types.SymbolKindFunction,
				"Config":       types.SymbolKindType,
				"Server":       types.SymbolKindType,
				"Logger":       types.SymbolKindType,
				"IHandler":     types.SymbolKindInterface,
			},
			EntryPoints: []string{"Main"},
			ExportedSymbols: []string{
				"Config", "Server", "Logger", "IHandler",
				"ProcessData", "FormatOutput",
			},
			DeadCodeSymbols: []string{
				"UnusedFunction", "AnotherUnused",
				"DeadChainStart", "DeadChainMiddle", "DeadChainEnd",
				"HashString", "EchoHandler", "UpperHandler", "Cache",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"Main": {"LoadConfig", "Initialize", "RunPipeline"},
			},
		},
		// Note: PHP is skipped because the PHP chunker is designed for embedded JS extraction.
		// Test data exists in testdata/languages/php/ for future testing.
		{
			Name:       "Ruby",
			Extensions: []string{".rb"},
			LangID:     "ruby",
			Files: []string{
				"ruby/main.rb",
				"ruby/models.rb",
				"ruby/utils.rb",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":           types.SymbolKindFunction,
				"load_config":    types.SymbolKindFunction,
				"initialize_app": types.SymbolKindFunction,
				"Config":         types.SymbolKindType,
				"Server":         types.SymbolKindType,
				"Logger":         types.SymbolKindType,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"Config", "Server", "Logger",
			},
			DeadCodeSymbols: []string{
				"unused_method", "another_unused",
				"dead_chain_start", "dead_chain_middle", "dead_chain_end",
				"hash_string", "EchoHandler", "UpperHandler", "Cache",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"main": {"load_config", "initialize_app"},
			},
		},
		{
			Name:       "Kotlin",
			Extensions: []string{".kt", ".kts"},
			LangID:     "kotlin",
			Files: []string{
				"kotlin/Main.kt",
				"kotlin/Models.kt",
				"kotlin/Utils.kt",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":       types.SymbolKindFunction,
				"loadConfig": types.SymbolKindFunction,
				"initialize": types.SymbolKindFunction,
				"Config":     types.SymbolKindType,
				"Server":     types.SymbolKindType,
				"Logger":     types.SymbolKindType,
				"Handler":    types.SymbolKindType, // interface parsed as type
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"Config", "Server", "Logger", "Handler",
				"processData", "formatOutput",
			},
			DeadCodeSymbols: []string{
				"unusedFunction", "anotherUnused",
				"deadChainStart", "deadChainMiddle", "deadChainEnd",
				"hashString", "EchoHandler", "UpperHandler", "Cache",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"main": {"loadConfig", "initialize", "processData"},
			},
		},
		{
			Name:       "Swift",
			Extensions: []string{".swift"},
			LangID:     "swift",
			Files: []string{
				"swift/main.swift",
				"swift/Models.swift",
				"swift/Utils.swift",
			},
			ExpectedSymbols: map[string]types.SymbolKind{
				"main":        types.SymbolKindFunction,
				"loadConfig":  types.SymbolKindFunction,
				"initialize":  types.SymbolKindFunction,
				"Config":      types.SymbolKindType,
				"Server":      types.SymbolKindType,
				"Logger":      types.SymbolKindType,
				"Handler":     types.SymbolKindInterface,
			},
			EntryPoints: []string{"main"},
			ExportedSymbols: []string{
				"Config", "Server", "Logger", "Handler",
				"processData", "formatOutput",
			},
			DeadCodeSymbols: []string{
				"unusedFunction", "anotherUnused",
				"deadChainStart", "deadChainMiddle", "deadChainEnd",
				"hashString", "EchoHandler", "UpperHandler", "Cache",
			},
			TestFunctions: []string{},
			ExpectedReferences: map[string][]string{
				"main": {"loadConfig", "initialize", "processData"},
			},
		},
	}
}

// TestChunking tests that chunking works correctly for each language.
func TestChunking(t *testing.T) {
	baseDir := testDataDir()
	if baseDir == "" {
		t.Skip("Could not find test data directory")
	}

	chunker := treesitter.New(treesitter.Config{MaxChunkSize: 2000})
	defer chunker.Close()

	for _, lang := range getTestLanguages() {
		t.Run(lang.Name, func(t *testing.T) {
			for _, file := range lang.Files {
				filePath := filepath.Join(baseDir, file)

				// Skip if file doesn't exist
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Logf("Skipping missing file: %s", filePath)
					continue
				}

				// Read file
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file %s: %v", filePath, err)
				}

				// Create source file
				sourceFile := &types.SourceFile{
					Path:     filePath,
					Content:  content,
					Language: detectLanguage(filePath),
				}

				// Chunk the file
				chunks, err := chunker.Chunk(sourceFile)
				if err != nil {
					t.Fatalf("Failed to chunk file %s: %v", filePath, err)
				}

				// Verify we got chunks
				if len(chunks) == 0 {
					t.Errorf("No chunks produced for file %s", file)
				}

				// Log chunk count for debugging
				t.Logf("File %s: %d chunks", file, len(chunks))

				// Verify chunks have content
				for i, chunk := range chunks {
					if chunk.Content == "" {
						t.Errorf("Chunk %d in file %s has empty content", i, file)
					}
					if chunk.StartLine == 0 {
						t.Errorf("Chunk %d in file %s has invalid start line", i, file)
					}
				}
			}
		})
	}
}

// TestSymbolExtraction tests that symbols are extracted correctly.
func TestSymbolExtraction(t *testing.T) {
	baseDir := testDataDir()
	if baseDir == "" {
		t.Skip("Could not find test data directory")
	}

	chunker := treesitter.New(treesitter.Config{MaxChunkSize: 2000})
	defer chunker.Close()

	for _, lang := range getTestLanguages() {
		t.Run(lang.Name, func(t *testing.T) {
			allSymbols := make(map[string]*types.Symbol)

			for _, file := range lang.Files {
				filePath := filepath.Join(baseDir, file)

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					continue
				}

				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file %s: %v", filePath, err)
				}

				sourceFile := &types.SourceFile{
					Path:     filePath,
					Content:  content,
					Language: detectLanguage(filePath),
				}

				// Extract symbols
				symbols, err := chunker.ExtractSymbols(sourceFile)
				if err != nil {
					t.Fatalf("Failed to extract symbols from %s: %v", filePath, err)
				}

				// Collect symbols
				for _, sym := range symbols {
					allSymbols[sym.Name] = sym
				}
			}

			// Verify expected symbols exist
			for symName, expectedKind := range lang.ExpectedSymbols {
				sym, found := allSymbols[symName]
				if !found {
					t.Errorf("Expected symbol %q not found", symName)
					continue
				}
				if sym.Kind != expectedKind {
					t.Errorf("Symbol %q: expected kind %v, got %v", symName, expectedKind, sym.Kind)
				}
			}

			// Log found symbols for debugging
			t.Logf("Found %d symbols", len(allSymbols))
			for name, sym := range allSymbols {
				t.Logf("  %s: %s", name, sym.Kind)
			}
		})
	}
}

// TestReferenceExtraction tests that references are extracted correctly.
// Only tests Go and Python where reference extraction is well-supported.
func TestReferenceExtraction(t *testing.T) {
	baseDir := testDataDir()
	if baseDir == "" {
		t.Skip("Could not find test data directory")
	}

	chunker := treesitter.New(treesitter.Config{MaxChunkSize: 2000})
	defer chunker.Close()

	// Only test languages where reference extraction is reliable
	testableLanguages := []string{"Go", "Python"}

	for _, lang := range getTestLanguages() {
		// Skip languages that don't have reliable reference extraction
		isTestable := false
		for _, testable := range testableLanguages {
			if lang.Name == testable {
				isTestable = true
				break
			}
		}
		if !isTestable {
			continue
		}

		t.Run(lang.Name, func(t *testing.T) {
			// Map from function name to its references
			refsByFunction := make(map[string][]string)
			totalRefs := 0

			for _, file := range lang.Files {
				filePath := filepath.Join(baseDir, file)

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					continue
				}

				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file %s: %v", filePath, err)
				}

				sourceFile := &types.SourceFile{
					Path:     filePath,
					Content:  content,
					Language: detectLanguage(filePath),
				}

				// Extract references
				refs, err := chunker.ExtractReferences(sourceFile)
				if err != nil {
					t.Fatalf("Failed to extract references from %s: %v", filePath, err)
				}

				totalRefs += len(refs)

				// Collect references by caller
				for _, ref := range refs {
					if ref.Kind == types.RefKindCall {
						// Extract function name from FromSymbol (e.g., "filepath:funcname:line")
						parts := strings.Split(ref.FromSymbol, ":")
						var callerName string
						if len(parts) >= 2 {
							callerName = parts[len(parts)-2]
						} else {
							callerName = ref.FromSymbol
						}

						// Extract callee name from ToSymbol
						calleeParts := strings.Split(ref.ToSymbol, ":")
						var calleeName string
						if len(calleeParts) >= 1 {
							// Get last part or the whole thing
							calleeName = calleeParts[len(calleeParts)-1]
							// Remove any line number suffix
							if idx := strings.LastIndex(calleeName, ":"); idx > 0 {
								calleeName = calleeName[:idx]
							}
						} else {
							calleeName = ref.ToSymbol
						}

						refsByFunction[callerName] = append(refsByFunction[callerName], calleeName)
					}
				}
			}

			// Just verify we extracted some references
			if totalRefs == 0 {
				t.Error("No references extracted")
			}

			// Log found references for debugging
			t.Logf("Found %d total references from %d functions", totalRefs, len(refsByFunction))
		})
	}
}

// TestExportedSymbols tests that exported/public symbols are correctly identified.
func TestExportedSymbols(t *testing.T) {
	baseDir := testDataDir()
	if baseDir == "" {
		t.Skip("Could not find test data directory")
	}

	chunker := treesitter.New(treesitter.Config{MaxChunkSize: 2000})
	defer chunker.Close()

	for _, lang := range getTestLanguages() {
		t.Run(lang.Name, func(t *testing.T) {
			exportedFound := make(map[string]bool)

			for _, file := range lang.Files {
				filePath := filepath.Join(baseDir, file)

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					continue
				}

				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file %s: %v", filePath, err)
				}

				sourceFile := &types.SourceFile{
					Path:     filePath,
					Content:  content,
					Language: detectLanguage(filePath),
				}

				symbols, err := chunker.ExtractSymbols(sourceFile)
				if err != nil {
					t.Fatalf("Failed to extract symbols: %v", err)
				}

				for _, sym := range symbols {
					if sym.Visibility == "public" || sym.Visibility == "exported" {
						exportedFound[sym.Name] = true
					}
				}
			}

			// Verify expected exported symbols
			for _, expectedExport := range lang.ExportedSymbols {
				if !exportedFound[expectedExport] {
					t.Errorf("Expected %q to be exported, but it was not found as public", expectedExport)
				}
			}

			t.Logf("Found %d exported symbols", len(exportedFound))
		})
	}
}

// TestTestFunctionDetection tests that test functions are correctly identified.
// Only tests Go where test function detection is reliable.
func TestTestFunctionDetection(t *testing.T) {
	baseDir := testDataDir()
	if baseDir == "" {
		t.Skip("Could not find test data directory")
	}

	chunker := treesitter.New(treesitter.Config{MaxChunkSize: 2000})
	defer chunker.Close()

	// Only test Go where test function detection is reliable
	for _, lang := range getTestLanguages() {
		if lang.Name != "Go" {
			continue
		}

		t.Run(lang.Name, func(t *testing.T) {
			testFuncsFound := make(map[string]bool)

			for _, file := range lang.Files {
				filePath := filepath.Join(baseDir, file)

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					continue
				}

				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file %s: %v", filePath, err)
				}

				sourceFile := &types.SourceFile{
					Path:     filePath,
					Content:  content,
					Language: detectLanguage(filePath),
				}

				symbols, err := chunker.ExtractSymbols(sourceFile)
				if err != nil {
					t.Fatalf("Failed to extract symbols: %v", err)
				}

				for _, sym := range symbols {
					if sym.Kind == types.SymbolKindFunction {
						// Check if it's a test function based on naming convention
						isTest := false
						name := sym.Name

						isTest = strings.HasPrefix(name, "Test") ||
							strings.HasPrefix(name, "Benchmark") ||
							strings.HasPrefix(name, "Example") ||
							strings.HasPrefix(name, "Fuzz")

						if isTest {
							testFuncsFound[name] = true
						}
					}
				}
			}

			// Verify we found some test functions
			if len(testFuncsFound) == 0 {
				t.Error("No test functions found")
			}

			// Check for specific expected test functions
			expectedTests := []string{"TestProcessData", "BenchmarkProcessData", "ExampleProcessData"}
			for _, expected := range expectedTests {
				if !testFuncsFound[expected] {
					t.Errorf("Expected test function %q not found", expected)
				}
			}

			t.Logf("Found %d test functions", len(testFuncsFound))
			for name := range testFuncsFound {
				t.Logf("  %s", name)
			}
		})
	}
}

// TestComplexityMetrics tests that complexity analysis works correctly.
func TestComplexityMetrics(t *testing.T) {
	baseDir := testDataDir()
	if baseDir == "" {
		t.Skip("Could not find test data directory")
	}

	analyzer := NewComplexityAnalyzer()

	// Test Go complexity file
	t.Run("Go", func(t *testing.T) {
		filePath := filepath.Join(baseDir, "go/complex.go")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Skip("complex.go not found")
		}

		// Test the complexity analysis
		metrics, err := analyzer.AnalyzeFile(filePath)
		if err != nil {
			t.Fatalf("Failed to get complexity metrics: %v", err)
		}

		// Verify we got metrics
		if len(metrics.Functions) == 0 {
			t.Error("No function metrics found")
		}

		// Log metrics for debugging
		t.Logf("File max complexity: %d", metrics.MaxComplexity)
		t.Logf("Functions: %d", len(metrics.Functions))

		for _, fn := range metrics.Functions {
			t.Logf("  %s: complexity=%d, depth=%d", fn.FunctionName, fn.CyclomaticComplexity, fn.MaxNestingDepth)
		}

		// Check that SimpleFunction has low complexity
		var simpleFound, complexFound bool
		for _, fn := range metrics.Functions {
			if fn.FunctionName == "SimpleFunction" {
				simpleFound = true
				if fn.CyclomaticComplexity > 2 {
					t.Errorf("SimpleFunction should have low complexity, got %d", fn.CyclomaticComplexity)
				}
			}
			// Use ExtremelyComplexFunction as the complex function test
			if fn.FunctionName == "ExtremelyComplexFunction" {
				complexFound = true
				if fn.CyclomaticComplexity < 10 {
					t.Errorf("ExtremelyComplexFunction should have high complexity, got %d", fn.CyclomaticComplexity)
				}
			}
		}

		if !simpleFound {
			t.Error("SimpleFunction not found in metrics")
		}
		if !complexFound {
			t.Error("ExtremelyComplexFunction not found in metrics")
		}
	})

	// Test Python complexity file
	t.Run("Python", func(t *testing.T) {
		filePath := filepath.Join(baseDir, "python/complex.py")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Skip("complex.py not found")
		}

		metrics, err := analyzer.AnalyzeFile(filePath)
		if err != nil {
			t.Fatalf("Failed to get complexity metrics: %v", err)
		}

		// Verify we got metrics
		if len(metrics.Functions) == 0 {
			t.Error("No function metrics found")
		}

		// Log metrics for debugging
		t.Logf("File max complexity: %d", metrics.MaxComplexity)
		t.Logf("Functions: %d", len(metrics.Functions))

		for _, fn := range metrics.Functions {
			t.Logf("  %s: complexity=%d, depth=%d", fn.FunctionName, fn.CyclomaticComplexity, fn.MaxNestingDepth)
		}
	})
}

// TestLanguageSupport tests that all expected languages are supported.
func TestLanguageSupport(t *testing.T) {
	chunker := treesitter.New(treesitter.Config{MaxChunkSize: 2000})
	defer chunker.Close()

	supportedLanguages := chunker.SupportedLanguages()
	supportedSet := make(map[string]bool)
	for _, lang := range supportedLanguages {
		supportedSet[lang] = true
	}

	expectedLanguages := []string{
		"go", "python", "javascript", "typescript", "java",
		"rust", "csharp", "c", "cpp",
		"pascal", "vbnet", "ruby", "kotlin", "swift",
		// Note: php is supported but skipped in tests (designed for embedded JS extraction)
	}

	for _, lang := range expectedLanguages {
		if !chunker.SupportsLanguage(lang) {
			t.Errorf("Language %q should be supported", lang)
		}
	}

	t.Logf("Supported languages: %v", supportedLanguages)
}
