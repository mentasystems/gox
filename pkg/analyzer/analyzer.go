// Package analyzer defines the core types every gox analyzer implements.
//
// An Analyzer inspects a parsed and type-checked Go package and reports
// Issues. The runner (pkg/loader) is responsible for parsing, type-checking,
// and invoking every registered analyzer.
package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Issue is a single problem reported by an analyzer.
type Issue struct {
	Analyzer string
	Pos      token.Position
	Message  string
	Hint     string
}

// Pass is the input given to an analyzer.
type Pass struct {
	Fset      *token.FileSet
	Pkg       *types.Package
	TypesInfo *types.Info
	Files     []*ast.File
	Report    func(Issue)
}

// Analyzer is a single rule.
type Analyzer struct {
	Name string
	Doc  string
	Run  func(*Pass)
}

// Annotation prefixes recognised on trailing line comments to opt out of a rule.
//
// Convention: //<prefix> <reason>
const (
	AnnSafeIgnore = "safe-ignore:"
	AnnGlobalOK   = "global-ok:"
	AnnAnyOK      = "any-ok:"
	AnnPanicOK    = "panic-ok:"
	AnnTimeoutOK  = "timeout-ok:"
)

// HasAnnotation reports whether a line comment group contains the given prefix
// and the prefix is followed by a non-empty reason.
func HasAnnotation(cg *ast.CommentGroup, prefix string) bool {
	if cg == nil {
		return false
	}
	for _, c := range cg.List {
		text := c.Text
		if len(text) >= 2 && text[:2] == "//" {
			text = text[2:]
		}
		// trim leading spaces
		i := 0
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
		text = text[i:]
		if len(text) >= len(prefix) && text[:len(prefix)] == prefix {
			rest := text[len(prefix):]
			// require at least one non-space char as the reason
			for _, r := range rest {
				if r != ' ' && r != '\t' {
					return true
				}
			}
		}
	}
	return false
}
