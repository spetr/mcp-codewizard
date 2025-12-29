// Package pascal provides Pascal language support for tree-sitter.
// Grammar source: https://github.com/Isopod/tree-sitter-pascal
// Supports Pascal, Delphi, and FreePascal dialects.
package pascal

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_pascal();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the Pascal tree-sitter language.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_pascal())
	return sitter.NewLanguage(ptr)
}
