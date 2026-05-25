# shadow

Reports variables that shadow names from an outer scope.

## What it catches

`if err := f(); err != nil { ... }` declares a *new* `err` that masks the outer
one. The outer `err` keeps its previous value (often `nil`), and the caller
proceeds as if everything is fine. This is one of the highest-frequency LLM bug
patterns in Go.

## Bad

```go
func handle() error {
    var err error
    if err := step1(); err != nil { // shadows outer err
        return err
    }
    if err := step2(); err != nil { // shadows again
        return err
    }
    return err // always nil — outer err was never assigned
}
```

## Good

```go
func handle() error {
    if err := step1(); err != nil { return err }
    if err := step2(); err != nil { return err }
    return nil
}
```

## Opt-out

`// safe-ignore: <reason>` on the declaration line.

```go
if ok := check(); ok { ... } // ok is exempted by name
```

## Limitations

- The name `ok` is exempted everywhere — it's the universal comma-ok pattern
  and shadow warnings on it are pure noise.
