# contextcheck

`context.Context` must propagate from a function's parameter, not be re-created
inside the body.

## What it catches

Creating `context.Background()` or `context.TODO()` inside a function that already
receives a `context.Context` breaks the cancellation/deadline chain. Requests hang
past their deadlines; cancellations never propagate to downstream calls. The bug
compiles cleanly and only surfaces under load.

## Bad

```go
func handle(ctx context.Context, req Req) {
    db.Query(context.Background(), req.Key) // ignores deadline from ctx
}
```

## Good

```go
func handle(ctx context.Context, req Req) {
    db.Query(ctx, req.Key)
}
```

## Opt-out

`// safe-ignore: <reason>` on the same line.

```go
go cleanup(context.Background()) // safe-ignore: detached cleanup must outlive request
```

## Limitations

- Only checks functions whose signature includes a `context.Context` parameter.
  Top-level `main`/`init`/test helpers are not flagged.
