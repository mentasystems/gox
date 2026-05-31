package namedargs_test

import (
	"testing"

	"github.com/mentasystems/gox/pkg/analyzer"
	"github.com/mentasystems/gox/pkg/analyzer/analyzertest"
)

func TestNamedArgs_twoStringsAdjacent(t *testing.T) {
	const src = `package p
func transfer(userID, orderID string) {}
func _() {
	transfer("u-1", "o-2")
}`
	issues := analyzertest.Run(t, get(), src)
	// Both args are flagged because they're adjacent same-type without comments.
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2", len(issues))
	}
}

func TestNamedArgs_inlineCommentsSatisfy(t *testing.T) {
	const src = `package p
func transfer(userID, orderID string) {}
func _() {
	transfer( /* userID */ "u-1" /* orderID */, "o-2")
}`
	analyzertest.AssertNone(t, analyzertest.Run(t, get(), src))
}

func TestNamedArgs_identNameMatchesParam(t *testing.T) {
	const src = `package p
func transfer(userID, orderID string) {}
func _() {
	var userID, orderID string
	transfer(userID, orderID)
}`
	analyzertest.AssertNone(t, analyzertest.Run(t, get(), src))
}

func TestNamedArgs_distinctTypesIsFine(t *testing.T) {
	const src = `package p
func open(path string, perm int) {}
func _() {
	open("/x", 644)
}`
	analyzertest.AssertNone(t, analyzertest.Run(t, get(), src))
}

func TestNamedArgs_stdlibCalleeExempt(t *testing.T) {
	const src = `package p
import "strings"
func _() {
	_ = strings.HasPrefix("hello world", "hello")
}`
	analyzertest.AssertNone(t, analyzertest.Run(t, get(), src))
}

func TestNamedArgs_safeIgnoreSameLine(t *testing.T) {
	const src = `package p
func transfer(userID, orderID string) {}
func _() {
	transfer("u-1", "o-2") // safe-ignore: positional args fixed by call site
}`
	analyzertest.AssertNone(t, analyzertest.Run(t, get(), src))
}

func TestNamedArgs_safeIgnoreOnClosingLine(t *testing.T) {
	const src = `package p
func transfer(userID, orderID string) {}
func _() {
	transfer(
		"u-1",
		"o-2",
	) // safe-ignore: positional args fixed by call site
}`
	analyzertest.AssertNone(t, analyzertest.Run(t, get(), src))
}

func TestNamedArgs_safeIgnoreWithoutReasonStillFires(t *testing.T) {
	const src = `package p
func transfer(userID, orderID string) {}
func _() {
	transfer("u-1", "o-2") // safe-ignore:
}`
	issues := analyzertest.Run(t, get(), src)
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2 (bare safe-ignore must not suppress)", len(issues))
	}
}

func get() *analyzer.Analyzer {
	for _, a := range analyzer.All() {
		if a.Name == "namedargs" {
			return a
		}
	}
	panic("namedargs analyzer not registered")
}
