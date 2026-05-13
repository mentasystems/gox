# gox

Strict static analyzer for Go. Zero external dependencies — every rule is
implemented from scratch on top of `go/ast`, `go/types`, and the `go list`
command.

The goal: catch the classes of bugs an LLM writing Go without supervision is
most likely to introduce. Be loud, be opinionated, fail closed.

> **Status: experimental.** Built as a tool for using Claude Code (and
> similar agents) to write Go without leaving silent bugs behind. Used
> daily, but the rule set is still evolving — expect breaking changes to
> annotation syntax until v1.0.

## Install

```sh
go install github.com/kidandcat/gox/cmd/gox@latest
```

## Use

```sh
gox check ./...    # analyze; exit 1 on any issue
gox list           # list registered analyzers
gox build [args]   # gox check && go build
gox test  [args]   # gox check && go test
```

## Rules

| Analyzer | What it catches |
|---|---|
| `errcheck` | `error` return values dropped silently |
| `shadow` | `:=` re-declaring an outer variable (except `ok`) |
| `forcetypeassert` | `x := v.(T)` without the comma-ok form |
| `namedargs` | call sites passing 2+ args of the same basic type without `/* paramName */` comments (user code only — stdlib calls exempt) |
| `exhaustive` | non-exhaustive switch on iota enums or sealed interfaces |
| `noglobals` | mutable package-level `var` declarations |
| `banany` | `any` / `interface{}` in declarations without justification |
| `bodyclose` | `*http.Response.Body` left unclosed |
| `contextcheck` | `context.Background()`/`TODO()` inside a function that already receives a `context.Context` |
| `goroutine` | `go f()` without a visible `*errgroup.Group`, `sync.WaitGroup`, or `context.CancelFunc` |

## Annotations

Every rule has a single opt-out marker. Annotations must include a reason
after the colon — empty reasons are ignored.

| Comment | Effect |
|---|---|
| `// safe-ignore: <why>` | Suppress `errcheck`, `forcetypeassert`, `bodyclose`, `contextcheck` on the same line |
| `// global-ok: <why>` | Allow a package-level `var` (`noglobals`) |
| `// any-ok: <why>` | Allow `any` / `interface{}` (`banany`) |
| `// goroutine-ok: <why>` | Allow a fire-and-forget `go` statement (`goroutine`) |
| `// exhaustive-ok: <why>` | Accept a `default:` case as covering missing variants (`exhaustive`) |

## Claude Code integration

Add a PostToolUse hook to `~/.claude/settings.json` that runs `gox check` on
every Go file edit. Claude will see the analyzer output as a tool-result
error and iterate until the file passes.

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "if [[ \"$CLAUDE_TOOL_INPUT_FILE_PATH\" == *.go ]]; then cd \"$(dirname \"$CLAUDE_TOOL_INPUT_FILE_PATH\")\" && gox check ./... 2>&1 | head -40; fi",
            "timeout": 30
          }
        ]
      }
    ]
  }
}
```

## Performance

A pure-Go implementation with no runtime overhead from external linters.
On a 442-package monorepo (~1800 .go files):

| | Time |
|---|---|
| Cold run (no cache) | ~9.4s |
| Warm run (full cache hit) | ~2.6s |

The cache is per-package, keyed by file mtime+size and the analyzer set
hash. It lives under `$XDG_CACHE_HOME/gox/v2` (or `~/.cache/gox/v2`).
Pass `--no-cache` to disable.

## Generated code

Files marked with the standard Go marker
`// Code generated <generator> DO NOT EDIT.` near the top are skipped
automatically. This covers `protoc-gen-go`, `yo`, `mockgen`, and most
common generators.

## Design notes

- **Zero external dependencies.** Everything uses Go stdlib + a shell-out to
  `go list -json`. No `golang.org/x/tools`, no third-party linter packages.
  Mantenimiento mínimo: cuando salga una nueva versión de Go, no hay nada que
  actualizar.
- **Fail closed.** Every rule defaults to error. Opt-outs require an explicit
  annotation with a written reason — the reason is the documentation.
- **Targeted at LLM-written code.** Heuristics are tuned so each rule catches
  a high-frequency LLM bug class without flooding human-readable idioms. For
  example, `shadow` exempts `ok` (the universal comma-ok name) but still
  catches `err` re-declaration, which is exactly the bug we want.
- **`namedargs` is the killer rule.** Two adjacent string/int/bool parameters
  in user-defined code force the call site to label them with inline comments.
  Stdlib calls are exempt because their conventions are memorized. The bug it
  prevents — `transfer(orderID, userID)` vs `transfer(userID, orderID)` —
  produces no compile error and no test failure, and is the single most
  common silent-bug class in unsupervised AI-written Go.

## Contributing

Issues and pull requests welcome. Two ground rules:

1. **No external dependencies.** Every rule must be implementable with the
   Go standard library plus an out-of-process `go list -json` call. Adding
   `golang.org/x/tools/go/packages`, `staticcheck`, or any third-party
   linter is out of scope.
2. **Each new rule must pass its own check.** Run `gox check ./...` on a
   fresh clone before opening a PR — gox runs against its own source as
   part of the smoke test.

## License

BSD 3-Clause. See `LICENSE`.
