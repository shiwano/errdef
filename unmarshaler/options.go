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

func WithAdditionalFields(keys ...errdef.FieldKey) Option {
	return func(u *Unmarshaler) {
		u.additionalFieldKeys = append(u.additionalFieldKeys, keys...)
	}
}

func WithStandardSentinelErrors() Option {
	return WithSentinelErrors(standardSentinelErrors()...)
}

func WithSentinelErrors(errors ...error) Option {
	return func(u *Unmarshaler) {
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

func standardSentinelErrors() []error {
	return []error{
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
	}
}
