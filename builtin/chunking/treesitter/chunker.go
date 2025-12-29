// Package treesitter implements chunking using Tree-sitter for AST-aware splitting.
package treesitter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	tsc "github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/cue"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/groovy"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	tsmarkdown "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown"
	"github.com/smacker/go-tree-sitter/ocaml"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/svelte"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	tstype "github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"

	"github.com/spetr/mcp-codewizard/builtin/chunking/treesitter/pascal"
	"github.com/spetr/mcp-codewizard/builtin/chunking/treesitter/vbnet"
	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// Default values
const (
	DefaultMaxChunkSize = 2000 // tokens
	CharsPerToken       = 4    // rough approximation
)

// Config contains configuration for TreeSitter chunking.
type Config struct {
	MaxChunkSize int // Maximum chunk size in tokens
}

// Chunker implements AST-aware chunking using Tree-sitter.
type Chunker struct {
	config  Config
	parsers map[string]*sitter.Parser
}

// New creates a new TreeSitter chunker.
func New(cfg Config) *Chunker {
	if cfg.MaxChunkSize == 0 {
		cfg.MaxChunkSize = DefaultMaxChunkSize
	}

	return &Chunker{
		config:  cfg,
		parsers: make(map[string]*sitter.Parser),
	}
}

// Name returns the strategy name.
func (c *Chunker) Name() string {
	return "treesitter"
}

// getParser returns a parser for the given language.
func (c *Chunker) getParser(lang string) (*sitter.Parser, *sitter.Language, bool) {
	var language *sitter.Language

	switch lang {
	case "go":
		language = golang.GetLanguage()
	case "python":
		language = python.GetLanguage()
	case "javascript", "jsx":
		language = javascript.GetLanguage()
	case "typescript":
		language = tstype.GetLanguage()
	case "tsx":
		language = tsx.GetLanguage()
	case "rust":
		language = rust.GetLanguage()
	case "java":
		language = java.GetLanguage()
	case "c", "h":
		language = tsc.GetLanguage()
	case "cpp", "hpp", "cc", "cxx":
		language = cpp.GetLanguage()
	case "ruby", "rb":
		language = ruby.GetLanguage()
	case "php":
		language = php.GetLanguage()
	case "csharp", "cs":
		language = csharp.GetLanguage()
	case "kotlin", "kt", "kts":
		language = kotlin.GetLanguage()
	case "swift":
		language = swift.GetLanguage()
	case "scala", "sc":
		language = scala.GetLanguage()
	case "html", "htm", "xhtml":
		language = html.GetLanguage()
	case "svelte":
		language = svelte.GetLanguage()
	case "lua":
		language = lua.GetLanguage()
	case "sql":
		language = sql.GetLanguage()
	case "proto", "protobuf":
		language = protobuf.GetLanguage()
	case "markdown", "md":
		language = tsmarkdown.GetLanguage()
	case "bash", "sh", "shell":
		language = bash.GetLanguage()
	case "css":
		language = css.GetLanguage()
	case "dockerfile":
		language = dockerfile.GetLanguage()
	case "yaml", "yml":
		language = yaml.GetLanguage()
	case "hcl", "tf", "terraform":
		language = hcl.GetLanguage()
	case "elixir", "ex", "exs":
		language = elixir.GetLanguage()
	case "elm":
		language = elm.GetLanguage()
	case "groovy", "gradle":
		language = groovy.GetLanguage()
	case "ocaml", "ml", "mli":
		language = ocaml.GetLanguage()
	case "toml":
		language = toml.GetLanguage()
	case "cue":
		language = cue.GetLanguage()
	case "pascal", "pas", "dpr", "pp", "delphi", "freepascal":
		language = pascal.GetLanguage()
	case "vbnet", "vb", "visualbasic", "vb.net":
		language = vbnet.GetLanguage()
	default:
		return nil, nil, false
	}

	parser := sitter.NewParser()
	parser.SetLanguage(language)

	return parser, language, true
}

// Chunk splits a file into semantic chunks based on AST structure.
func (c *Chunker) Chunk(file *types.SourceFile) ([]*types.Chunk, error) {
	// Handle languages with embedded JavaScript
	if IsEmbeddedJSLanguage(file.Language) {
		return c.chunkEmbeddedLanguage(file)
	}

	parser, _, ok := c.getParser(file.Language)
	if !ok {
		// Fall back to simple chunking for unsupported languages
		return nil, fmt.Errorf("language %s not supported by TreeSitter", file.Language)
	}
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, file.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	content := string(file.Content)
	lines := strings.Split(content, "\n")

	var chunks []*types.Chunk
	maxChars := c.config.MaxChunkSize * CharsPerToken

	// Walk the AST and extract meaningful nodes
	c.walkNode(root, file, lines, maxChars, &chunks, "")

	// If no chunks were created, create one for the whole file
	if len(chunks) == 0 && len(content) > 0 {
		// Truncate if necessary to avoid embedding failures
		chunkContent := content
		name := ""
		if len(chunkContent) > maxChars {
			chunkContent = chunkContent[:maxChars]
			name = "(truncated)"
		}
		hash := sha256.Sum256([]byte(chunkContent))
		chunks = append(chunks, &types.Chunk{
			ID:        fmt.Sprintf("%s:1:%s", file.Path, hex.EncodeToString(hash[:8])),
			FilePath:  file.Path,
			Language:  file.Language,
			Content:   chunkContent,
			ChunkType: types.ChunkTypeFile,
			Name:      name,
			StartLine: 1,
			EndLine:   len(lines),
			Hash:      hex.EncodeToString(hash[:]),
		})
	}

	return chunks, nil
}

// walkNode recursively walks the AST to extract chunks.
func (c *Chunker) walkNode(node *sitter.Node, file *types.SourceFile, lines []string, maxChars int, chunks *[]*types.Chunk, parentName string) {
	nodeType := node.Type()
	content := string(file.Content)

	// Determine if this node should be a chunk
	chunkType, name := c.classifyNode(nodeType, node, content, file.Language)

	if chunkType != "" {
		startLine := int(node.StartPoint().Row) + 1
		endLine := int(node.EndPoint().Row) + 1

		// Extract content for this node
		startByte := node.StartByte()
		endByte := node.EndByte()
		nodeContent := content[startByte:endByte]

		// Check if chunk is too large
		if len(nodeContent) > maxChars {
			// For very large functions/classes, we still create a chunk
			// but also recurse into children
			c.createChunk(file, nodeContent, chunkType, name, parentName, startLine, endLine, chunks)

			// Also walk children for nested definitions
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				c.walkNode(child, file, lines, maxChars, chunks, name)
			}
		} else {
			c.createChunk(file, nodeContent, chunkType, name, parentName, startLine, endLine, chunks)
		}
		return
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		c.walkNode(child, file, lines, maxChars, chunks, parentName)
	}
}

// classifyNode determines the chunk type and name for a node.
func (c *Chunker) classifyNode(nodeType string, node *sitter.Node, content string, lang string) (types.ChunkType, string) {
	switch lang {
	case "go":
		return c.classifyGoNode(nodeType, node, content)
	case "python":
		return c.classifyPythonNode(nodeType, node, content)
	case "javascript", "jsx", "typescript", "tsx":
		return c.classifyJSNode(nodeType, node, content)
	case "rust":
		return c.classifyRustNode(nodeType, node, content)
	case "java":
		return c.classifyJavaNode(nodeType, node, content)
	case "c", "cpp", "h", "hpp":
		return c.classifyCNode(nodeType, node, content)
	case "ruby", "rb":
		return c.classifyRubyNode(nodeType, node, content)
	case "php":
		return c.classifyPHPNode(nodeType, node, content)
	case "csharp", "cs":
		return c.classifyCSharpNode(nodeType, node, content)
	case "kotlin", "kt", "kts":
		return c.classifyKotlinNode(nodeType, node, content)
	case "swift":
		return c.classifySwiftNode(nodeType, node, content)
	case "scala", "sc":
		return c.classifyScalaNode(nodeType, node, content)
	case "lua":
		return c.classifyLuaNode(nodeType, node, content)
	case "sql":
		return c.classifySQLNode(nodeType, node, content)
	case "proto", "protobuf":
		return c.classifyProtobufNode(nodeType, node, content)
	case "markdown", "md":
		return c.classifyMarkdownNode(nodeType, node, content)
	case "bash", "sh", "shell":
		return c.classifyBashNode(nodeType, node, content)
	case "css":
		return c.classifyCSSNode(nodeType, node, content)
	case "dockerfile":
		return c.classifyDockerfileNode(nodeType, node, content)
	case "yaml", "yml":
		return c.classifyYAMLNode(nodeType, node, content)
	case "hcl", "tf", "terraform":
		return c.classifyHCLNode(nodeType, node, content)
	case "elixir", "ex", "exs":
		return c.classifyElixirNode(nodeType, node, content)
	case "elm":
		return c.classifyElmNode(nodeType, node, content)
	case "groovy", "gradle":
		return c.classifyGroovyNode(nodeType, node, content)
	case "ocaml", "ml", "mli":
		return c.classifyOCamlNode(nodeType, node, content)
	case "toml":
		return c.classifyTOMLNode(nodeType, node, content)
	case "cue":
		return c.classifyCueNode(nodeType, node, content)
	case "pascal", "pas", "dpr", "pp", "delphi", "freepascal":
		return c.classifyPascalNode(nodeType, node, content)
	case "vbnet", "vb", "visualbasic", "vb.net":
		return c.classifyVBNetNode(nodeType, node, content)
	}
	return "", ""
}

func (c *Chunker) classifyGoNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "method_declaration":
		name := c.findChildByType(node, "field_identifier", content)
		return types.ChunkTypeMethod, name
	case "type_declaration":
		spec := c.findChildNodeByType(node, "type_spec")
		if spec != nil {
			name := c.findChildByType(spec, "type_identifier", content)
			return types.ChunkTypeClass, name
		}
	}
	return "", ""
}

func (c *Chunker) classifyPythonNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "class_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyJSNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_declaration", "function":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "class_declaration", "class":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "method_definition":
		name := c.findChildByType(node, "property_identifier", content)
		return types.ChunkTypeMethod, name
	case "arrow_function":
		// Try to find the variable name if this is assigned
		parent := node.Parent()
		if parent != nil && parent.Type() == "variable_declarator" {
			name := c.findChildByType(parent, "identifier", content)
			return types.ChunkTypeFunction, name
		}
	}
	return "", ""
}

func (c *Chunker) classifyRustNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_item":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "impl_item":
		// Get the type being implemented
		typeNode := c.findChildNodeByType(node, "type_identifier")
		if typeNode != nil {
			name := content[typeNode.StartByte():typeNode.EndByte()]
			return types.ChunkTypeClass, "impl " + name
		}
	case "struct_item":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "enum_item":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "trait_item":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyJavaNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "method_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeMethod, name
	case "class_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "interface_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyCNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_definition":
		declarator := c.findChildNodeByType(node, "function_declarator")
		if declarator != nil {
			name := c.findChildByType(declarator, "identifier", content)
			return types.ChunkTypeFunction, name
		}
	case "struct_specifier":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "class_specifier":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyRubyNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "method":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "singleton_method":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeMethod, name
	case "class":
		name := c.findChildByType(node, "constant", content)
		return types.ChunkTypeClass, name
	case "module":
		name := c.findChildByType(node, "constant", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyPHPNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "name", content)
		return types.ChunkTypeFunction, name
	case "method_declaration":
		name := c.findChildByType(node, "name", content)
		return types.ChunkTypeMethod, name
	case "class_declaration":
		name := c.findChildByType(node, "name", content)
		return types.ChunkTypeClass, name
	case "interface_declaration":
		name := c.findChildByType(node, "name", content)
		return types.ChunkTypeClass, name
	case "trait_declaration":
		name := c.findChildByType(node, "name", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyCSharpNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "method_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeMethod, name
	case "class_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "interface_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "struct_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "enum_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "record_declaration":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyKotlinNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "simple_identifier", content)
		return types.ChunkTypeFunction, name
	case "class_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "object_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "interface_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifySwiftNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "simple_identifier", content)
		return types.ChunkTypeFunction, name
	case "class_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "struct_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "enum_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "protocol_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, name
	case "extension_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return types.ChunkTypeClass, "extension " + name
	}
	return "", ""
}

func (c *Chunker) classifyScalaNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "class_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "object_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "trait_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyLuaNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_declaration":
		// Named function: function name() end
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "function_definition":
		// Anonymous function assigned to variable
		parent := node.Parent()
		if parent != nil && parent.Type() == "assignment_statement" {
			varList := c.findChildNodeByType(parent, "variable_list")
			if varList != nil {
				name := c.findChildByType(varList, "identifier", content)
				return types.ChunkTypeFunction, name
			}
		}
	case "local_function":
		// Local function: local function name() end
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	}
	return "", ""
}

func (c *Chunker) classifySQLNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "create_function_statement":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "create_procedure_statement":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "create_table_statement":
		name := c.findChildByType(node, "identifier", content)
		if name == "" {
			name = c.findChildByType(node, "object_reference", content)
		}
		return types.ChunkTypeClass, name
	case "create_view_statement":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "create_trigger_statement":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "create_index_statement":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeBlock, name
	}
	return "", ""
}

func (c *Chunker) classifyProtobufNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "message":
		name := c.findChildByType(node, "message_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return types.ChunkTypeClass, name
	case "enum":
		name := c.findChildByType(node, "enum_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return types.ChunkTypeClass, name
	case "service":
		name := c.findChildByType(node, "service_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return types.ChunkTypeClass, name
	case "rpc":
		name := c.findChildByType(node, "rpc_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return types.ChunkTypeMethod, name
	}
	return "", ""
}

func (c *Chunker) classifyMarkdownNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "atx_heading", "setext_heading":
		// Extract heading text
		startByte := node.StartByte()
		endByte := node.EndByte()
		if endByte > startByte {
			text := content[startByte:endByte]
			// Clean up heading markers
			text = strings.TrimLeft(text, "# ")
			text = strings.TrimRight(text, "\n\r")
			if len(text) > 50 {
				text = text[:50] + "..."
			}
			return types.ChunkTypeBlock, text
		}
	case "fenced_code_block", "indented_code_block":
		// Code blocks as separate chunks
		infoStr := c.findChildByType(node, "info_string", content)
		name := "code"
		if infoStr != "" {
			name = "code:" + infoStr
		}
		return types.ChunkTypeBlock, name
	}
	return "", ""
}

func (c *Chunker) classifyBashNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "word", content)
		return types.ChunkTypeFunction, name
	case "if_statement", "for_statement", "while_statement", "case_statement":
		return types.ChunkTypeBlock, nodeType
	}
	return "", ""
}

func (c *Chunker) classifyCSSNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "rule_set":
		// CSS rule: selector { ... }
		selector := c.findChildByType(node, "selectors", content)
		if selector == "" {
			selector = c.findChildByType(node, "selector_list", content)
		}
		if len(selector) > 50 {
			selector = selector[:50] + "..."
		}
		return types.ChunkTypeBlock, selector
	case "media_statement":
		return types.ChunkTypeBlock, "@media"
	case "keyframes_statement":
		name := c.findChildByType(node, "keyframes_name", content)
		return types.ChunkTypeBlock, "@keyframes " + name
	case "import_statement":
		return types.ChunkTypeBlock, "@import"
	}
	return "", ""
}

func (c *Chunker) classifyDockerfileNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "from_instruction":
		image := c.findChildByType(node, "image_spec", content)
		return types.ChunkTypeBlock, "FROM " + image
	case "run_instruction":
		return types.ChunkTypeBlock, "RUN"
	case "copy_instruction":
		return types.ChunkTypeBlock, "COPY"
	case "add_instruction":
		return types.ChunkTypeBlock, "ADD"
	case "cmd_instruction":
		return types.ChunkTypeBlock, "CMD"
	case "entrypoint_instruction":
		return types.ChunkTypeBlock, "ENTRYPOINT"
	case "env_instruction":
		return types.ChunkTypeBlock, "ENV"
	case "expose_instruction":
		return types.ChunkTypeBlock, "EXPOSE"
	case "volume_instruction":
		return types.ChunkTypeBlock, "VOLUME"
	case "workdir_instruction":
		return types.ChunkTypeBlock, "WORKDIR"
	case "arg_instruction":
		return types.ChunkTypeBlock, "ARG"
	case "label_instruction":
		return types.ChunkTypeBlock, "LABEL"
	}
	return "", ""
}

func (c *Chunker) classifyYAMLNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "block_mapping_pair":
		// Top-level key-value pair
		key := c.findChildByType(node, "flow_node", content)
		if key == "" {
			// Try to get key from the first child
			if node.ChildCount() > 0 {
				firstChild := node.Child(0)
				key = content[firstChild.StartByte():firstChild.EndByte()]
			}
		}
		if len(key) > 50 {
			key = key[:50] + "..."
		}
		return types.ChunkTypeBlock, key
	case "block_sequence":
		return types.ChunkTypeBlock, "sequence"
	}
	return "", ""
}

func (c *Chunker) classifyHCLNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "block":
		// HCL block: resource "type" "name" { ... }
		// Get block type (resource, variable, output, etc.)
		blockType := ""
		blockName := ""
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			childType := child.Type()
			if childType == "identifier" {
				if blockType == "" {
					blockType = content[child.StartByte():child.EndByte()]
				}
			} else if childType == "string_lit" {
				text := content[child.StartByte():child.EndByte()]
				text = strings.Trim(text, "\"")
				if blockName == "" {
					blockName = text
				} else {
					blockName = blockName + "." + text
				}
			}
		}
		name := blockType
		if blockName != "" {
			name = blockType + " " + blockName
		}
		return types.ChunkTypeBlock, name
	case "attribute":
		// Variable assignment
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeBlock, name
	}
	return "", ""
}

func (c *Chunker) classifyElixirNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "call":
		// Check if it's a def/defp/defmodule
		target := c.findChildByType(node, "identifier", content)
		switch target {
		case "def", "defp":
			// Function definition
			args := c.findChildNodeByType(node, "arguments")
			if args != nil {
				name := c.findChildByType(args, "identifier", content)
				return types.ChunkTypeFunction, name
			}
		case "defmodule":
			args := c.findChildNodeByType(node, "arguments")
			if args != nil {
				alias := c.findChildNodeByType(args, "alias")
				if alias != nil {
					name := content[alias.StartByte():alias.EndByte()]
					return types.ChunkTypeClass, name
				}
			}
		}
	}
	return "", ""
}

func (c *Chunker) classifyElmNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "value_declaration":
		name := c.findChildByType(node, "lower_case_identifier", content)
		return types.ChunkTypeFunction, name
	case "type_declaration":
		name := c.findChildByType(node, "upper_case_identifier", content)
		return types.ChunkTypeClass, name
	case "type_alias_declaration":
		name := c.findChildByType(node, "upper_case_identifier", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyGroovyNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "method_declaration", "function_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "class_definition":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "closure":
		return types.ChunkTypeFunction, "closure"
	}
	return "", ""
}

func (c *Chunker) classifyOCamlNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "let_binding", "value_definition":
		name := c.findChildByType(node, "value_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return types.ChunkTypeFunction, name
	case "type_definition":
		name := c.findChildByType(node, "type_constructor", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return types.ChunkTypeClass, name
	case "module_definition":
		name := c.findChildByType(node, "module_name", content)
		return types.ChunkTypeClass, name
	}
	return "", ""
}

func (c *Chunker) classifyTOMLNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "table":
		// [section.name]
		key := c.findChildByType(node, "dotted_key", content)
		if key == "" {
			key = c.findChildByType(node, "bare_key", content)
		}
		return types.ChunkTypeBlock, key
	case "table_array_element":
		// [[array.section]]
		key := c.findChildByType(node, "dotted_key", content)
		if key == "" {
			key = c.findChildByType(node, "bare_key", content)
		}
		return types.ChunkTypeBlock, key
	case "pair":
		key := c.findChildByType(node, "bare_key", content)
		if key == "" {
			key = c.findChildByType(node, "quoted_key", content)
		}
		return types.ChunkTypeBlock, key
	}
	return "", ""
}

func (c *Chunker) classifyCueNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "field":
		// Field definition
		label := c.findChildByType(node, "identifier", content)
		if label == "" {
			label = c.findChildByType(node, "string", content)
		}
		return types.ChunkTypeBlock, label
	case "package_clause":
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeBlock, "package " + name
	}
	return "", ""
}

func (c *Chunker) classifyPascalNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "declProc":
		// Function or procedure declaration
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeFunction, name
	case "declClass":
		// Class declaration
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "declIntf":
		// Interface declaration
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeClass, name
	case "declType":
		// Type declaration (record, enum, etc.)
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeBlock, name
	case "declVar":
		// Variable declaration block
		return types.ChunkTypeBlock, ""
	case "declConst":
		// Constant declaration block
		return types.ChunkTypeBlock, ""
	case "unit", "program", "library":
		// Top-level unit/program/library declaration
		name := c.findChildByType(node, "identifier", content)
		return types.ChunkTypeBlock, name
	}
	return "", ""
}

// findChildByType finds a child node of the given type and returns its content.
func (c *Chunker) findChildByType(node *sitter.Node, childType string, content string) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == childType {
			return content[child.StartByte():child.EndByte()]
		}
	}
	return ""
}

// findChildNodeByType finds a child node of the given type.
func (c *Chunker) findChildNodeByType(node *sitter.Node, childType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == childType {
			return child
		}
	}
	return nil
}

// createChunk creates a chunk and appends it to the list.
// If content exceeds maxChunkSize, it will be truncated to avoid embedding failures.
func (c *Chunker) createChunk(file *types.SourceFile, content string, chunkType types.ChunkType, name, parentName string, startLine, endLine int, chunks *[]*types.Chunk) {
	maxChars := c.config.MaxChunkSize * CharsPerToken

	// Truncate oversized chunks to prevent embedding failures
	truncated := false
	if len(content) > maxChars {
		content = content[:maxChars]
		truncated = true
	}

	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:8])

	chunkName := name
	if truncated && name != "" {
		chunkName = name + " (truncated)"
	}

	chunk := &types.Chunk{
		ID:         fmt.Sprintf("%s:%d:%s", file.Path, startLine, hashStr),
		FilePath:   file.Path,
		Language:   file.Language,
		Content:    content,
		ChunkType:  chunkType,
		Name:       chunkName,
		ParentName: parentName,
		StartLine:  startLine,
		EndLine:    endLine,
		Hash:       hex.EncodeToString(hash[:]),
	}

	*chunks = append(*chunks, chunk)
}

// SupportedLanguages returns languages supported by this chunker.
func (c *Chunker) SupportedLanguages() []string {
	return []string{
		"go", "python", "javascript", "typescript", "jsx", "tsx",
		"rust", "java", "c", "cpp", "h",
		"ruby", "php", "csharp", "kotlin", "swift", "scala",
		"html", "htm", "xhtml", "svelte",
		"lua", "sql", "proto", "protobuf", "markdown", "md",
		"bash", "sh", "shell", "css", "dockerfile", "yaml", "yml",
		"hcl", "tf", "terraform",
		"elixir", "ex", "exs", "elm", "groovy", "gradle",
		"ocaml", "ml", "mli", "toml", "cue",
		"pascal", "pas", "dpr", "pp", "delphi", "freepascal",
		"vbnet", "vb", "visualbasic", "vb.net",
	}
}

// SupportsLanguage checks if a language is supported.
func (c *Chunker) SupportsLanguage(lang string) bool {
	_, _, ok := c.getParser(lang)
	return ok
}

// ExtractSymbols extracts symbols from a file using Tree-sitter.
func (c *Chunker) ExtractSymbols(file *types.SourceFile) ([]*types.Symbol, error) {
	// Handle languages with embedded JavaScript
	if IsEmbeddedJSLanguage(file.Language) {
		return c.extractSymbolsFromEmbeddedLanguage(file)
	}

	parser, _, ok := c.getParser(file.Language)
	if !ok {
		return nil, nil
	}
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, file.Content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	content := string(file.Content)
	var symbols []*types.Symbol

	c.extractSymbolsFromNode(tree.RootNode(), file, content, &symbols)

	return symbols, nil
}

// extractSymbolsFromNode recursively extracts symbols from AST nodes.
func (c *Chunker) extractSymbolsFromNode(node *sitter.Node, file *types.SourceFile, content string, symbols *[]*types.Symbol) {
	nodeType := node.Type()

	// Check if this node is a symbol definition
	if sym := c.nodeToSymbol(nodeType, node, file, content); sym != nil {
		*symbols = append(*symbols, sym)
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		c.extractSymbolsFromNode(node.Child(i), file, content, symbols)
	}
}

// nodeToSymbol converts a node to a Symbol if applicable.
func (c *Chunker) nodeToSymbol(nodeType string, node *sitter.Node, file *types.SourceFile, content string) *types.Symbol {
	startLine := int(node.StartPoint().Row) + 1
	endLine := int(node.EndPoint().Row) + 1

	var sym *types.Symbol

	switch file.Language {
	case "go":
		sym = c.goNodeToSymbol(nodeType, node, content)
	case "python":
		sym = c.pythonNodeToSymbol(nodeType, node, content)
	case "javascript", "jsx", "typescript", "tsx":
		sym = c.jsNodeToSymbol(nodeType, node, content)
	case "rust":
		sym = c.rustNodeToSymbol(nodeType, node, content)
	case "java":
		sym = c.javaNodeToSymbol(nodeType, node, content)
	case "c", "cpp", "h", "hpp":
		sym = c.cNodeToSymbol(nodeType, node, content)
	case "ruby", "rb":
		sym = c.rubyNodeToSymbol(nodeType, node, content)
	case "php":
		sym = c.phpNodeToSymbol(nodeType, node, content)
	case "csharp", "cs":
		sym = c.csharpNodeToSymbol(nodeType, node, content)
	case "kotlin", "kt", "kts":
		sym = c.kotlinNodeToSymbol(nodeType, node, content)
	case "swift":
		sym = c.swiftNodeToSymbol(nodeType, node, content)
	case "scala", "sc":
		sym = c.scalaNodeToSymbol(nodeType, node, content)
	case "lua":
		sym = c.luaNodeToSymbol(nodeType, node, content)
	case "sql":
		sym = c.sqlNodeToSymbol(nodeType, node, content)
	case "proto", "protobuf":
		sym = c.protobufNodeToSymbol(nodeType, node, content)
	case "markdown", "md":
		sym = c.markdownNodeToSymbol(nodeType, node, content)
	case "bash", "sh", "shell":
		sym = c.bashNodeToSymbol(nodeType, node, content)
	case "css":
		sym = c.cssNodeToSymbol(nodeType, node, content)
	case "dockerfile":
		sym = c.dockerfileNodeToSymbol(nodeType, node, content)
	case "yaml", "yml":
		sym = c.yamlNodeToSymbol(nodeType, node, content)
	case "hcl", "tf", "terraform":
		sym = c.hclNodeToSymbol(nodeType, node, content)
	case "elixir", "ex", "exs":
		sym = c.elixirNodeToSymbol(nodeType, node, content)
	case "elm":
		sym = c.elmNodeToSymbol(nodeType, node, content)
	case "groovy", "gradle":
		sym = c.groovyNodeToSymbol(nodeType, node, content)
	case "ocaml", "ml", "mli":
		sym = c.ocamlNodeToSymbol(nodeType, node, content)
	case "toml":
		sym = c.tomlNodeToSymbol(nodeType, node, content)
	case "cue":
		sym = c.cueNodeToSymbol(nodeType, node, content)
	case "pascal", "pas", "dpr", "pp", "delphi", "freepascal":
		sym = c.pascalNodeToSymbol(nodeType, node, content)
	case "vbnet", "vb", "visualbasic", "vb.net":
		sym = c.vbnetNodeToSymbol(nodeType, node, content)
	}

	if sym != nil {
		sym.FilePath = file.Path
		sym.StartLine = startLine
		sym.EndLine = endLine
		sym.LineCount = endLine - startLine + 1
		sym.ID = fmt.Sprintf("%s:%s:%d", file.Path, sym.Name, startLine)

		// Extract doc comment if present
		sym.DocComment = c.extractDocComment(node, content)

		// Determine visibility
		sym.Visibility = c.determineVisibility(sym.Name, file.Language)
	}

	return sym
}

func (c *Chunker) goNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "method_declaration":
		name := c.findChildByType(node, "field_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindMethod,
		}
	case "type_declaration":
		spec := c.findChildNodeByType(node, "type_spec")
		if spec != nil {
			name := c.findChildByType(spec, "type_identifier", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindType,
			}
		}
	case "const_spec":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindConstant,
		}
	case "var_spec":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) pythonNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "class_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	}
	return nil
}

func (c *Chunker) jsNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "class_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "method_definition":
		name := c.findChildByType(node, "property_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindMethod,
		}
	}
	return nil
}

func (c *Chunker) rustNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_item":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "struct_item":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "trait_item":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "enum_item":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "const_item":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindConstant,
		}
	case "static_item":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) javaNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "method_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindMethod,
		}
	case "class_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "interface_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "enum_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "field_declaration":
		declarator := c.findChildNodeByType(node, "variable_declarator")
		if declarator != nil {
			name := c.findChildByType(declarator, "identifier", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindVariable,
			}
		}
	case "constant_declaration":
		declarator := c.findChildNodeByType(node, "variable_declarator")
		if declarator != nil {
			name := c.findChildByType(declarator, "identifier", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindConstant,
			}
		}
	}
	return nil
}

func (c *Chunker) cNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_definition":
		declarator := c.findChildNodeByType(node, "function_declarator")
		if declarator != nil {
			name := c.findChildByType(declarator, "identifier", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindFunction,
			}
		}
	case "declaration":
		// Check if it's a function declaration
		declarator := c.findChildNodeByType(node, "function_declarator")
		if declarator != nil {
			name := c.findChildByType(declarator, "identifier", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindFunction,
			}
		}
		// Variable declaration
		initDeclarator := c.findChildNodeByType(node, "init_declarator")
		if initDeclarator != nil {
			name := c.findChildByType(initDeclarator, "identifier", content)
			if name != "" {
				return &types.Symbol{
					Name: name,
					Kind: types.SymbolKindVariable,
				}
			}
		}
	case "struct_specifier":
		name := c.findChildByType(node, "type_identifier", content)
		if name != "" {
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindType,
			}
		}
	case "class_specifier":
		name := c.findChildByType(node, "type_identifier", content)
		if name != "" {
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindType,
			}
		}
	case "enum_specifier":
		name := c.findChildByType(node, "type_identifier", content)
		if name != "" {
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindType,
			}
		}
	case "type_definition":
		declarator := c.findChildNodeByType(node, "type_declarator")
		if declarator != nil {
			name := c.findChildByType(declarator, "type_identifier", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindType,
			}
		}
	}
	return nil
}

func (c *Chunker) rubyNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "method":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "singleton_method":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindMethod,
		}
	case "class":
		name := c.findChildByType(node, "constant", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "module":
		name := c.findChildByType(node, "constant", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "constant":
		// Top-level constant assignment
		parent := node.Parent()
		if parent != nil && parent.Type() == "assignment" {
			name := content[node.StartByte():node.EndByte()]
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindConstant,
			}
		}
	}
	return nil
}

func (c *Chunker) phpNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "name", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "method_declaration":
		name := c.findChildByType(node, "name", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindMethod,
		}
	case "class_declaration":
		name := c.findChildByType(node, "name", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "interface_declaration":
		name := c.findChildByType(node, "name", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "trait_declaration":
		name := c.findChildByType(node, "name", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "const_declaration":
		elem := c.findChildNodeByType(node, "const_element")
		if elem != nil {
			name := c.findChildByType(elem, "name", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindConstant,
			}
		}
	}
	return nil
}

func (c *Chunker) csharpNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "method_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindMethod,
		}
	case "class_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "interface_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "struct_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "enum_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "record_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "field_declaration":
		declarator := c.findChildNodeByType(node, "variable_declarator")
		if declarator != nil {
			name := c.findChildByType(declarator, "identifier", content)
			return &types.Symbol{
				Name: name,
				Kind: types.SymbolKindVariable,
			}
		}
	case "property_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) kotlinNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "simple_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "class_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "object_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "interface_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "property_declaration":
		name := c.findChildByType(node, "simple_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) swiftNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "simple_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "class_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "struct_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "enum_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "protocol_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "extension_declaration":
		name := c.findChildByType(node, "type_identifier", content)
		return &types.Symbol{
			Name: "extension " + name,
			Kind: types.SymbolKindType,
		}
	case "property_declaration":
		name := c.findChildByType(node, "simple_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) scalaNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "class_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "object_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "trait_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "val_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindConstant,
		}
	case "var_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) luaNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_declaration":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "local_function":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "function_definition":
		// Anonymous function assigned to variable
		parent := node.Parent()
		if parent != nil && parent.Type() == "assignment_statement" {
			varList := c.findChildNodeByType(parent, "variable_list")
			if varList != nil {
				name := c.findChildByType(varList, "identifier", content)
				if name != "" {
					return &types.Symbol{
						Name: name,
						Kind: types.SymbolKindFunction,
					}
				}
			}
		}
	case "variable_declaration":
		// Local variables
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) sqlNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "create_function_statement":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "create_procedure_statement":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "create_table_statement":
		name := c.findChildByType(node, "identifier", content)
		if name == "" {
			name = c.findChildByType(node, "object_reference", content)
		}
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "create_view_statement":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "create_trigger_statement":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "create_index_statement":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) protobufNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "message":
		name := c.findChildByType(node, "message_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "enum":
		name := c.findChildByType(node, "enum_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "service":
		name := c.findChildByType(node, "service_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindInterface,
		}
	case "rpc":
		name := c.findChildByType(node, "rpc_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindMethod,
		}
	}
	return nil
}

func (c *Chunker) markdownNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	// Markdown doesn't have traditional symbols, but we can track headings
	switch nodeType {
	case "atx_heading", "setext_heading":
		startByte := node.StartByte()
		endByte := node.EndByte()
		if endByte > startByte {
			text := content[startByte:endByte]
			text = strings.TrimLeft(text, "# ")
			text = strings.TrimRight(text, "\n\r")
			if len(text) > 50 {
				text = text[:50] + "..."
			}
			return &types.Symbol{
				Name: text,
				Kind: types.SymbolKindVariable, // Using Variable for headings
			}
		}
	}
	return nil
}

func (c *Chunker) bashNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "function_definition":
		name := c.findChildByType(node, "word", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	}
	return nil
}

func (c *Chunker) cssNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "rule_set":
		selector := c.findChildByType(node, "selectors", content)
		if selector == "" {
			selector = c.findChildByType(node, "selector_list", content)
		}
		if len(selector) > 50 {
			selector = selector[:50] + "..."
		}
		return &types.Symbol{
			Name: selector,
			Kind: types.SymbolKindVariable, // Using Variable for CSS rules
		}
	case "keyframes_statement":
		name := c.findChildByType(node, "keyframes_name", content)
		return &types.Symbol{
			Name: "@keyframes " + name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) dockerfileNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "from_instruction":
		image := c.findChildByType(node, "image_spec", content)
		return &types.Symbol{
			Name: "FROM " + image,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) yamlNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "block_mapping_pair":
		key := c.findChildByType(node, "flow_node", content)
		if key == "" && node.ChildCount() > 0 {
			firstChild := node.Child(0)
			key = content[firstChild.StartByte():firstChild.EndByte()]
		}
		if len(key) > 50 {
			key = key[:50] + "..."
		}
		return &types.Symbol{
			Name: key,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) hclNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "block":
		blockType := ""
		blockName := ""
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			childType := child.Type()
			if childType == "identifier" {
				if blockType == "" {
					blockType = content[child.StartByte():child.EndByte()]
				}
			} else if childType == "string_lit" {
				text := content[child.StartByte():child.EndByte()]
				text = strings.Trim(text, "\"")
				if blockName == "" {
					blockName = text
				} else {
					blockName = blockName + "." + text
				}
			}
		}
		name := blockType
		if blockName != "" {
			name = blockType + " " + blockName
		}
		// Map HCL block types to symbol kinds
		kind := types.SymbolKindVariable
		switch blockType {
		case "resource", "data":
			kind = types.SymbolKindType
		case "variable", "output", "locals":
			kind = types.SymbolKindVariable
		case "module":
			kind = types.SymbolKindType
		case "provider":
			kind = types.SymbolKindInterface
		}
		return &types.Symbol{
			Name: name,
			Kind: kind,
		}
	case "attribute":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) elixirNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "call":
		target := c.findChildByType(node, "identifier", content)
		switch target {
		case "def", "defp":
			args := c.findChildNodeByType(node, "arguments")
			if args != nil {
				name := c.findChildByType(args, "identifier", content)
				kind := types.SymbolKindFunction
				if target == "defp" {
					kind = types.SymbolKindFunction // private
				}
				return &types.Symbol{
					Name: name,
					Kind: kind,
				}
			}
		case "defmodule":
			args := c.findChildNodeByType(node, "arguments")
			if args != nil {
				alias := c.findChildNodeByType(args, "alias")
				if alias != nil {
					name := content[alias.StartByte():alias.EndByte()]
					return &types.Symbol{
						Name: name,
						Kind: types.SymbolKindType,
					}
				}
			}
		}
	}
	return nil
}

func (c *Chunker) elmNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "value_declaration":
		name := c.findChildByType(node, "lower_case_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "type_declaration":
		name := c.findChildByType(node, "upper_case_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "type_alias_declaration":
		name := c.findChildByType(node, "upper_case_identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	}
	return nil
}

func (c *Chunker) groovyNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "method_declaration", "function_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "class_definition":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	}
	return nil
}

func (c *Chunker) ocamlNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "let_binding", "value_definition":
		name := c.findChildByType(node, "value_name", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindFunction,
		}
	case "type_definition":
		name := c.findChildByType(node, "type_constructor", content)
		if name == "" {
			name = c.findChildByType(node, "identifier", content)
		}
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	case "module_definition":
		name := c.findChildByType(node, "module_name", content)
		return &types.Symbol{
			Name: name,
			Kind: types.SymbolKindType,
		}
	}
	return nil
}

func (c *Chunker) tomlNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "table":
		key := c.findChildByType(node, "dotted_key", content)
		if key == "" {
			key = c.findChildByType(node, "bare_key", content)
		}
		return &types.Symbol{
			Name: key,
			Kind: types.SymbolKindVariable,
		}
	case "table_array_element":
		key := c.findChildByType(node, "dotted_key", content)
		if key == "" {
			key = c.findChildByType(node, "bare_key", content)
		}
		return &types.Symbol{
			Name: key,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) cueNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "field":
		label := c.findChildByType(node, "identifier", content)
		if label == "" {
			label = c.findChildByType(node, "string", content)
		}
		return &types.Symbol{
			Name: label,
			Kind: types.SymbolKindVariable,
		}
	}
	return nil
}

func (c *Chunker) pascalNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "declProc":
		// Function or procedure
		name := c.findChildByType(node, "identifier", content)
		// Check if it's a function or procedure by looking for kFunction/kProcedure
		kind := types.SymbolKindFunction
		nodeContent := content[node.StartByte():node.EndByte()]
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(nodeContent)), "procedure") {
			kind = types.SymbolKindFunction
		}
		// Determine visibility - Pascal uses published/public/private/protected sections
		visibility := "public"
		if strings.HasPrefix(name, "_") {
			visibility = "private"
		}
		return &types.Symbol{
			Name:       name,
			Kind:       kind,
			Visibility: visibility,
		}
	case "declClass":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: "public",
		}
	case "declIntf":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindInterface,
			Visibility: "public",
		}
	case "declType":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: "public",
		}
	case "declEnum":
		name := c.findChildByType(node, "identifier", content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: "public",
		}
	case "declConst":
		// Constant declaration
		name := c.findChildByType(node, "identifier", content)
		if name != "" {
			return &types.Symbol{
				Name:       name,
				Kind:       types.SymbolKindConstant,
				Visibility: "public",
			}
		}
	case "declVar":
		// Variable declaration
		name := c.findChildByType(node, "identifier", content)
		if name != "" {
			return &types.Symbol{
				Name:       name,
				Kind:       types.SymbolKindVariable,
				Visibility: "public",
			}
		}
	}
	return nil
}

func (c *Chunker) classifyVBNetNode(nodeType string, node *sitter.Node, content string) (types.ChunkType, string) {
	switch nodeType {
	case "class_block":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeClass, name
	case "module_block":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeClass, name
	case "structure_block":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeClass, name
	case "interface_block":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeClass, name
	case "enum_block":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeBlock, name
	case "method_declaration":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeFunction, name
	case "constructor_declaration":
		return types.ChunkTypeFunction, "New"
	case "property_declaration":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeFunction, name
	case "event_declaration":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeBlock, name
	case "delegate_declaration":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeBlock, name
	case "namespace_block":
		name := c.findVBNetIdentifier(node, content)
		return types.ChunkTypeBlock, name
	}
	return "", ""
}

func (c *Chunker) findVBNetIdentifier(node *sitter.Node, content string) string {
	// Try to find identifier in various child positions
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()
		if childType == "identifier" || childType == "qualified_identifier" {
			return content[child.StartByte():child.EndByte()]
		}
		// Also check in declaration headers
		if strings.Contains(childType, "declaration") || strings.Contains(childType, "header") {
			for j := 0; j < int(child.ChildCount()); j++ {
				grandchild := child.Child(j)
				if grandchild.Type() == "identifier" {
					return content[grandchild.StartByte():grandchild.EndByte()]
				}
			}
		}
	}
	return ""
}

func (c *Chunker) vbnetNodeToSymbol(nodeType string, node *sitter.Node, content string) *types.Symbol {
	switch nodeType {
	case "class_block":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "module_block":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: "public",
		}
	case "structure_block":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "interface_block":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindInterface,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "enum_block":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "method_declaration":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindFunction,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "constructor_declaration":
		return &types.Symbol{
			Name:       "New",
			Kind:       types.SymbolKindFunction,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "property_declaration":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindVariable,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "event_declaration":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindVariable,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "delegate_declaration":
		name := c.findVBNetIdentifier(node, content)
		return &types.Symbol{
			Name:       name,
			Kind:       types.SymbolKindType,
			Visibility: c.getVBNetVisibility(node, content),
		}
	case "field_declaration":
		name := c.findVBNetIdentifier(node, content)
		if name != "" {
			return &types.Symbol{
				Name:       name,
				Kind:       types.SymbolKindVariable,
				Visibility: c.getVBNetVisibility(node, content),
			}
		}
	case "const_declaration":
		name := c.findVBNetIdentifier(node, content)
		if name != "" {
			return &types.Symbol{
				Name:       name,
				Kind:       types.SymbolKindConstant,
				Visibility: c.getVBNetVisibility(node, content),
			}
		}
	}
	return nil
}

func (c *Chunker) getVBNetVisibility(node *sitter.Node, content string) string {
	nodeContent := strings.ToLower(content[node.StartByte():node.EndByte()])
	if strings.Contains(nodeContent, "private ") {
		return "private"
	}
	if strings.Contains(nodeContent, "protected ") {
		return "protected"
	}
	if strings.Contains(nodeContent, "friend ") {
		return "internal"
	}
	return "public"
}

// extractDocComment extracts documentation comment preceding a node.
func (c *Chunker) extractDocComment(node *sitter.Node, content string) string {
	// Look for comment nodes before this node
	prev := node.PrevSibling()
	if prev != nil && strings.Contains(prev.Type(), "comment") {
		return content[prev.StartByte():prev.EndByte()]
	}
	return ""
}

// determineVisibility determines if a symbol is public or private.
func (c *Chunker) determineVisibility(name, lang string) string {
	if name == "" {
		return "private"
	}

	switch lang {
	case "go":
		// Go: uppercase first letter = public
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			return "public"
		}
		return "private"
	case "python":
		// Python: leading underscore = private
		if strings.HasPrefix(name, "_") {
			return "private"
		}
		return "public"
	default:
		return "public"
	}
}

// ExtractReferences extracts references from a file.
func (c *Chunker) ExtractReferences(file *types.SourceFile) ([]*types.Reference, error) {
	// Handle languages with embedded JavaScript
	if IsEmbeddedJSLanguage(file.Language) {
		return c.extractRefsFromEmbeddedLanguage(file)
	}

	parser, _, ok := c.getParser(file.Language)
	if !ok {
		return nil, nil
	}
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, file.Content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	content := string(file.Content)
	var refs []*types.Reference

	// First, build a map of local symbols for determining if refs are external
	localSymbols := make(map[string]bool)
	symbols, _ := c.ExtractSymbols(file)
	for _, sym := range symbols {
		localSymbols[sym.Name] = true
	}

	c.extractRefsFromNode(tree.RootNode(), file, content, localSymbols, &refs, "")

	return refs, nil
}

// extractRefsFromNode recursively extracts references from AST nodes.
func (c *Chunker) extractRefsFromNode(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()

	// Track current function context
	switch file.Language {
	case "go":
		if nodeType == "function_declaration" || nodeType == "method_declaration" {
			name := c.findChildByType(node, "identifier", content)
			if name == "" {
				name = c.findChildByType(node, "field_identifier", content)
			}
			currentFunc = name
		}
	}

	// Extract references based on language
	switch file.Language {
	case "go":
		c.extractGoRefs(node, file, content, localSymbols, refs, currentFunc)
	case "python":
		c.extractPythonRefs(node, file, content, localSymbols, refs, currentFunc)
	case "javascript", "jsx", "typescript", "tsx":
		c.extractJSRefs(node, file, content, localSymbols, refs, currentFunc)
	case "rust":
		c.extractRustRefs(node, file, content, localSymbols, refs, currentFunc)
	case "java":
		c.extractJavaRefs(node, file, content, localSymbols, refs, currentFunc)
	case "c", "cpp", "h", "hpp":
		c.extractCRefs(node, file, content, localSymbols, refs, currentFunc)
	case "ruby", "rb":
		c.extractRubyRefs(node, file, content, localSymbols, refs, currentFunc)
	case "php":
		c.extractPHPRefs(node, file, content, localSymbols, refs, currentFunc)
	case "csharp", "cs":
		c.extractCSharpRefs(node, file, content, localSymbols, refs, currentFunc)
	case "kotlin", "kt", "kts":
		c.extractKotlinRefs(node, file, content, localSymbols, refs, currentFunc)
	case "swift":
		c.extractSwiftRefs(node, file, content, localSymbols, refs, currentFunc)
	case "scala", "sc":
		c.extractScalaRefs(node, file, content, localSymbols, refs, currentFunc)
	case "lua":
		c.extractLuaRefs(node, file, content, localSymbols, refs, currentFunc)
	case "sql":
		c.extractSQLRefs(node, file, content, localSymbols, refs, currentFunc)
	case "proto", "protobuf":
		c.extractProtobufRefs(node, file, content, localSymbols, refs, currentFunc)
	// markdown doesn't have meaningful references
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		c.extractRefsFromNode(node.Child(i), file, content, localSymbols, refs, currentFunc)
	}
}

// extractGoRefs extracts references from Go code.
func (c *Chunker) extractGoRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call_expression":
		// Function/method calls
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode == nil {
			// Try selector_expression for method calls
			selectorNode := c.findChildNodeByType(node, "selector_expression")
			if selectorNode != nil {
				funcNode = c.findChildNodeByType(selectorNode, "field_identifier")
			}
		}
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: currentFunc,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "type_identifier":
		// Type usage (but not in type declarations)
		parent := node.Parent()
		if parent != nil && parent.Type() != "type_spec" {
			typeName := content[node.StartByte():node.EndByte()]
			if typeName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}

	case "import_spec":
		// Import statements
		pathNode := c.findChildNodeByType(node, "interpreted_string_literal")
		if pathNode != nil {
			importPath := content[pathNode.StartByte():pathNode.EndByte()]
			importPath = strings.Trim(importPath, "\"")
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, importPath),
				FromSymbol: file.Path,
				ToSymbol:   importPath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}
	}
}

// extractPythonRefs extracts references from Python code.
func (c *Chunker) extractPythonRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call":
		// Function calls
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode == nil {
			// Try attribute for method calls
			attrNode := c.findChildNodeByType(node, "attribute")
			if attrNode != nil {
				funcNode = c.findChildNodeByType(attrNode, "identifier")
			}
		}
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "import_statement", "import_from_statement":
		// Import statements
		modNode := c.findChildNodeByType(node, "dotted_name")
		if modNode != nil {
			modName := content[modNode.StartByte():modNode.EndByte()]
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, modName),
				FromSymbol: file.Path,
				ToSymbol:   modName,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}
	}
}

// extractJSRefs extracts references from JavaScript/TypeScript code.
func (c *Chunker) extractJSRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call_expression":
		// Function calls
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode == nil {
			// Try member_expression for method calls
			memberNode := c.findChildNodeByType(node, "member_expression")
			if memberNode != nil {
				funcNode = c.findChildNodeByType(memberNode, "property_identifier")
			}
		}
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "import_statement":
		// Import statements
		sourceNode := c.findChildNodeByType(node, "string")
		if sourceNode != nil {
			source := content[sourceNode.StartByte():sourceNode.EndByte()]
			source = strings.Trim(source, "\"'")
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, source),
				FromSymbol: file.Path,
				ToSymbol:   source,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}
	}
}

// extractRustRefs extracts references from Rust code.
func (c *Chunker) extractRustRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call_expression":
		// Function calls
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode == nil {
			// Try field_expression for method calls
			fieldNode := c.findChildNodeByType(node, "field_expression")
			if fieldNode != nil {
				funcNode = c.findChildNodeByType(fieldNode, "field_identifier")
			}
		}
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "use_declaration":
		// Use statements
		pathNode := c.findChildNodeByType(node, "scoped_identifier")
		if pathNode == nil {
			pathNode = c.findChildNodeByType(node, "identifier")
		}
		if pathNode != nil {
			usePath := content[pathNode.StartByte():pathNode.EndByte()]
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, usePath),
				FromSymbol: file.Path,
				ToSymbol:   usePath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "type_identifier":
		// Type usage
		parent := node.Parent()
		if parent != nil && parent.Type() != "struct_item" && parent.Type() != "enum_item" {
			typeName := content[node.StartByte():node.EndByte()]
			if typeName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractJavaRefs extracts references from Java code.
func (c *Chunker) extractJavaRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "method_invocation":
		// Method calls
		nameNode := c.findChildNodeByType(node, "identifier")
		if nameNode != nil {
			calledFunc := content[nameNode.StartByte():nameNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "object_creation_expression":
		// Constructor calls (new ClassName())
		typeNode := c.findChildNodeByType(node, "type_identifier")
		if typeNode != nil {
			typeName := content[typeNode.StartByte():typeNode.EndByte()]
			if typeName != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, typeName),
					FromSymbol: fromSym,
					ToSymbol:   typeName,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}

	case "import_declaration":
		// Import statements
		pathNode := c.findChildNodeByType(node, "scoped_identifier")
		if pathNode != nil {
			importPath := content[pathNode.StartByte():pathNode.EndByte()]
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, importPath),
				FromSymbol: file.Path,
				ToSymbol:   importPath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "type_identifier":
		// Type usage
		parent := node.Parent()
		if parent != nil && parent.Type() != "class_declaration" && parent.Type() != "interface_declaration" {
			typeName := content[node.StartByte():node.EndByte()]
			if typeName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractCRefs extracts references from C/C++ code.
func (c *Chunker) extractCRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call_expression":
		// Function calls
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode == nil {
			// Try field_expression for method calls
			fieldNode := c.findChildNodeByType(node, "field_expression")
			if fieldNode != nil {
				funcNode = c.findChildNodeByType(fieldNode, "field_identifier")
			}
		}
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "preproc_include":
		// #include statements
		pathNode := c.findChildNodeByType(node, "string_literal")
		if pathNode == nil {
			pathNode = c.findChildNodeByType(node, "system_lib_string")
		}
		if pathNode != nil {
			includePath := content[pathNode.StartByte():pathNode.EndByte()]
			includePath = strings.Trim(includePath, "\"<>")
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, includePath),
				FromSymbol: file.Path,
				ToSymbol:   includePath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "type_identifier":
		// Type usage
		parent := node.Parent()
		if parent != nil && parent.Type() != "struct_specifier" && parent.Type() != "class_specifier" {
			typeName := content[node.StartByte():node.EndByte()]
			if typeName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractRubyRefs extracts references from Ruby code.
func (c *Chunker) extractRubyRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call":
		// Method calls
		methodNode := c.findChildNodeByType(node, "identifier")
		if methodNode != nil {
			calledFunc := content[methodNode.StartByte():methodNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "require", "require_relative":
		// Require statements
		argNode := c.findChildNodeByType(node, "argument_list")
		if argNode != nil {
			strNode := c.findChildNodeByType(argNode, "string")
			if strNode != nil {
				requirePath := content[strNode.StartByte():strNode.EndByte()]
				requirePath = strings.Trim(requirePath, "\"'")
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, requirePath),
					FromSymbol: file.Path,
					ToSymbol:   requirePath,
					Kind:       types.RefKindImport,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: true,
				}
				*refs = append(*refs, ref)
			}
		}

	case "constant":
		// Class/module references
		parent := node.Parent()
		if parent != nil && parent.Type() != "class" && parent.Type() != "module" {
			constName := content[node.StartByte():node.EndByte()]
			if constName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, constName),
					FromSymbol: currentFunc,
					ToSymbol:   constName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[constName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractPHPRefs extracts references from PHP code.
func (c *Chunker) extractPHPRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "function_call_expression":
		// Function calls
		funcNode := c.findChildNodeByType(node, "name")
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "member_call_expression", "scoped_call_expression":
		// Method calls
		nameNode := c.findChildNodeByType(node, "name")
		if nameNode != nil {
			calledFunc := content[nameNode.StartByte():nameNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "namespace_use_declaration":
		// Use statements
		clauseNode := c.findChildNodeByType(node, "namespace_use_clause")
		if clauseNode != nil {
			nameNode := c.findChildNodeByType(clauseNode, "qualified_name")
			if nameNode != nil {
				usePath := content[nameNode.StartByte():nameNode.EndByte()]
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, usePath),
					FromSymbol: file.Path,
					ToSymbol:   usePath,
					Kind:       types.RefKindImport,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: true,
				}
				*refs = append(*refs, ref)
			}
		}

	case "object_creation_expression":
		// new ClassName()
		nameNode := c.findChildNodeByType(node, "name")
		if nameNode != nil {
			typeName := content[nameNode.StartByte():nameNode.EndByte()]
			if typeName != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, typeName),
					FromSymbol: fromSym,
					ToSymbol:   typeName,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractCSharpRefs extracts references from C# code.
func (c *Chunker) extractCSharpRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "invocation_expression":
		// Method calls
		nameNode := c.findChildNodeByType(node, "identifier")
		if nameNode == nil {
			memberNode := c.findChildNodeByType(node, "member_access_expression")
			if memberNode != nil {
				nameNode = c.findChildNodeByType(memberNode, "identifier")
			}
		}
		if nameNode != nil {
			calledFunc := content[nameNode.StartByte():nameNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "object_creation_expression":
		// new ClassName()
		typeNode := c.findChildNodeByType(node, "identifier")
		if typeNode != nil {
			typeName := content[typeNode.StartByte():typeNode.EndByte()]
			if typeName != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, typeName),
					FromSymbol: fromSym,
					ToSymbol:   typeName,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}

	case "using_directive":
		// Using statements
		nameNode := c.findChildNodeByType(node, "qualified_name")
		if nameNode == nil {
			nameNode = c.findChildNodeByType(node, "identifier")
		}
		if nameNode != nil {
			usingPath := content[nameNode.StartByte():nameNode.EndByte()]
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, usingPath),
				FromSymbol: file.Path,
				ToSymbol:   usingPath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "identifier":
		// Type usage in declarations
		parent := node.Parent()
		if parent != nil && (parent.Type() == "variable_declaration" || parent.Type() == "parameter") {
			typeName := content[node.StartByte():node.EndByte()]
			if typeName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractKotlinRefs extracts references from Kotlin code.
func (c *Chunker) extractKotlinRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call_expression":
		// Function calls
		funcNode := c.findChildNodeByType(node, "simple_identifier")
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "import_header":
		// Import statements
		idNode := c.findChildNodeByType(node, "identifier")
		if idNode != nil {
			importPath := content[idNode.StartByte():idNode.EndByte()]
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, importPath),
				FromSymbol: file.Path,
				ToSymbol:   importPath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "user_type":
		// Type usage
		parent := node.Parent()
		if parent != nil && parent.Type() != "class_declaration" && parent.Type() != "interface_declaration" {
			typeNode := c.findChildNodeByType(node, "type_identifier")
			if typeNode != nil {
				typeName := content[typeNode.StartByte():typeNode.EndByte()]
				if typeName != "" && currentFunc != "" {
					ref := &types.Reference{
						ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
						FromSymbol: currentFunc,
						ToSymbol:   typeName,
						Kind:       types.RefKindTypeUse,
						FilePath:   file.Path,
						Line:       line,
						IsExternal: !localSymbols[typeName],
					}
					*refs = append(*refs, ref)
				}
			}
		}
	}
}

// extractSwiftRefs extracts references from Swift code.
func (c *Chunker) extractSwiftRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call_expression":
		// Function calls
		funcNode := c.findChildNodeByType(node, "simple_identifier")
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "import_declaration":
		// Import statements
		pathNode := c.findChildNodeByType(node, "identifier")
		if pathNode != nil {
			importPath := content[pathNode.StartByte():pathNode.EndByte()]
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, importPath),
				FromSymbol: file.Path,
				ToSymbol:   importPath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "type_identifier":
		// Type usage
		parent := node.Parent()
		if parent != nil && parent.Type() != "class_declaration" && parent.Type() != "struct_declaration" && parent.Type() != "protocol_declaration" {
			typeName := content[node.StartByte():node.EndByte()]
			if typeName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractScalaRefs extracts references from Scala code.
func (c *Chunker) extractScalaRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "call_expression":
		// Function calls
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "import_declaration":
		// Import statements
		pathNode := c.findChildNodeByType(node, "stable_identifier")
		if pathNode != nil {
			importPath := content[pathNode.StartByte():pathNode.EndByte()]
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, importPath),
				FromSymbol: file.Path,
				ToSymbol:   importPath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "type_identifier":
		// Type usage
		parent := node.Parent()
		if parent != nil && parent.Type() != "class_definition" && parent.Type() != "trait_definition" && parent.Type() != "object_definition" {
			typeName := content[node.StartByte():node.EndByte()]
			if typeName != "" && currentFunc != "" {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractLuaRefs extracts references from Lua code.
func (c *Chunker) extractLuaRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "function_call":
		// Function calls
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}

	case "method_index_expression":
		// Method calls like obj:method()
		methodNode := c.findChildNodeByType(node, "identifier")
		if methodNode != nil {
			calledFunc := content[methodNode.StartByte():methodNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractSQLRefs extracts references from SQL code.
func (c *Chunker) extractSQLRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "table_reference", "object_reference":
		// Table references in SELECT, INSERT, UPDATE, DELETE
		name := content[node.StartByte():node.EndByte()]
		if name != "" {
			fromSym := currentFunc
			if fromSym == "" {
				fromSym = file.Path
			}
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, name),
				FromSymbol: fromSym,
				ToSymbol:   name,
				Kind:       types.RefKindTypeUse,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: !localSymbols[name],
			}
			*refs = append(*refs, ref)
		}

	case "function_call":
		// Function calls in SQL
		funcNode := c.findChildNodeByType(node, "identifier")
		if funcNode != nil {
			calledFunc := content[funcNode.StartByte():funcNode.EndByte()]
			if calledFunc != "" {
				fromSym := currentFunc
				if fromSym == "" {
					fromSym = file.Path
				}
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:call:%s", file.Path, line, calledFunc),
					FromSymbol: fromSym,
					ToSymbol:   calledFunc,
					Kind:       types.RefKindCall,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[calledFunc],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// extractProtobufRefs extracts references from Protobuf code.
func (c *Chunker) extractProtobufRefs(node *sitter.Node, file *types.SourceFile, content string, localSymbols map[string]bool, refs *[]*types.Reference, currentFunc string) {
	nodeType := node.Type()
	line := int(node.StartPoint().Row) + 1

	switch nodeType {
	case "import":
		// Import statements
		pathNode := c.findChildNodeByType(node, "string")
		if pathNode != nil {
			importPath := content[pathNode.StartByte():pathNode.EndByte()]
			importPath = strings.Trim(importPath, "\"")
			ref := &types.Reference{
				ID:         fmt.Sprintf("%s:%d:import:%s", file.Path, line, importPath),
				FromSymbol: file.Path,
				ToSymbol:   importPath,
				Kind:       types.RefKindImport,
				FilePath:   file.Path,
				Line:       line,
				IsExternal: true,
			}
			*refs = append(*refs, ref)
		}

	case "type":
		// Type references in field definitions
		typeName := content[node.StartByte():node.EndByte()]
		if typeName != "" && currentFunc != "" {
			// Skip built-in types
			builtinTypes := map[string]bool{
				"int32": true, "int64": true, "uint32": true, "uint64": true,
				"sint32": true, "sint64": true, "fixed32": true, "fixed64": true,
				"sfixed32": true, "sfixed64": true, "bool": true, "string": true,
				"bytes": true, "float": true, "double": true,
			}
			if !builtinTypes[typeName] {
				ref := &types.Reference{
					ID:         fmt.Sprintf("%s:%d:type:%s", file.Path, line, typeName),
					FromSymbol: currentFunc,
					ToSymbol:   typeName,
					Kind:       types.RefKindTypeUse,
					FilePath:   file.Path,
					Line:       line,
					IsExternal: !localSymbols[typeName],
				}
				*refs = append(*refs, ref)
			}
		}
	}
}

// chunkEmbeddedLanguage chunks files with embedded JavaScript (HTML, Svelte, PHP).
func (c *Chunker) chunkEmbeddedLanguage(file *types.SourceFile) ([]*types.Chunk, error) {
	extractor := NewEmbeddedJSExtractor()
	defer extractor.Close()

	var scripts []ExtractedScript
	var err error

	switch file.Language {
	case "html", "htm", "xhtml":
		scripts, err = extractor.ExtractFromHTML(file.Content)
	case "svelte":
		scripts, err = extractor.ExtractFromSvelte(file.Content)
	case "php":
		scripts, err = extractor.ExtractFromPHP(file.Content)
	default:
		return nil, fmt.Errorf("unsupported embedded JS language: %s", file.Language)
	}

	if err != nil {
		return nil, err
	}

	return c.ChunkEmbeddedJS(file, scripts)
}

// extractSymbolsFromEmbeddedLanguage extracts symbols from embedded JavaScript.
func (c *Chunker) extractSymbolsFromEmbeddedLanguage(file *types.SourceFile) ([]*types.Symbol, error) {
	extractor := NewEmbeddedJSExtractor()
	defer extractor.Close()

	var scripts []ExtractedScript
	var err error

	switch file.Language {
	case "html", "htm", "xhtml":
		scripts, err = extractor.ExtractFromHTML(file.Content)
	case "svelte":
		scripts, err = extractor.ExtractFromSvelte(file.Content)
	case "php":
		scripts, err = extractor.ExtractFromPHP(file.Content)
	default:
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return c.ExtractSymbolsFromEmbeddedJS(file, scripts)
}

// extractRefsFromEmbeddedLanguage extracts references from embedded JavaScript.
func (c *Chunker) extractRefsFromEmbeddedLanguage(file *types.SourceFile) ([]*types.Reference, error) {
	extractor := NewEmbeddedJSExtractor()
	defer extractor.Close()

	var scripts []ExtractedScript
	var err error

	switch file.Language {
	case "html", "htm", "xhtml":
		scripts, err = extractor.ExtractFromHTML(file.Content)
	case "svelte":
		scripts, err = extractor.ExtractFromSvelte(file.Content)
	case "php":
		scripts, err = extractor.ExtractFromPHP(file.Content)
	default:
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return c.ExtractRefsFromEmbeddedJS(file, scripts)
}

// Close releases resources.
func (c *Chunker) Close() error {
	for _, parser := range c.parsers {
		parser.Close()
	}
	return nil
}

// Ensure Chunker implements ChunkingStrategy interface
var _ provider.ChunkingStrategy = (*Chunker)(nil)
