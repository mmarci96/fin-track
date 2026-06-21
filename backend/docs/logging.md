# Logging & Error Handling

This project uses **slog over zap** for logging and **cockroachdb/errors** for
error wrapping with origin stack traces. The goal: every failure is logged
exactly once, with full context and a stack trace, while clients only ever see a
safe message.

## The three rules

1. **Wrap at the origin.** Whenever you return an error from a dependency
   (database, HTTP call, file I/O, OCR), wrap it with context:
   ```go
   if err != nil {
       return errors.Wrapf(err, "insert receipt merchant_id=%d", id)
   }
   ```
   `cockroachdb/errors` captures the stack at the first wrap. Use `errors.Wrap` /
   `errors.Wrapf`; use `errors.Newf` to create a fresh error.

2. **Classify at the edge.** When an error is about to leave a layer that knows
   its HTTP meaning, wrap it in an `apperr`:
   ```go
   httpx.Respond(c, apperr.Internal("could not save receipt", err))
   httpx.Respond(c, apperr.BadRequest("image required", err))
   ```
   Unclassified errors automatically become a generic `500 INTERNAL`, so
   internals never leak.

3. **Log once, at the boundary.** Lower layers **return** errors; they do not log
   them. The HTTP boundary (`httpx.Respond` / `httpx.Recovery`) and `main` are
   the only places that log errors. This avoids duplicate, noisy log lines.

## Request-scoped logging

Every request gets a logger carrying `request_id`, `method`, and `path`. An
inbound `X-Request-ID` header is reused (and echoed back); otherwise one is
generated. Retrieve it from the request context and add fields as you learn
them:

```go
log := logger.FromContext(c.Request.Context())
log = log.With("merchant", result.Merchant.Name)
log.Info("receipt mapped", "products", len(result.Products))
```

Never use `fmt.Println` / the standard `log` package — they bypass the structured
logger and the request id.

## Sentinel errors

Known domain conditions are sentinels compared with `errors.Is`:

```go
if errors.Is(err, apperr.ErrNoMerchantMatch) {
    // soft outcome, not a 500
}
```

Add new sentinels in `internal/apperr`. Do **not** match on `err.Error()` strings.

## Lifecycle

- `logger.Init` is called once in `main`; its error is fatal.
- `defer logger.Sync()` flushes buffered logs on exit — important so logs are
  not lost on crash.
- The server runs with graceful shutdown (SIGINT/SIGTERM) so in-flight requests
  drain and logs flush.

## Configuration

| Env var       | Default     | Effect                                        |
|---------------|-------------|-----------------------------------------------|
| `LOG_LEVEL`   | `DEBUG`     | `debug` / `info` / `warn` / `error`           |
| `RUNTIME_ENV` | `dev`       | `dev`/`development`/`local` → console encoder; anything else → JSON |

## Adding a new endpoint — checklist

- [ ] Wrap dependency errors at the origin with `errors.Wrap`/`Wrapf`.
- [ ] Return errors up the stack; don't log them in services/repositories.
- [ ] Classify with `apperr.*` and respond via `httpx.Respond(c, err)`.
- [ ] Use `logger.FromContext(c.Request.Context())` for any informational logs.
- [ ] Use sentinels + `errors.Is` for expected, non-fatal conditions.

## Deferred (wire in later, no rework needed)

- **Rotating files**: add a `lumberjack`-backed zap core in `logger.Init`.
- **Sentry**: `cockroachdb/errors` is Sentry-aware; add a hook in `httpx.Respond`
  for `5xx` errors to report with the captured stack.
