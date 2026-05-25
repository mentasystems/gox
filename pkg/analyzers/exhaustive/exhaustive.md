# exhaustive

Requires switch exhaustiveness over iota enums and sealed interfaces.

## What it catches

When you add a new enum value or interface implementation, every `switch` over
the old set silently falls through. The change compiles, tests pass, and
production takes an unhandled path. Forcing exhaustiveness turns that into a
compile-time fail that breaks the build the moment the new variant lands.

## Bad

```go
type Color int
const (
    Red Color = iota
    Green
    Blue
)
switch c {
case Red:
case Green:
} // Blue silently unhandled
```

## Good

```go
switch c {
case Red:
case Green:
case Blue:
}
// or
switch c {
case Red:
case Green:
default: // exhaustive-ok: any future variant routes here on purpose
}
```

## Opt-out

`// exhaustive-ok: <reason>` on the `default:` line marks intentional fallthrough.

## Limitations

- Detects iota-defined enums in the same package, and interfaces with private
  marker methods (the standard "sealed interface" pattern).
