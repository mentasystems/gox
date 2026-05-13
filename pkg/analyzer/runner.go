package analyzer

import (
	"fmt"
	"os"
	"sort"

	"github.com/mentasystems/gox/internal/astutil"
	"github.com/mentasystems/gox/pkg/loader"
)

// RunOptions controls the behavior of Run.
type RunOptions struct {
	UseCache bool
	// CacheGet/Put are injected to avoid the analyzer package depending on
	// pkg/cache (which itself imports pkg/analyzer for the Issue type). The
	// CLI wires them up.
	CacheKey func(info *loader.PackageInfo) (string, error)
	CacheGet func(key string) ([]Issue, bool)
	CachePut func(key string, issues []Issue) error
}

// Stats reports cache hit/miss counts for a Run.
type Stats struct {
	PackagesTotal int
	CacheHits     int
	CacheMisses   int
}

// Run loads (or cache-replays) every package matched by patterns and applies
// every registered analyzer.
func Run(patterns []string, analyzers []*Analyzer, opts RunOptions) ([]Issue, Stats, error) {
	infos, listErr := loader.List(patterns...)
	if listErr != nil {
		return nil, Stats{}, listErr
	}

	var issues []Issue
	stats := Stats{PackagesTotal: len(infos)}

	for _, info := range infos {
		// Cache lookup
		var cacheKey string
		if opts.UseCache && opts.CacheKey != nil && opts.CacheGet != nil {
			key, keyErr := opts.CacheKey(info)
			if keyErr == nil {
				cacheKey = key
				if cached, ok := opts.CacheGet(key); ok {
					issues = append(issues, cached...)
					stats.CacheHits++
					continue
				}
			} else {
				fmt.Fprintf(os.Stderr, "gox cache: key %s: %v\n", info.ImportPath, keyErr)
			}
		}

		// Cache miss → parse + type-check + analyze
		pkg, loadErr := loader.LoadPackage(info)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "gox: %s: %v\n", info.ImportPath, loadErr)
			continue
		}

		// Precompute generated-file set so analyzers can still see all symbols
		// (needed for cross-file type info) but issues reported from generated
		// files are dropped.
		generated := map[string]bool{}
		for _, path := range info.AbsFiles() {
			if astutil.IsGenerated(path) {
				generated[path] = true
			}
		}

		var pkgIssues []Issue
		report := func(i Issue) {
			if generated[i.Pos.Filename] {
				return
			}
			pkgIssues = append(pkgIssues, i)
		}
		pass := &Pass{
			Fset:      pkg.Fset,
			Pkg:       pkg.Pkg,
			TypesInfo: pkg.TypesInfo,
			Files:     pkg.Files,
			Report:    report,
		}
		for _, a := range analyzers {
			a.Run(pass)
		}
		issues = append(issues, pkgIssues...)
		stats.CacheMisses++

		if opts.UseCache && opts.CachePut != nil && cacheKey != "" {
			if putErr := opts.CachePut(cacheKey, pkgIssues); putErr != nil {
				fmt.Fprintf(os.Stderr, "gox cache: put %s: %v\n", info.ImportPath, putErr)
			}
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i].Pos, issues[j].Pos
		if a.Filename != b.Filename {
			return a.Filename < b.Filename
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return issues[i].Analyzer < issues[j].Analyzer
	})
	return issues, stats, nil
}
