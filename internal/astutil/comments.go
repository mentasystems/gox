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
