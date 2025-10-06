package unmarshaler

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"

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
			u.sentinelErrors[key] = err
		}
	}
}

func standardSentinelErrors() []error {
	return []error{
		io.EOF,
		io.ErrUnexpectedEOF,
		io.ErrShortWrite,
		io.ErrShortBuffer,
		io.ErrNoProgress,
		io.ErrClosedPipe,
		context.Canceled,
		context.DeadlineExceeded,
		os.ErrInvalid,
		os.ErrPermission,
		os.ErrExist,
		os.ErrNotExist,
		os.ErrClosed,
		os.ErrNoDeadline,
		os.ErrDeadlineExceeded,
		os.ErrProcessDone,
		fs.ErrInvalid,
		fs.ErrPermission,
		fs.ErrExist,
		fs.ErrNotExist,
		fs.ErrClosed,
		net.ErrClosed,
		net.ErrWriteToConnected,
		http.ErrNotSupported,
		http.ErrMissingBoundary,
		http.ErrNotMultipart,
		http.ErrBodyNotAllowed,
		http.ErrHijacked,
		http.ErrContentLength,
		http.ErrBodyReadAfterClose,
		http.ErrHandlerTimeout,
		http.ErrLineTooLong,
		http.ErrMissingFile,
		http.ErrNoCookie,
		http.ErrNoLocation,
		http.ErrSchemeMismatch,
		http.ErrServerClosed,
		http.ErrSkipAltProtocol,
		http.ErrUseLastResponse,
		sql.ErrConnDone,
		sql.ErrTxDone,
		sql.ErrNoRows,
	}
}

func makeSentinelKey(err error) sentinelKey {
	return sentinelKey{
		typeName: fmt.Sprintf("%T", err),
		message:  err.Error(),
	}
}
