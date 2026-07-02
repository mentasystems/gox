package analyzer

import "sort"

// global-ok: analyzer registration is a process-level singleton by design;
// each analyzer's init() populates this map at program start.
var registered = map[string]*Analyzer{}

// Register adds an analyzer to the global registry. Panics on duplicate names
// so a build-time mistake is loud.
func Register(a *Analyzer) {
	if _, dup := registered[a.Name]; dup {
		panic("gox: duplicate analyzer name: " + a.Name)
	}
	registered[a.Name] = a
}

// Defaults returns every registered analyzer that runs by default (OptIn
// false), sorted by name.
func Defaults() []*Analyzer {
	out := make([]*Analyzer, 0, len(registered))
	for _, a := range registered {
		if !a.OptIn {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// All returns every registered analyzer sorted by name.
func All() []*Analyzer {
	out := make([]*Analyzer, 0, len(registered))
	for _, a := range registered {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
