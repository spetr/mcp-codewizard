// Package r provides R language support for tree-sitter.
// Grammar source: https://github.com/r-lib/tree-sitter-r
// Supports R programming language for statistical computing and data science.
package r

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_r();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the R tree-sitter language.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_r())
	return sitter.NewLanguage(ptr)
}
