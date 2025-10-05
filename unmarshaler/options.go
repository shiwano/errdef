package unmarshaler

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
)

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
		os.ErrNotExist,
		os.ErrExist,
		os.ErrPermission,
		os.ErrClosed,
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
