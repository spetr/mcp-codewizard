// Package json provides JSON language support for tree-sitter.
// Grammar source: https://github.com/tree-sitter/tree-sitter-json
package json

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_json();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the JSON tree-sitter language.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_json())
	return sitter.NewLanguage(ptr)
}
