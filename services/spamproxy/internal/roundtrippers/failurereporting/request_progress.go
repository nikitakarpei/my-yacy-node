package failurereporting

import (
	"context"
	"net/http/httptrace"
	"sync/atomic"
)

type Step string

const (
	ConnectFailed      Step = "connect_failed"
	RequestWriteFailed Step = "request_write_failed"
	HeadersReadFailed  Step = "headers_read_failed"
)

type requestProgress struct {
	wasConnected      atomic.Bool
	wasRequestWritten atomic.Bool
}

func (p *requestProgress) tracedContextFrom(ctx context.Context) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { p.wasConnected.Store(true) },
		WroteRequest: func(written httptrace.WroteRequestInfo) {
			p.wasRequestWritten.Store(written.Err == nil)
		},
	})
}

func (p *requestProgress) failedStep() Step {
	switch {
	case !p.wasConnected.Load():
		return ConnectFailed
	case !p.wasRequestWritten.Load():
		return RequestWriteFailed
	default:
		return HeadersReadFailed
	}
}
