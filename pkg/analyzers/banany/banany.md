# banany

Forbids `any` / `interface{}` in declarations without an inline justification.

## What it catches

LLMs frequently reach for `any` when they don't know the exact type, then write code
that runtime-asserts and panics in production. Forcing a written reason at the
declaration site converts a silent any-the-world escape into an explicit, reviewed
choice.

## Bad

```go
func process(in any) any { ... }   // both flagged
var cache map[string]any            // flagged
```

## Good

```go
func process(in []byte) Result { ... }
var cache map[string]Entry
```

## Opt-out

`// any-ok: <reason>` on the same line.

```go
var bus chan any // any-ok: bus carries heterogeneous event types by design
```

## Limitations

- Empty reason is rejected.
- Function-typed parameters such as `func(any)` are also flagged.
