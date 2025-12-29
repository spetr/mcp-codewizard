// Package treesitter implements chunking using Tree-sitter for AST-aware splitting.
package treesitter

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/svelte"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

// ExtractedScript represents JavaScript code extracted from an embedded context.
type ExtractedScript struct {
	Content   string
	StartLine int
	EndLine   int
	Type      string // "inline", "module", "event-handler"
}

// EmbeddedJSExtractor extracts JavaScript from files that embed it.
type EmbeddedJSExtractor struct {
	htmlParser    *sitter.Parser
	svelteParser  *sitter.Parser
	phpParser     *sitter.Parser
	jsParser      *sitter.Parser
}

// NewEmbeddedJSExtractor creates a new extractor for embedded JavaScript.
func NewEmbeddedJSExtractor() *EmbeddedJSExtractor {
	htmlParser := sitter.NewParser()
	htmlParser.SetLanguage(html.GetLanguage())

	svelteParser := sitter.NewParser()
	svelteParser.SetLanguage(svelte.GetLanguage())

	phpParser := sitter.NewParser()
	phpParser.SetLanguage(php.GetLanguage())

	jsParser := sitter.NewParser()
	jsParser.SetLanguage(javascript.GetLanguage())

	return &EmbeddedJSExtractor{
		htmlParser:   htmlParser,
		svelteParser: svelteParser,
		phpParser:    phpParser,
		jsParser:     jsParser,
	}
}

// Close releases parser resources.
func (e *EmbeddedJSExtractor) Close() {
	if e.htmlParser != nil {
		e.htmlParser.Close()
	}
	if e.svelteParser != nil {
		e.svelteParser.Close()
	}
	if e.phpParser != nil {
		e.phpParser.Close()
	}
	if e.jsParser != nil {
		e.jsParser.Close()
	}
}

// ExtractFromHTML extracts JavaScript from HTML files.
func (e *EmbeddedJSExtractor) ExtractFromHTML(content []byte) ([]ExtractedScript, error) {
	tree, err := e.htmlParser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	defer tree.Close()

	return e.extractScriptsFromHTMLTree(tree.RootNode(), string(content)), nil
}

// ExtractFromSvelte extracts JavaScript from Svelte files.
func (e *EmbeddedJSExtractor) ExtractFromSvelte(content []byte) ([]ExtractedScript, error) {
	tree, err := e.svelteParser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Svelte: %w", err)
	}
	defer tree.Close()

	scripts := e.extractScriptsFromHTMLTree(tree.RootNode(), string(content))

	// Also extract Svelte expressions {expression}
	expressions := e.extractSvelteExpressions(tree.RootNode(), string(content))
	scripts = append(scripts, expressions...)

	return scripts, nil
}

// ExtractFromPHP extracts JavaScript from PHP files.
// Handles <?php ?>, <? ?> (short tags), and <?= ?> (echo shorthand).
func (e *EmbeddedJSExtractor) ExtractFromPHP(content []byte) ([]ExtractedScript, error) {
	tree, err := e.phpParser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PHP: %w", err)
	}
	defer tree.Close()

	return e.extractScriptsFromPHPTree(tree.RootNode(), string(content)), nil
}

// extractScriptsFromHTMLTree extracts scripts from HTML/Svelte AST.
func (e *EmbeddedJSExtractor) extractScriptsFromHTMLTree(root *sitter.Node, content string) []ExtractedScript {
	var scripts []ExtractedScript

	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		nodeType := node.Type()

		// Look for script_element nodes
		if nodeType == "script_element" {
			script := e.extractScriptElement(node, content)
			if script != nil && len(strings.TrimSpace(script.Content)) > 0 {
				scripts = append(scripts, *script)
			}
		}

		// Recurse into children
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)
	return scripts
}

// extractScriptElement extracts content from a script_element node.
func (e *EmbeddedJSExtractor) extractScriptElement(node *sitter.Node, content string) *ExtractedScript {
	var rawText *sitter.Node
	var scriptType string = "inline"

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "raw_text":
			rawText = child
		case "start_tag":
			// Check for type="module" attribute
			for j := 0; j < int(child.ChildCount()); j++ {
				attr := child.Child(j)
				if attr.Type() == "attribute" {
					attrContent := content[attr.StartByte():attr.EndByte()]
					if strings.Contains(attrContent, "module") {
						scriptType = "module"
					}
				}
			}
		}
	}

	if rawText == nil {
		return nil
	}

	scriptContent := content[rawText.StartByte():rawText.EndByte()]
	// Trim leading newline if present (common in formatted HTML)
	scriptContent = strings.TrimPrefix(scriptContent, "\n")

	return &ExtractedScript{
		Content:   scriptContent,
		StartLine: int(rawText.StartPoint().Row) + 1,
		EndLine:   int(rawText.EndPoint().Row) + 1,
		Type:      scriptType,
	}
}

// extractSvelteExpressions extracts JavaScript from Svelte {expression} syntax.
func (e *EmbeddedJSExtractor) extractSvelteExpressions(root *sitter.Node, content string) []ExtractedScript {
	var scripts []ExtractedScript

	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		// Look for expression nodes with raw_text_expr
		if node.Type() == "expression" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "raw_text_expr" {
					exprContent := content[child.StartByte():child.EndByte()]
					// Only include non-trivial expressions
					if len(exprContent) > 20 || strings.Contains(exprContent, "(") {
						scripts = append(scripts, ExtractedScript{
							Content:   exprContent,
							StartLine: int(child.StartPoint().Row) + 1,
							EndLine:   int(child.EndPoint().Row) + 1,
							Type:      "expression",
						})
					}
				}
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)
	return scripts
}

// extractScriptsFromPHPTree extracts scripts from PHP AST.
// PHP files have HTML in text_interpolation -> text nodes.
// Script tags may be interrupted by PHP code (<?php ?>, <? ?>, <?= ?>).
func (e *EmbeddedJSExtractor) extractScriptsFromPHPTree(root *sitter.Node, content string) []ExtractedScript {
	// Collect all text segments (HTML parts between PHP code)
	var textSegments []textSegment
	e.collectTextSegments(root, content, &textSegments)

	// Parse text segments to find script tags
	return e.findScriptsInTextSegments(textSegments, content)
}

type textSegment struct {
	content   string
	startByte uint32
	endByte   uint32
	startLine int
	endLine   int
}

// collectTextSegments collects all text nodes from PHP AST.
func (e *EmbeddedJSExtractor) collectTextSegments(node *sitter.Node, content string, segments *[]textSegment) {
	if node.Type() == "text" {
		seg := textSegment{
			content:   content[node.StartByte():node.EndByte()],
			startByte: node.StartByte(),
			endByte:   node.EndByte(),
			startLine: int(node.StartPoint().Row) + 1,
			endLine:   int(node.EndPoint().Row) + 1,
		}
		*segments = append(*segments, seg)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		e.collectTextSegments(node.Child(i), content, segments)
	}
}

// scriptTagRegex matches script opening and closing tags.
var (
	scriptOpenRegex  = regexp.MustCompile(`(?i)<script[^>]*>`)
	scriptCloseRegex = regexp.MustCompile(`(?i)</script>`)
)

// findScriptsInTextSegments finds <script> tags across text segments.
// Handles cases where PHP code interrupts a script tag.
func (e *EmbeddedJSExtractor) findScriptsInTextSegments(segments []textSegment, fullContent string) []ExtractedScript {
	var scripts []ExtractedScript

	inScript := false
	var scriptStartLine int
	var scriptContent strings.Builder
	var scriptType string

	for _, seg := range segments {
		remaining := seg.content
		lineOffset := seg.startLine

		for len(remaining) > 0 {
			if !inScript {
				// Look for <script> opening tag
				loc := scriptOpenRegex.FindStringIndex(remaining)
				if loc == nil {
					break // No script tag in this segment
				}

				// Found opening tag
				tagContent := remaining[loc[0]:loc[1]]
				if strings.Contains(strings.ToLower(tagContent), "module") {
					scriptType = "module"
				} else {
					scriptType = "inline"
				}

				// Count newlines before tag for accurate line number
				lineOffset += strings.Count(remaining[:loc[0]], "\n")
				scriptStartLine = lineOffset

				inScript = true
				remaining = remaining[loc[1]:]

				// Count newlines in tag itself
				lineOffset += strings.Count(tagContent, "\n")

			} else {
				// Look for </script> closing tag
				loc := scriptCloseRegex.FindStringIndex(remaining)
				if loc == nil {
					// No closing tag in this segment - script continues
					// This handles PHP interrupting a script:
					// <script>
					// const x = <?= $data ?>;  <- PHP here
					// </script>
					scriptContent.WriteString(remaining)
					break
				}

				// Found closing tag
				scriptContent.WriteString(remaining[:loc[0]])

				// Calculate end line
				endLine := lineOffset + strings.Count(remaining[:loc[0]], "\n")

				content := strings.TrimSpace(scriptContent.String())
				if len(content) > 0 {
					scripts = append(scripts, ExtractedScript{
						Content:   content,
						StartLine: scriptStartLine,
						EndLine:   endLine,
						Type:      scriptType,
					})
				}

				// Reset state
				inScript = false
				scriptContent.Reset()
				remaining = remaining[loc[1]:]
				lineOffset += strings.Count(remaining[:loc[1]], "\n")
			}
		}
	}

	return scripts
}

// ChunkEmbeddedJS creates chunks from extracted JavaScript, using the JS parser.
func (c *Chunker) ChunkEmbeddedJS(file *types.SourceFile, scripts []ExtractedScript) ([]*types.Chunk, error) {
	var allChunks []*types.Chunk
	maxChars := c.config.MaxChunkSize * CharsPerToken

	for _, script := range scripts {
		if len(strings.TrimSpace(script.Content)) == 0 {
			continue
		}

		// Create a virtual file for the JS content
		jsFile := &types.SourceFile{
			Path:     file.Path,
			Content:  []byte(script.Content),
			Language: "javascript",
		}

		// Parse and chunk the JavaScript
		chunks, err := c.chunkJavaScript(jsFile, script.StartLine)
		if err != nil {
			// If parsing fails, create a single chunk for the entire script
			// Truncate if necessary to avoid embedding failures
			content := script.Content
			name := fmt.Sprintf("script_%s", script.Type)
			if len(content) > maxChars {
				content = content[:maxChars]
				name += " (truncated)"
			}
			chunks = []*types.Chunk{
				{
					ID:        fmt.Sprintf("%s:%d:embedded", file.Path, script.StartLine),
					FilePath:  file.Path,
					Language:  "javascript",
					Content:   content,
					ChunkType: types.ChunkTypeBlock,
					Name:      name,
					StartLine: script.StartLine,
					EndLine:   script.EndLine,
				},
			}
		}

		// Adjust line numbers to match original file
		for _, chunk := range chunks {
			chunk.StartLine += script.StartLine - 1
			chunk.EndLine += script.StartLine - 1
		}

		allChunks = append(allChunks, chunks...)
	}

	return allChunks, nil
}

// chunkJavaScript chunks JavaScript content.
func (c *Chunker) chunkJavaScript(file *types.SourceFile, baseLineOffset int) ([]*types.Chunk, error) {
	parser, _, ok := c.getParser("javascript")
	if !ok {
		return nil, fmt.Errorf("JavaScript parser not available")
	}
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, file.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JavaScript: %w", err)
	}
	defer tree.Close()

	content := string(file.Content)
	lines := strings.Split(content, "\n")
	maxChars := c.config.MaxChunkSize * CharsPerToken

	var chunks []*types.Chunk
	c.walkNode(tree.RootNode(), file, lines, maxChars, &chunks, "")

	return chunks, nil
}

// ExtractSymbolsFromEmbeddedJS extracts symbols from embedded JavaScript.
func (c *Chunker) ExtractSymbolsFromEmbeddedJS(file *types.SourceFile, scripts []ExtractedScript) ([]*types.Symbol, error) {
	var allSymbols []*types.Symbol

	for _, script := range scripts {
		if len(strings.TrimSpace(script.Content)) == 0 {
			continue
		}

		jsFile := &types.SourceFile{
			Path:     file.Path,
			Content:  []byte(script.Content),
			Language: "javascript",
		}

		symbols, err := c.ExtractSymbols(jsFile)
		if err != nil {
			continue
		}

		// Adjust line numbers
		for _, sym := range symbols {
			sym.StartLine += script.StartLine - 1
			sym.EndLine += script.StartLine - 1
			sym.ID = fmt.Sprintf("%s:%s:%d", file.Path, sym.Name, sym.StartLine)
		}

		allSymbols = append(allSymbols, symbols...)
	}

	return allSymbols, nil
}

// ExtractRefsFromEmbeddedJS extracts references from embedded JavaScript.
func (c *Chunker) ExtractRefsFromEmbeddedJS(file *types.SourceFile, scripts []ExtractedScript) ([]*types.Reference, error) {
	var allRefs []*types.Reference

	for _, script := range scripts {
		if len(strings.TrimSpace(script.Content)) == 0 {
			continue
		}

		jsFile := &types.SourceFile{
			Path:     file.Path,
			Content:  []byte(script.Content),
			Language: "javascript",
		}

		refs, err := c.ExtractReferences(jsFile)
		if err != nil {
			continue
		}

		// Adjust line numbers
		for _, ref := range refs {
			ref.Line += script.StartLine - 1
			ref.ID = fmt.Sprintf("%s:%d:%s:%s", file.Path, ref.Line, ref.Kind, ref.ToSymbol)
		}

		allRefs = append(allRefs, refs...)
	}

	return allRefs, nil
}

// IsEmbeddedJSLanguage returns true if the language embeds JavaScript.
func IsEmbeddedJSLanguage(lang string) bool {
	switch lang {
	case "html", "htm", "xhtml", "svelte", "php":
		return true
	}
	return false
}
