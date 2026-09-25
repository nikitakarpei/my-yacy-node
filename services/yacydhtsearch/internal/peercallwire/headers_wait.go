package peercallwire

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

type headersWait struct {
	ctx         context.Context
	endWait     context.CancelFunc
	stopTimer   func()
	headersLate atomic.Bool
}

func headersWaitWithin(
	ctx context.Context,
	headersTimeout time.Duration,
	clock Clock,
) *headersWait {
	ctx, endWait := context.WithCancel(ctx)
	wait := &headersWait{ctx: ctx, endWait: endWait, stopTimer: func() {}}
	if headersTimeout > 0 {
		wait.stopTimer = clock.After(headersTimeout, wait.expire)
	}

	return wait
}

func (wait *headersWait) expire() {
	wait.headersLate.Store(true)
	wait.endWait()
}

func (wait *headersWait) responseTo(
	client *http.Client,
	request *http.Request,
) (*http.Response, error) {
	response, err := client.Do(request.WithContext(wait.ctx))
	wait.stopTimer()

	return response, err //nolint:wrapcheck // the wire reads the failure of the call as it came
}

func (wait *headersWait) end() {
	wait.endWait()
}
