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
	endpoints, err := localEndpointsOn(ctx, broker.ClientURL())
	if err != nil {
		return errors.Join(err, ops.Close())
	}

	services, err := servicesFor(ctx, cfg, endpoints, getenv)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	logStarted(ctx, endpoints, ops)
	opsMux := opsMuxFor(opsmetrics.ProcessRegistry(), services)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, endpoints, services)
	})
}

func logStarted(ctx context.Context, endpoints localEndpoints, ops net.Listener) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", endpoints.natsURL),
		slog.String("egressProxyUrl", endpoints.egressProxyURL()),
		slog.String("crawlerApiAddr", endpoints.crawlerAPIAddress()),
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

func runEvery(ctx context.Context, endpoints localEndpoints, services []service) error {
	runs := make([]func(context.Context) error, 0, len(services)+1)
	runs = append(runs, endpoints.runEgressProxy)
	for _, service := range services {
		runs = append(runs, service.run)
	}
	return servergroup.Run(ctx, 0, nil, runs...)
}
