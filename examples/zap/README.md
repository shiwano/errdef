# Zap Integration Example

This example demonstrates how to log `errdef` errors with Zap's structured logging, including automatic field extraction and stack trace integration.

## Basic Example

```go
package main

import (
  "context"

  "github.com/shiwano/errdef"
  zaphelper "github.com/shiwano/errdef/examples/zap"
  "go.uber.org/zap"
)

var (
  ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))
  UserID, _   = errdef.DefineField[string]("user_id")
)

func main() {
  ctx := context.Background()

  config := zap.NewProductionConfig()
  logger, _ := config.Build()
  defer logger.Sync()

  err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

  // Error nests error information under "error" key
  logger.Info("operation failed", zaphelper.Error(err))
}
```

## ErrorInline

Use `ErrorInline()` to expand all error information at the top level instead of nesting under "error" key:

```go
err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

// ErrorInline expands all error information at the top level
logger.Info("operation failed", zaphelper.ErrorInline(err))
```

## Sensitive Data

Use `errdef.Redacted[T]` to ensure sensitive data is never exposed in logs:

```go
var Email, _ = errdef.DefineField[errdef.Redacted[string]]("email")

err := ErrNotFound.With(ctx,
  UserID("user123"),
  Email(errdef.Redact("user@example.com")),
).New("user not found")

logger.Info("operation failed", zaphelper.Error(err))
// Email will appear as "[REDACTED]" in logs
```

## Running the Example

```bash
go test -v
```
