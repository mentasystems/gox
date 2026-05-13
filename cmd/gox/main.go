// gox is a strict static analyzer for Go.
//
// Usage:
//
//	gox check [flags] [packages...]   # run all analyzers, exit 1 on any issue
//	gox list                          # list registered analyzers
//	gox build [args...]               # gox check && go build
//	gox test  [args...]               # gox check && go test
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/mentasystems/gox/pkg/analyzer"
	"github.com/mentasystems/gox/pkg/cache"
	"github.com/mentasystems/gox/pkg/loader"

	// register all analyzers
	_ "github.com/mentasystems/gox/pkg/analyzers/banany"
	_ "github.com/mentasystems/gox/pkg/analyzers/bodyclose"
	_ "github.com/mentasystems/gox/pkg/analyzers/contextcheck"
	_ "github.com/mentasystems/gox/pkg/analyzers/errcheck"
	_ "github.com/mentasystems/gox/pkg/analyzers/exhaustive"
	_ "github.com/mentasystems/gox/pkg/analyzers/forcetypeassert"
	_ "github.com/mentasystems/gox/pkg/analyzers/goroutine"
	_ "github.com/mentasystems/gox/pkg/analyzers/namedargs"
	_ "github.com/mentasystems/gox/pkg/analyzers/noglobals"
	_ "github.com/mentasystems/gox/pkg/analyzers/shadow"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "check":
		os.Exit(runCheck(args))
	case "list":
		runList()
	case "build":
		if code := runCheck(nil); code != 0 {
			os.Exit(code)
		}
		os.Exit(runGo("build", args))
	case "test":
		if code := runCheck(nil); code != 0 {
			os.Exit(code)
		}
		os.Exit(runGo("test", args))
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "gox: unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `gox: strict static analyzer for Go

Usage:
  gox check [flags] [packages...]
                            run all analyzers; exit 1 on any issue
  gox list                  list registered analyzers
  gox build [args...]       run check, then go build
  gox test  [args...]       run check, then go test

check flags:
  --no-cache                disable the incremental cache
  --stats                   print cache hit/miss counts and elapsed time
`)
}

func runList() {
	for _, a := range analyzer.All() {
		fmt.Printf("%-22s %s\n", a.Name, a.Doc)
	}
}

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	noCache := fs.Bool("no-cache", false, "disable the incremental cache")
	stats := fs.Bool("stats", false, "print cache hit/miss counts and elapsed time")
	if parseErr := fs.Parse(args); parseErr != nil {
		return 2
	}
	patterns := fs.Args()

	analyzers := analyzer.All()
	opts := analyzer.RunOptions{UseCache: !*noCache}

	if opts.UseCache {
		dir, dirErr := cache.Dir()
		if dirErr != nil {
			fmt.Fprintf(os.Stderr, "gox: cache dir: %v\n", dirErr)
			opts.UseCache = false
		} else {
			version := cache.AnalyzersVersion(analyzers)
			opts.CacheKey = func(info *loader.PackageInfo) (string, error) {
				return cache.Key(info.ImportPath, info.AbsFiles(), version)
			}
			opts.CacheGet = func(key string) ([]analyzer.Issue, bool) {
				return cache.Get(dir, key)
			}
			opts.CachePut = func(key string, issues []analyzer.Issue) error {
				return cache.Put(dir, key, issues)
			}
		}
	}

	start := time.Now()
	issues, runStats, runErr := analyzer.Run(patterns, analyzers, opts)
	elapsed := time.Since(start)
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "gox:", runErr)
		return 2
	}
	for _, is := range issues {
		fmt.Printf("%s:%d:%d: %s: %s\n", is.Pos.Filename, is.Pos.Line, is.Pos.Column, is.Analyzer, is.Message)
		if is.Hint != "" {
			fmt.Printf("    hint: %s\n", is.Hint)
		}
	}
	if *stats {
		fmt.Fprintf(os.Stderr, "gox: %d packages (hits=%d misses=%d) in %s\n",
			runStats.PackagesTotal, runStats.CacheHits, runStats.CacheMisses, elapsed.Round(time.Millisecond))
	}
	if len(issues) > 0 {
		fmt.Fprintf(os.Stderr, "gox: %d issue(s)\n", len(issues))
		return 1
	}
	return 0
}

func runGo(sub string, args []string) int {
	cmd := exec.Command("go", append([]string{sub}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if runErr := cmd.Run(); runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "gox:", runErr)
		return 2
	}
	return 0
}
