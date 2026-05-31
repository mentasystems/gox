// Package astutil contains small helpers shared by multiple analyzers.
package astutil

import (
	"go/ast"
	"go/token"
)

// TrailingComment returns the line comment that appears on the same source
// line as `endPos` in the given file, if any.
func TrailingComment(fset *token.FileSet, file *ast.File, endPos token.Pos) *ast.CommentGroup {
	endLine := fset.Position(endPos).Line
	for _, cg := range file.Comments {
		cgLine := fset.Position(cg.Pos()).Line
		if cgLine == endLine && cg.Pos() >= endPos {
			return cg
		}
	}
	return nil
}

// LineComments returns every comment group that appears on the same source
// line as `pos` in the given file. Unlike TrailingComment it does not require
// the comment to start at or after `pos`, so it also matches annotations whose
// reported position is in the middle of the line (e.g. an identifier inside an
// `if`-init clause, or an argument column inside a call expression).
func LineComments(fset *token.FileSet, file *ast.File, pos token.Pos) []*ast.CommentGroup {
	line := fset.Position(pos).Line
	var out []*ast.CommentGroup
	for _, cg := range file.Comments {
		if fset.Position(cg.End()).Line == line || fset.Position(cg.Pos()).Line == line {
			out = append(out, cg)
		}
	}
	return out
}

// FileFor returns the *ast.File containing `pos`, or nil.
func FileFor(fset *token.FileSet, files []*ast.File, pos token.Pos) *ast.File {
	tf := fset.File(pos)
	if tf == nil {
		return nil
	}
	name := tf.Name()
	for _, f := range files {
		ff := fset.File(f.Pos())
		if ff != nil && ff.Name() == name {
			return f
		}
	}
	return nil
}
