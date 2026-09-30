package idlecancelling

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtripcancel"
)

type idleCancellingBody struct {
	body          io.ReadCloser
	clock         Clock
	idleTimeout   time.Duration
	readingCtx    context.Context
	cancelReading context.CancelCauseFunc
}

func (b idleCancellingBody) Read(chunk []byte) (int, error) {
	stopIdleTimeout := b.clock.After(b.idleTimeout, func() {
		b.cancelReading(roundtripcancel.ErrIdleTimeoutPassed)
	})
	defer stopIdleTimeout()
	bytesRead, err := b.body.Read(chunk)
	if err != nil && errors.Is(context.Cause(b.readingCtx), roundtripcancel.ErrIdleTimeoutPassed) {
		return bytesRead, roundtripcancel.ErrIdleTimeoutPassed
	}
	//nolint:wrapcheck // io.EOF reaches the caller of a reader unwrapped
	return bytesRead, err
}

func (b idleCancellingBody) Close() error {
	b.cancelReading(nil)
	return b.body.Close() //nolint:wrapcheck // the caller of Close gets the cause of the body
}
