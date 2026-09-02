// Package processentrypoint runs the one command a container exists to run.
// It optionally waits for a process environment lease, overlays the
// granted values on the inherited environment, starts the command, forwards
// signals to it, stops it when the lease is revoked, and returns the exit
// status the container must exit with.
package processentrypoint

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/foregroundprocess"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/processenvironmentleaseclient"
)

const (
	EnvProcessEnvironmentLeaseSocket = "PROCESS_ENVIRONMENT_LEASE_SOCKET"

	revocationGrace = 8 * time.Second
)

var ErrNoCommand = errors.New("no command to run")

func Run(
	ctx context.Context,
	command []string,
	getenv func(string) string,
	inheritedEnvironment []string,
) (int, error) {
	if len(command) == 0 {
		return 0, ErrNoCommand
	}

	lease, err := leaseOf(ctx, getenv)
	if err != nil {
		return 0, err
	}
	defer closeLease(ctx, lease)

	child, err := foregroundprocess.Start(
		ctx, command, processEnvironmentFor(lease, inheritedEnvironment),
	)
	if err != nil {
		return 0, fmt.Errorf("start the foreground process: %w", err)
	}

	supervised := supervision{child: child}

	return supervised.exitStatusAfter(ctx, revocationOf(lease)), nil
}

func leaseOf(
	ctx context.Context,
	getenv func(string) string,
) (*processenvironmentleaseclient.Lease, error) {
	socketPath := strings.TrimSpace(getenv(EnvProcessEnvironmentLeaseSocket))
	if socketPath == "" {
		return nil, nil
	}

	lease, err := processenvironmentleaseclient.Acquire(ctx, socketPath)
	if err != nil {
		return nil, fmt.Errorf("acquire the process environment lease: %w", err)
	}

	slog.DebugContext(ctx, "process environment lease acquired",
		slog.String("socketPath", socketPath),
	)

	return lease, nil
}

func processEnvironmentFor(
	lease *processenvironmentleaseclient.Lease,
	inheritedEnvironment []string,
) []string {
	if lease == nil {
		return inheritedEnvironment
	}

	return lease.Grant().LeasedProcessEnvironmentFrom(inheritedEnvironment)
}

func revocationOf(lease *processenvironmentleaseclient.Lease) <-chan error {
	if lease == nil {
		return nil
	}

	return lease.Revocation()
}

func closeLease(ctx context.Context, lease *processenvironmentleaseclient.Lease) {
	if lease == nil {
		return
	}

	if err := lease.Close(); err != nil {
		slog.WarnContext(ctx, "process environment lease not closed", slog.Any("error", err))
	}
}

type supervision struct {
	child                *foregroundprocess.Process
	terminationRequested bool
}

func (supervised *supervision) exitStatusAfter(ctx context.Context, revocation <-chan error) int {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(signals)

	for {
		select {
		case <-supervised.child.Completed():
			return supervised.child.ExitStatus()
		case err := <-revocation:
			slog.WarnContext(ctx, "process environment lease revoked", slog.Any("error", err))
			supervised.child.Terminate(ctx, revocationGrace)
		case received := <-signals:
			supervised.forward(ctx, received)
		}
	}
}

func (supervised *supervision) forward(ctx context.Context, received os.Signal) {
	terminating := received == syscall.SIGTERM || received == os.Interrupt
	if !terminating {
		supervised.child.Forward(ctx, received)

		return
	}

	if supervised.terminationRequested {
		slog.WarnContext(ctx, "second termination signal received",
			slog.String("signal", received.String()),
		)
		supervised.child.Forward(ctx, os.Kill)

		return
	}

	supervised.terminationRequested = true
	supervised.child.Forward(ctx, received)
}
