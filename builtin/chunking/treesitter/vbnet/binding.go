// Package vbnet provides Visual Basic .NET language support for tree-sitter.
// Grammar source: https://github.com/CodeAnt-AI/tree-sitter-vb-dotnet
// Supports VB 16.9 / .NET 5 syntax.
package vbnet

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_vb_dotnet();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the VB.NET tree-sitter language.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_vb_dotnet())
	return sitter.NewLanguage(ptr)
}
