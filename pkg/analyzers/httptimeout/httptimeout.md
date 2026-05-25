# httptimeout

HTTP clients and shortcut calls must set an explicit `Timeout`.

## What it catches

`http.Get`, `http.Post`, `http.Head`, `http.PostForm`, and any call on
`http.DefaultClient` go through a zero-Timeout client — a hanging server will
block the goroutine forever. Same trap for a freshly constructed `&http.Client{}`
whose `Timeout` field is left at the zero value (or explicitly set to 0). This
is the single most common cause of "the request never returned" production
incidents in LLM-written Go.

## Bad

```go
resp, _ := http.Get(url)

client := &http.Client{Transport: t}
resp, _ := client.Get(url) // client has no Timeout
```

## Good

```go
client := &http.Client{Timeout: 30 * time.Second}
resp, err := client.Get(url)

// or
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := client.Do(req)
```

## Opt-out

`// timeout-ok: <reason>` on the same line as the flagged call or the opening
brace of the literal.

```go
resp, _ := http.Get(url) // timeout-ok: probe via mock RoundTripper in tests

client := &http.Client{ // timeout-ok: streaming server, deadline managed per-request
    Transport: t,
}
```

## Limitations

- Does not track `*http.Client` values across function boundaries. A
  `var c http.Client` followed by `c.Get(url)` is not flagged — the literal
  construction is the gate.
- Does not infer deadlines on requests built via `NewRequestWithContext` — the
  presence of a configured Timeout (or trust in a non-constant Timeout
  expression) is the contract.
