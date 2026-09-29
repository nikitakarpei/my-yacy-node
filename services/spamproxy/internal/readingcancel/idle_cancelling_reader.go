package readingcancel

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrIdleTimeoutPassed = errors.New("reading idle timeout passed")

func IdleCancellingReaderFrom(
	bodyReader io.Reader,
	clock Clock,
	cancelReading context.CancelCauseFunc,
	idleTimeout time.Duration,
) io.Reader {
	return idleCancellingReader{
		bodyReader:    bodyReader,
		clock:         clock,
		cancelReading: cancelReading,
		idleTimeout:   idleTimeout,
	}
}

type idleCancellingReader struct {
	bodyReader    io.Reader
	clock         Clock
	cancelReading context.CancelCauseFunc
	idleTimeout   time.Duration
}

func (r idleCancellingReader) Read(chunk []byte) (int, error) {
	stop := r.clock.After(r.idleTimeout, func() { r.cancelReading(ErrIdleTimeoutPassed) })
	defer stop()
	//nolint:wrapcheck // io.EOF reaches the caller of a reader unwrapped
	return r.bodyReader.Read(chunk)
}
