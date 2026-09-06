// Package processenvironmentleaseclient holds the consumer end of one process
// environment lease. It connects to the lease socket, reads the single grant
// frame, keeps the connection open for as long as the grant is valid, and
// reports the revocation that the producer signals by closing the connection.
package processenvironmentleaseclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
)

const (
	grantReadTimeout  = 5 * time.Second
	firstRetryDelay   = 100 * time.Millisecond
	longestRetryDelay = 2 * time.Second
	socketNetwork     = "unix"
)

var ErrRevoked = errors.New("process environment lease revoked")

type Lease struct {
	connection net.Conn
	grant      processenvironmentlease.Grant
	revocation chan error
}

func Acquire(ctx context.Context, socketPath string) (*Lease, error) {
	delay := firstRetryDelay
	for {
		lease, err := acquireOnce(ctx, socketPath)
		if err == nil {
			return lease, nil
		}

		slog.WarnContext(ctx, "process environment lease not acquired",
			slog.String("socketPath", socketPath),
			slog.Duration("retryDelay", delay),
			slog.Any("error", err),
		)
		if err := sleep(ctx, delay); err != nil {
			return nil, err
		}
		delay = min(delay*2, longestRetryDelay)
	}
}

func acquireOnce(ctx context.Context, socketPath string) (*Lease, error) {
	dialer := net.Dialer{}

	connection, err := dialer.DialContext(ctx, socketNetwork, socketPath)
	if err != nil {
		return nil, fmt.Errorf("dial process environment lease socket: %w", err)
	}

	reader := bufio.NewReaderSize(connection, processenvironmentlease.MaximumFrameByte+1)

	grant, err := grantFrom(ctx, connection, reader)
	if err != nil {
		closeConnection(ctx, connection)

		return nil, err
	}

	lease := &Lease{
		connection: connection,
		grant:      grant,
		revocation: make(chan error, 1),
	}
	go lease.watch(context.WithoutCancel(ctx), reader)

	return lease, nil
}

func grantFrom(
	ctx context.Context,
	connection net.Conn,
	reader *bufio.Reader,
) (processenvironmentlease.Grant, error) {
	if err := connection.SetReadDeadline(grantDeadlineFrom(ctx)); err != nil {
		return processenvironmentlease.Grant{}, fmt.Errorf("await the grant frame: %w", err)
	}

	frame, err := reader.ReadSlice('\n')
	if err != nil {
		return processenvironmentlease.Grant{}, fmt.Errorf("read grant frame: %w", err)
	}

	grant, err := processenvironmentlease.GrantFrom(frame)
	if err != nil {
		return processenvironmentlease.Grant{}, fmt.Errorf("read grant frame: %w", err)
	}

	if err := connection.SetReadDeadline(time.Time{}); err != nil {
		return processenvironmentlease.Grant{}, fmt.Errorf("hold the grant: %w", err)
	}

	return grant, nil
}

func grantDeadlineFrom(ctx context.Context) time.Time {
	deadline := time.Now().Add(grantReadTimeout)

	if acquisitionDeadline, ok := ctx.Deadline(); ok && acquisitionDeadline.Before(deadline) {
		return acquisitionDeadline
	}

	return deadline
}

func (lease *Lease) watch(ctx context.Context, reader *bufio.Reader) {
	trailing, err := reader.ReadByte()
	switch {
	case errors.Is(err, io.EOF):
		lease.revocation <- ErrRevoked
	case err != nil:
		lease.revocation <- fmt.Errorf("%w: read after the grant: %w", ErrRevoked, err)
	default:
		lease.revocation <- fmt.Errorf(
			"%w: %w: byte %q after the grant frame",
			ErrRevoked, processenvironmentlease.ErrBadGrant, trailing,
		)
	}

	slog.DebugContext(ctx, "process environment lease watch ended")
}

func (lease *Lease) Grant() processenvironmentlease.Grant {
	return lease.grant
}

func (lease *Lease) Revocation() <-chan error {
	return lease.revocation
}

func (lease *Lease) Close() error {
	if err := lease.connection.Close(); err != nil {
		return fmt.Errorf("close process environment lease: %w", err)
	}

	return nil
}

func closeConnection(ctx context.Context, connection net.Conn) {
	if err := connection.Close(); err != nil {
		slog.WarnContext(ctx, "process environment lease connection not closed",
			slog.Any("error", err),
		)
	}
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("acquire process environment lease: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
