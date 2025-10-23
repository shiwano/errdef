# Zerolog Integration Example

This example demonstrates how to log `errdef` errors with zerolog's structured logging, including automatic field extraction and stack trace integration.

## Basic Example

```go
package main

import (
  "context"
  "os"

  "github.com/rs/zerolog"
  "github.com/shiwano/errdef"
  zerologhelper "github.com/shiwano/errdef/examples/zerolog"
)

var (
  ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))
  UserID, _   = errdef.DefineField[string]("user_id")
)

func main() {
  ctx := context.Background()

  logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

  err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

  // Error with Object() nests error information under "error" key
  logger.Info().Object("error", zerologhelper.Error(err)).Msg("operation failed")
}
```

## EmbedObject

Use `EmbedObject()` to expand all error information at the top level instead of nesting under "error" key:

```go
err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

// EmbedObject expands all error information at the top level
logger.Info().EmbedObject(zerologhelper.Error(err)).Msg("operation failed")
```

## Sensitive Data

Use `errdef.Redacted[T]` to ensure sensitive data is never exposed in logs:

```go
var Email, _ = errdef.DefineField[errdef.Redacted[string]]("email")

err := ErrNotFound.With(ctx,
  UserID("user123"),
  Email(errdef.Redact("user@example.com")),
).New("user not found")

logger.Info().Object("error", zerologhelper.Error(err)).Msg("operation failed")
// Email will appear as "[REDACTED]" in logs
```

## Running the Example

```bash
go test -v
```
