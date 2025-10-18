package unmarshaler

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/shiwano/errdef"
)

// WithStrictMode returns an Option that enables strict validation mode.
// When enabled, the unmarshaler performs strict validation on both fields and kinds:
//   - Returns ErrUnknownField if it encounters a field that is not defined in
//     the error definition or registered via WithCustomFields
//   - Returns ErrUnknownKind if it encounters an unknown kind, even when using
//     a DefaultResolver
//
// This option is useful in development and testing environments to detect
// schema inconsistencies early, ensuring strict compatibility between error
// producers and consumers. In production environments, it's recommended to omit
// this option to allow graceful handling of unknown fields and kinds from
// different versions.
func WithStrictMode() Option {
	return func(u *unmarshaler) {
		u.strictMode = true
	}
}

// WithCustomFields returns an Option that registers custom field keys
// that are not defined in the error definition but should be recognized during
// unmarshaling.
//
// This option is useful when you need to unmarshal custom fields that are not
// part of the error definition. When used with WithStrictMode,
// these custom fields will be allowed while other unknown fields will
// trigger ErrUnknownField.
func WithCustomFields(keys ...errdef.FieldKey) Option {
	return func(u *unmarshaler) {
		u.customFieldKeys = append(u.customFieldKeys, keys...)
	}
}

// WithBuiltinFields returns an Option that registers all built-in field keys
// from the errdef package to be recognized during unmarshaling.
//
// This includes: http_status, log_level, trace_id, domain, user_hint, public,
// retryable, retry_after, unreportable, exit_code, help_url.
//
// This is a convenience function that calls WithCustomFields with all
// built-in field keys. When unmarshaling errors with built-in fields, these
// fields will be properly recognized and accessible via their respective
// extractors (e.g., errdef.HTTPStatusFrom, errdef.IsPublic).
func WithBuiltinFields() Option {
	return WithCustomFields(
		errdef.HTTPStatus.Key(),
		errdef.LogLevel.Key(),
		errdef.TraceID.Key(),
		errdef.Domain.Key(),
		errdef.UserHint.Key(),
		errdef.Public.Key(),
		errdef.Retryable.Key(),
		errdef.RetryAfter.Key(),
		errdef.Unreportable.Key(),
		errdef.ExitCode.Key(),
		errdef.HelpURL.Key(),
		errdef.Details{}.Key(),
	)
}

// WithSentinelErrors returns an Option that registers custom sentinel errors
// to be recognized during unmarshaling.
//
// When unmarshaling error causes, if an error's type name and message match
// a registered sentinel error, the original error instance will be restored
// instead of creating a new unknown error wrapper. This is essential for
// preserving error identity when using errors.Is() checks.
//
// The function panics if duplicate sentinel errors (same type and message)
// are registered.
func WithSentinelErrors(errors ...error) Option {
	return func(u *unmarshaler) {
		if u.sentinelErrors == nil {
			u.sentinelErrors = make(map[sentinelKey]error)
		}
		for _, err := range errors {
			key := sentinelKey{
				typeName: fmt.Sprintf("%T", err),
				message:  err.Error(),
			}

			if _, exists := u.sentinelErrors[key]; exists {
				panic("duplicate sentinel error: " + key.typeName + " - " + key.message)
			}
			u.sentinelErrors[key] = err
		}
	}
}

// WithStandardSentinelErrors returns an Option that registers commonly used
// sentinel errors from the Go standard library (such as context.Canceled,
// io.EOF, os.ErrNotExist, etc.).
//
// This is a convenience function that calls WithSentinelErrors with a predefined
// set of standard errors. When unmarshaling, these sentinel errors will be
// restored to their original error instances instead of being wrapped as
// unknown errors.
func WithStandardSentinelErrors() Option {
	return WithSentinelErrors(
		context.Canceled,
		context.DeadlineExceeded,
		csv.ErrBareQuote,
		csv.ErrFieldCount,
		csv.ErrQuote,
		exec.ErrNotFound,
		fs.ErrClosed,     // Same as os.ErrClosed
		fs.ErrExist,      // Same as os.ErrExist
		fs.ErrInvalid,    // Same as os.ErrInvalid
		fs.ErrNotExist,   // Same as os.ErrNotExist
		fs.ErrPermission, // Same as os.ErrPermission
		http.ErrBodyNotAllowed,
		http.ErrBodyReadAfterClose,
		http.ErrContentLength,
		http.ErrHandlerTimeout,
		http.ErrHijacked,
		http.ErrLineTooLong,
		http.ErrMissingBoundary,
		http.ErrMissingFile,
		http.ErrNoCookie,
		http.ErrNoLocation,
		http.ErrNotMultipart,
		http.ErrNotSupported,
		http.ErrSchemeMismatch,
		http.ErrServerClosed,
		http.ErrSkipAltProtocol,
		http.ErrUseLastResponse,
		io.EOF,
		io.ErrClosedPipe,
		io.ErrNoProgress,
		io.ErrShortBuffer,
		io.ErrShortWrite,
		io.ErrUnexpectedEOF,
		net.ErrClosed,
		net.ErrWriteToConnected,
		os.ErrDeadlineExceeded,
		os.ErrNoDeadline,
		os.ErrProcessDone,
		sql.ErrConnDone,
		sql.ErrNoRows,
		sql.ErrTxDone,
		zip.ErrAlgorithm,
		zip.ErrChecksum,
		zip.ErrFormat,
	)
}
