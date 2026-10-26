// Package tcp times how long a TCP connection to an address takes to open. It closes each
// connection it opens at once, and it gives no time for an address it cannot connect to.
package tcp

import (
	"context"
	"log/slog"
	"net"
	"time"
)

const connectTimeout = 5 * time.Second

type ConnectTimer struct {
	dialer net.Dialer
}

func New() *ConnectTimer {
	return &ConnectTimer{dialer: net.Dialer{Timeout: connectTimeout}}
}

func (c *ConnectTimer) ConnectTimeOf(ctx context.Context, address string) (time.Duration, bool) {
	start := time.Now()
	connection, err := c.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		slog.DebugContext(ctx, "address not connected",
			slog.String("address", address),
			slog.Any("reason", err),
		)
		return 0, false
	}
	connectTime := time.Since(start)
	_ = connection.Close()
	return connectTime, true
}
