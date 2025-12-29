// Package powershell provides PowerShell language support for tree-sitter.
// Grammar source: https://github.com/airbus-cert/tree-sitter-powershell
// Supports PowerShell scripting language for Windows automation.
package powershell

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_powershell();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the PowerShell tree-sitter language.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_powershell())
	return sitter.NewLanguage(ptr)
}
