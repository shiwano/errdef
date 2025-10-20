# Sentry Integration Example

This example demonstrates how to report `errdef` errors to Sentry with automatic field extraction and stack trace integration.

## Basic Example

```go
package main

import (
  "context"

  sentrygo "github.com/getsentry/sentry-go"
  "github.com/shiwano/errdef"
  "github.com/shiwano/errdef/examples/sentry"
)

var (
  ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404), sentry.Level(sentrygo.LevelInfo))
  UserID, _   = errdef.DefineField[string]("user_id")
)

func main() {
  // Initialize Sentry
  if err := sentrygo.Init(sentrygo.ClientOptions{
    Dsn:            "https://examplePublicKey@o0.ingest.sentry.io/0",
    Debug:          true,
    SendDefaultPII: true,
  }); err != nil {
    log.Fatalf("sentry.Init: %s", err)
  }
  defer sentrygo.Flush(2 * time.Second)

  ctx := context.Background()

  // Report error with context
  err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")
  sentry.CaptureError(ctx, err)
}
```

## Sensitive Data

Use `errdef.Redacted[T]` to ensure sensitive data is never exposed in Sentry reports:

```go
var Email, _ = errdef.DefineField[errdef.Redacted[string]]("email")

err := ErrValidation.With(ctx,
  Email(errdef.Redact("user@example.com")),
).New("invalid email")

sentry.CaptureError(ctx, err)
// Email will appear as "[REDACTED]" in Sentry
```

## Running the Example

```bash
go test -v
```
