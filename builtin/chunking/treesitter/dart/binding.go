// Package dart provides Dart language support for tree-sitter.
// Grammar source: https://github.com/UserNobody14/tree-sitter-dart
// Supports Dart language including Flutter.
package dart

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_dart();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the Dart tree-sitter language.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_dart())
	return sitter.NewLanguage(ptr)
}
