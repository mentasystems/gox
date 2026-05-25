# noglobals

Forbids mutable package-level `var` declarations without justification.

## What it catches

Mutable package-level state is the cheapest source of test flakiness, race
conditions, and order-dependent test failures. Every `var x = ...` at package
scope is a hidden parameter to every function in the package. LLMs reach for it
constantly because it's easier than threading state through call signatures.

## Bad

```go
var cache = map[string]Result{}
var defaultClient = &http.Client{}
```

## Good

```go
type Service struct {
    cache  map[string]Result
    client *http.Client
}

const MaxRetries = 3
```

## Opt-out

`// global-ok: <reason>` on the same line.

```go
var registered = map[string]*Analyzer{} // global-ok: registration is a process-level singleton by design
```

## Limitations

- `const` declarations are not flagged.
- `var` of immutable types declared and never reassigned would still be flagged
  — use `const` when possible, annotate when not.
