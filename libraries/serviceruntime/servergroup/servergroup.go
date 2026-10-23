// Package servergroup binds the addresses a service serves on, and runs a set
// of HTTP servers on their listeners alongside background workers until the
// context is cancelled or any server or worker returns, then shuts every
// server down gracefully. A request carries the values of the run context, but
// not its cancellation.
package servergroup

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

type NamedServer struct {
	Name     string
	Listener net.Listener
	Server   *http.Server
}

// Run serves every server and runs every worker until ctx is cancelled or any
// server or worker returns. The first return then triggers a graceful shutdown
// of every server within shutdownTimeout. Run returns the first non-nil error;
// http.ErrServerClosed from a shutdown server is not an error.
func Run(
	ctx context.Context,
	shutdownTimeout time.Duration,
	servers []NamedServer,
	workers ...func(context.Context) error,
) error {
	group, groupCtx := errgroup.WithContext(ctx)
	runCtx, cancel := context.WithCancel(groupCtx)
	defer cancel()

	for _, s := range servers {
		if s.Server.BaseContext == nil {
			s.Server.BaseContext = func(net.Listener) context.Context {
				return context.WithoutCancel(ctx)
			}
		}
		group.Go(func() error {
			defer cancel()
			if err := s.Server.Serve(s.Listener); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("serve %s: %w", s.Name, err)
			}
			return nil
		})
	}

	for _, worker := range workers {
		group.Go(func() error {
			defer cancel()
			return worker(runCtx)
		})
	}

	group.Go(func() error {
		<-runCtx.Done()
		return shutdown(servers, shutdownTimeout)
	})

	return group.Wait()
}

func shutdown(servers []NamedServer, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var failures error
	for _, s := range servers {
		if err := s.Server.Shutdown(shutdownCtx); err != nil {
			failures = errors.Join(failures, fmt.Errorf("shutdown %s: %w", s.Name, err))
		}
	}
	return failures
}
