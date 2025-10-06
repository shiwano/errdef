package unmarshaler

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
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
			key := makeSentinelKey(err)
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
		os.ErrClosed, // fs.ErrClosed is aliased to os.ErrClosed
		os.ErrDeadlineExceeded,
		os.ErrExist,   // fs.ErrExist is aliased to os.ErrExist
		os.ErrInvalid, // fs.ErrInvalid is aliased to os.ErrInvalid
		os.ErrNoDeadline,
		os.ErrNotExist,   // fs.ErrNotExist is aliased to os.ErrNotExist
		os.ErrPermission, // fs.ErrPermission is aliased to os.ErrPermission
		os.ErrProcessDone,
		sql.ErrConnDone,
		sql.ErrNoRows,
		sql.ErrTxDone,
		zip.ErrAlgorithm,
		zip.ErrChecksum,
		zip.ErrFormat,
	}
}

func makeSentinelKey(err error) sentinelKey {
	return sentinelKey{
		typeName: fmt.Sprintf("%T", err),
		message:  err.Error(),
	}
}
