// Package servicehost runs one service of the stack and its ops endpoint in an app
// process until the app stops it.
package servicehost

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

type Service func(ctx context.Context, registry *prometheus.Registry) error

type Host struct {
	opsAddr  string
	service  Service
	stopped  chan struct{}
	stopOnce sync.Once
}

func HostFor(opsAddr string, service Service) *Host {
	return &Host{opsAddr: opsAddr, service: service, stopped: make(chan struct{})}
}

func (h *Host) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-h.stopped:
			cancel()
		case <-ctx.Done():
		}
	}()
	return h.serve(ctx)
}

func (h *Host) serve(ctx context.Context) error {
	ops, err := servergroup.Listen(ctx, h.opsAddr)
	if err != nil {
		return err
	}
	registry := opsmetrics.ProcessRegistry()
	return opsmetrics.ServeAlongside(
		ctx,
		ops,
		opsmetrics.MuxFor(registry),
		func(ctx context.Context) error {
			return h.service(ctx, registry)
		},
	)
}

func (h *Host) Stop() {
	h.stopOnce.Do(func() { close(h.stopped) })
}
