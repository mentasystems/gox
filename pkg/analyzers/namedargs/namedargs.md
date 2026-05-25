# namedargs

Requires inline `/* paramName */` comments when consecutive args share a type.

## What it catches

`transfer(orderID, userID)` vs `transfer(userID, orderID)` — both compile, both
pass type checking, both pass any test that doesn't specifically swap them.
Wire transfer goes to the wrong account. This is the highest-cost class of
silent bug in unsupervised AI codegen.

## Bad

```go
func transfer(from, to string, amount int64) {}
transfer(orderID, userID, amount) // swap-prone, silent
```

## Good

```go
transfer(/* from */ srcID, /* to */ dstID, amount)
```

## Opt-out

`// safe-ignore: <reason>` on the same line. The annotation suppresses every
namedargs violation on that line.

```go
transfer(a, b, amount) // safe-ignore: caller validated argument order via runtime check
```

## Limitations

- Standard-library calls are exempt — their conventions are well-known and
  labeling them adds noise.
- Same-type pairs of structs and named types are flagged; built-in numeric
  types like `int` / `string` / `bool` are the primary target.
