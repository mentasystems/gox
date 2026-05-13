// Package namedargs requires call sites to name their arguments when two or
// more consecutive parameters share the same type.
//
// Rationale: `transfer(userID, orderID)` and `transfer(orderID, userID)` are
// both compilable Go but mean opposite things; this is one of the highest-
// frequency silent bug classes when code is written without supervision.
//
// Required form at the call site:
//
//	transfer(/* userID */ a, /* orderID */ b)
//
// The comment text must match the parameter name from the declaration. The
// rule fires only when the function has 2+ consecutive parameters of the same
// underlying type (or one is assignable to the other), so calls like
// `Add(2, 3)` to a `func Add(int, int)` still trigger but `Open(path)` does
// not.
//
// Variadic, single-typed-group calls (e.g. `fmt.Println(a, b, c)`) are
// exempt: the parameters share a type by design and the call site does not
// distinguish them.
package namedargs

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/mentasystems/gox/pkg/analyzer"
)

func init() {
	analyzer.Register(&analyzer.Analyzer{
		Name: "namedargs",
		Doc:  "requires inline `/* paramName */` comments when consecutive params share a type",
		Run:  run,
	})
}

func run(pass *analyzer.Pass) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			checkCall(pass, file, call)
			return true
		})
	}
}

func checkCall(pass *analyzer.Pass, file *ast.File, call *ast.CallExpr) {
	sig := calleeSignature(pass, call)
	if sig == nil {
		return
	}
	params := sig.Params()
	if params.Len() < 2 {
		return
	}

	// Compute parameter names — skip if any are blank/anonymous (we have nothing to match against).
	names := make([]string, params.Len())
	for i := 0; i < params.Len(); i++ {
		n := params.At(i).Name()
		if n == "" || n == "_" {
			return
		}
		names[i] = n
	}

	// Find consecutive groups of compatible-typed params.
	flagged := make([]bool, params.Len())
	for i := 0; i < params.Len()-1; i++ {
		ti := params.At(i).Type()
		tj := params.At(i + 1).Type()
		if compatible(ti, tj) {
			flagged[i] = true
			flagged[i+1] = true
		}
	}
	// Variadic last group of same type: exempt the variadic position itself.
	if sig.Variadic() {
		flagged[params.Len()-1] = false
	}

	anyFlagged := false
	for _, b := range flagged {
		if b {
			anyFlagged = true
			break
		}
	}
	if !anyFlagged {
		return
	}

	// Match arguments to parameters (variadic flattens the tail).
	for i, arg := range call.Args {
		if i >= params.Len() {
			break // variadic tail
		}
		if !flagged[i] {
			continue
		}
		// Trivially-named arguments (an identifier whose source name matches
		// the parameter name) are self-documenting and don't need a comment.
		if id, ok := arg.(*ast.Ident); ok && id.Name == names[i] {
			continue
		}
		wantedName := names[i]
		got := leadingArgComment(pass.Fset, file, arg.Pos())
		if got == wantedName {
			continue
		}
		pass.Report(analyzer.Issue{
			Analyzer: "namedargs",
			Pos:      pass.Fset.Position(arg.Pos()),
			Message:  fmt.Sprintf("argument shares a type with an adjacent argument; prefix with /* %s */", wantedName),
			Hint:     fmt.Sprintf("change to: /* %s */ %s", wantedName, exprSrc(pass.Fset, arg)),
		})
	}
}

// calleeSignature returns the *types.Signature for the callee of a CallExpr,
// or nil for type conversions / built-in calls we don't want to enforce.
func calleeSignature(pass *analyzer.Pass, call *ast.CallExpr) *types.Signature {
	tv, ok := pass.TypesInfo.Types[call.Fun]
	if !ok {
		return nil
	}
	if tv.IsType() {
		return nil // T(x) conversion
	}
	if tv.IsBuiltin() {
		return nil
	}
	sig, ok := tv.Type.(*types.Signature)
	if !ok {
		return nil
	}
	if isStdlibCallee(pass, call) {
		// Stdlib has stable conventions every Go developer (and trained model)
		// knows by heart. Enforcing namedargs there is pure noise.
		return nil
	}
	return sig
}

// isStdlibCallee reports whether call.Fun resolves to a function/method whose
// package belongs to the Go standard library.
func isStdlibCallee(pass *analyzer.Pass, call *ast.CallExpr) bool {
	var obj types.Object
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		obj = pass.TypesInfo.ObjectOf(fn.Sel)
	case *ast.Ident:
		obj = pass.TypesInfo.ObjectOf(fn)
	}
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	path := obj.Pkg().Path()
	// Heuristic: stdlib import paths never contain a dot in their first
	// segment (they look like "fmt", "net/http", "encoding/json"). Third-party
	// paths start with a domain ("github.com/...").
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			return true // reached a segment boundary without seeing a dot
		}
		if path[i] == '.' {
			return false
		}
	}
	return true // single segment, no dot
}

// compatible reports whether swapping two adjacent argument positions would
// silently compile. We restrict the rule to basic types (string, integer,
// float, bool) — pointers and structs that share a kind are almost always
// caught by the compiler the moment they are swapped, so the noise/value
// trade-off there is poor.
func compatible(a, b types.Type) bool {
	ba, aOK := a.Underlying().(*types.Basic)
	bb, bOK := b.Underlying().(*types.Basic)
	if !aOK || !bOK {
		return false
	}
	if !isBugProneBasic(ba) || !isBugProneBasic(bb) {
		return false
	}
	// Same basic kind (e.g. both string, both int) — bug-prone if swapped.
	return ba.Kind() == bb.Kind() ||
		(isIntegerBasic(ba) && isIntegerBasic(bb)) ||
		(isFloatBasic(ba) && isFloatBasic(bb))
}

func isBugProneBasic(b *types.Basic) bool {
	info := b.Info()
	if info&types.IsString != 0 {
		return true
	}
	if info&types.IsNumeric != 0 {
		return true
	}
	if info&types.IsBoolean != 0 {
		return true
	}
	return false
}

func isIntegerBasic(b *types.Basic) bool { return b.Info()&types.IsInteger != 0 }
func isFloatBasic(b *types.Basic) bool   { return b.Info()&types.IsFloat != 0 }

// leadingArgComment returns the trimmed text of a single /* ... */ comment
// that immediately precedes `pos` on the same line, or "".
func leadingArgComment(fset *token.FileSet, file *ast.File, pos token.Pos) string {
	line := fset.Position(pos).Line
	var best *ast.CommentGroup
	for _, cg := range file.Comments {
		end := fset.Position(cg.End())
		if end.Line != line {
			continue
		}
		if cg.End() > pos {
			continue
		}
		if best == nil || cg.End() > best.End() {
			best = cg
		}
	}
	if best == nil {
		return ""
	}
	// Take the last comment in the group, expect /* ... */ form
	c := best.List[len(best.List)-1]
	if !strings.HasPrefix(c.Text, "/*") || !strings.HasSuffix(c.Text, "*/") {
		return ""
	}
	inner := strings.TrimSpace(c.Text[2 : len(c.Text)-2])
	return inner
}

// exprSrc renders an expression position back to a short hint string.
func exprSrc(_ *token.FileSet, e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return "..."
}
