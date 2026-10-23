package yacypeerassembly

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const brokerDirectoryName = "nats"

func RunPeer(
	ctx context.Context,
	cfg PeerConfig,
	ops net.Listener,
	getenv func(string) string,
) error {
	broker, err := natsbroker.Start(cfg.directoryOf(brokerDirectoryName))
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	defer broker.Shutdown()

	services, err := servicesFor(ctx, cfg, broker.ClientURL(), getenv)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	logStarted(ctx, broker, ops)
	opsMux := opsMuxFor(opsmetrics.ProcessRegistry(), services)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, services)
	})
}

func logStarted(ctx context.Context, broker *natsbroker.Broker, ops net.Listener) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", broker.ClientURL()),
		slog.String("opsAddr", ops.Addr().String()),
	)
}

func opsMuxFor(processRegistry *prometheus.Registry, services []service) *http.ServeMux {
	mux := opsmetrics.MuxFor(processRegistry)
	for _, service := range services {
		mux.Handle(opsmetrics.PathMetrics+"/"+service.name, opsmetrics.HandlerFor(service.registry))
	}
	return mux
}

func runEvery(ctx context.Context, services []service) error {
	runs := make([]func(context.Context) error, 0, len(services))
	for _, service := range services {
		runs = append(runs, service.run)
	}
	return servergroup.Run(ctx, 0, nil, runs...)
}
