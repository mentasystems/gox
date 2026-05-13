package shadow_test

import (
	"testing"

	"github.com/mentasystems/gox/pkg/analyzer"
	"github.com/mentasystems/gox/pkg/analyzer/analyzertest"
)

func TestShadow_errShadow(t *testing.T) {
	const src = `package p
func mayFail() error { return nil }
func _() {
	err := mayFail()
	if err != nil {
		err := mayFail()
		_ = err
	}
	_ = err
}`
	issues := analyzertest.Run(t, get(), src)
	analyzertest.AssertLines(t, issues, []int{6})
}

func TestShadow_okIsExempt(t *testing.T) {
	const src = `package p
func _() {
	var m = map[string]int{"a": 1}
	v, ok := m["a"]
	if ok {
		v, ok := m["b"]
		_ = v
		_ = ok
	}
	_ = v
}`
	issues := analyzertest.Run(t, get(), src)
	// `ok` is whitelisted; the only shadow flagged should be `v`.
	analyzertest.AssertLines(t, issues, []int{6})
}

func TestShadow_sameScopeRedeclareIsNotShadow(t *testing.T) {
	const src = `package p
func mayFail() error { return nil }
func _() {
	err := mayFail()
	if err != nil {
		return
	}
	x, err := 1, mayFail() // same scope, mixed redeclare — not shadow
	_ = x
	_ = err
}`
	analyzertest.AssertNone(t, analyzertest.Run(t, get(), src))
}

// get returns the registered analyzer by name.
func get() *analyzer.Analyzer {
	for _, a := range analyzer.All() {
		if a.Name == "shadow" {
			return a
		}
	}
	panic("shadow analyzer not registered")
}
