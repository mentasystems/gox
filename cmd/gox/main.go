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
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mentasystems/gox/pkg/analyzer"
	"github.com/mentasystems/gox/pkg/baseline"
	"github.com/mentasystems/gox/pkg/cache"
	"github.com/mentasystems/gox/pkg/loader"

	// register all analyzers
	_ "github.com/mentasystems/gox/pkg/analyzers/banany"
	_ "github.com/mentasystems/gox/pkg/analyzers/bodyclose"
	_ "github.com/mentasystems/gox/pkg/analyzers/contextcheck"
	_ "github.com/mentasystems/gox/pkg/analyzers/errcheck"
	_ "github.com/mentasystems/gox/pkg/analyzers/errorlint"
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
	case "install":
		os.Exit(runInstall(args))
	case "baseline":
		os.Exit(runBaseline(args))
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
  gox baseline              capture current issues into .gox-baseline.json
                            at the module root; check filters these out
  gox install claude        install PostToolUse hook into ~/.claude/settings.json

check flags:
  --no-cache                disable the incremental cache
  --no-baseline             ignore .gox-baseline.json; report all issues
  --stats                   print cache hit/miss counts and elapsed time
  --max-issues N            cap printed issues (default 100, 0 = unlimited;
                            env: GOX_MAX_ISSUES)
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
	noBaseline := fs.Bool("no-baseline", false, "ignore .gox-baseline.json; report all issues")
	stats := fs.Bool("stats", false, "print cache hit/miss counts and elapsed time")
	maxIssues := fs.Int("max-issues", defaultMaxIssues(), "cap printed issues; 0 = unlimited (env: GOX_MAX_ISSUES)")
	if parseErr := fs.Parse(args); parseErr != nil {
		return 2
	}
	patterns := fs.Args()

	issues, runStats, elapsed, runErr := runAnalyzers(patterns, *noCache)
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "gox:", runErr)
		return 2
	}

	// Baseline filtering (only for `check`, not `baseline` capture).
	totalBeforeFilter := len(issues)
	baselinedCount := 0
	if !*noBaseline {
		issues, baselinedCount = applyBaseline(issues)
	}

	// Cap the printed issues so a package with hundreds of findings does not
	// flood a consumer's context (the Claude Stop hook pipes this straight
	// back to the model). Issues are already sorted by file:line, so the
	// truncation is deterministic. 0 = unlimited.
	shown := issues
	hidden := 0
	if *maxIssues > 0 && len(issues) > *maxIssues {
		shown = issues[:*maxIssues]
		hidden = len(issues) - *maxIssues
	}
	for _, is := range shown {
		fmt.Printf("%s:%d:%d: %s: %s\n", is.Pos.Filename, is.Pos.Line, is.Pos.Column, is.Analyzer, is.Message)
		if is.Hint != "" {
			fmt.Printf("    hint: %s\n", is.Hint)
		}
	}
	if hidden > 0 {
		fmt.Printf("... %d more issue(s) hidden (showing %d of %d; raise with --max-issues=N or GOX_MAX_ISSUES, 0 = all)\n",
			hidden, len(shown), len(issues))
	}
	if *stats {
		fmt.Fprintf(os.Stderr, "gox: %d packages (hits=%d misses=%d) in %s\n",
			runStats.PackagesTotal, runStats.CacheHits, runStats.CacheMisses, elapsed.Round(time.Millisecond))
		if baselinedCount > 0 {
			fmt.Fprintf(os.Stderr, "gox: %d baselined / %d total\n", baselinedCount, totalBeforeFilter)
		}
	}
	if len(issues) > 0 {
		fmt.Fprintf(os.Stderr, "gox: %d issue(s)\n", len(issues))
		return 1
	}
	return 0
}

// defaultMaxIssuesValue is the cap applied to printed issues when neither
// --max-issues nor GOX_MAX_ISSUES is set. Each issue prints 1–2 lines, so
// 100 keeps the output comfortably small for context-limited consumers.
const defaultMaxIssuesValue = 100

// defaultMaxIssues resolves the default for --max-issues: GOX_MAX_ISSUES if
// it holds a valid non-negative integer, otherwise defaultMaxIssuesValue.
func defaultMaxIssues() int {
	if v := os.Getenv("GOX_MAX_ISSUES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return defaultMaxIssuesValue
}

// runAnalyzers wires the cache and runs the analyzers — shared by `check`
// and `baseline`.
func runAnalyzers(patterns []string, noCache bool) ([]analyzer.Issue, analyzer.Stats, time.Duration, error) {
	analyzers := analyzer.All()
	opts := analyzer.RunOptions{UseCache: !noCache}

	if opts.UseCache {
		dir, dirErr := cache.Dir()
		if dirErr != nil {
			fmt.Fprintf(os.Stderr, "gox: cache dir: %v\n", dirErr)
			opts.UseCache = false
		} else {
			version := cache.AnalyzersVersion(analyzers)
			opts.CacheKey = func(info *loader.PackageInfo) (string, error) {
				return cache.Key( /* importPath */ info.ImportPath /* files */, info.AbsFiles() /* analyzersVersion */, version)
			}
			opts.CacheGet = func(key string) ([]analyzer.Issue, bool) {
				return cache.Get( /* dir */ dir /* key */, key)
			}
			opts.CachePut = func(key string, issues []analyzer.Issue) error {
				return cache.Put( /* dir */ dir /* key */, key, issues)
			}
		}
	}

	start := time.Now()
	issues, runStats, runErr := analyzer.Run(patterns, analyzers, opts)
	return issues, runStats, time.Since(start), runErr
}

// applyBaseline loads .gox-baseline.json from the module root (if present)
// and filters out matching issues. Returns the filtered issues and the count
// removed.
func applyBaseline(issues []analyzer.Issue) ([]analyzer.Issue, int) {
	root, rootErr := baseline.ModuleRoot()
	if rootErr != nil {
		return issues, 0
	}
	bf, loadErr := baseline.Load(filepath.Join(root, baseline.Filename))
	if loadErr != nil {
		fmt.Fprintln(os.Stderr, "gox baseline:", loadErr)
		return issues, 0
	}
	if bf == nil {
		return issues, 0
	}
	filtered := bf.Filter( /* issues */ issues /* moduleRoot */, root)
	return filtered, len(issues) - len(filtered)
}

func runBaseline(args []string) int {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	noCache := fs.Bool("no-cache", false, "disable the incremental cache")
	if parseErr := fs.Parse(args); parseErr != nil {
		return 2
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	root, rootErr := baseline.ModuleRoot()
	if rootErr != nil {
		fmt.Fprintln(os.Stderr, "gox baseline:", rootErr)
		return 2
	}

	issues, _, _, runErr := runAnalyzers(patterns, *noCache)
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "gox baseline:", runErr)
		return 2
	}

	bf, buildErr := baseline.Build(issues, root, cache.AnalyzersVersion(analyzer.All()))
	if buildErr != nil {
		fmt.Fprintln(os.Stderr, "gox baseline:", buildErr)
		return 2
	}
	path := filepath.Join(root, baseline.Filename)
	if saveErr := baseline.Save(path, bf); saveErr != nil {
		fmt.Fprintln(os.Stderr, "gox baseline:", saveErr)
		return 2
	}
	fmt.Printf("captured %d issue(s) into %s\n", bf.IssueCount, path)
	fmt.Println("from now on, `gox check` will report only NEW issues.")
	fmt.Println("commit the file so the rest of the team gets the same view.")
	return 0
}

func runGo(sub string, args []string) int {
	cmd := exec.Command("go", append([]string{sub}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if runErr := cmd.Run(); runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "gox:", runErr)
		return 2
	}
	return 0
}
