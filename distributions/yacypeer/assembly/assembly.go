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
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

const (
	brokerDirectoryName = "nats"
	egressProxyName     = "egressproxy"
	loopbackAddress     = "127.0.0.1:0"
)

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
	egressProxy, err := egressproxy.Listen(ctx)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	crawlerAPI, err := servergroup.Listen(ctx, loopbackAddress)
	if err != nil {
		return errors.Join(err, ops.Close())
	}

	endpoints := localEndpoints{
		crawlerAPI:     crawlerAPI,
		natsURL:        broker.ClientURL(),
		egressProxyURL: egressProxy.URL(),
	}
	services, err := servicesFor(ctx, cfg, endpoints, getenv)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	logStarted(ctx, endpoints, ops)
	opsMux := opsMuxFor(opsmetrics.ProcessRegistry(), services)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, egressProxy, services)
	})
}

func logStarted(ctx context.Context, endpoints localEndpoints, ops net.Listener) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", endpoints.natsURL),
		slog.String("egressProxyUrl", endpoints.egressProxyURL),
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

func runEvery(ctx context.Context, egressProxy *egressproxy.EgressProxy, services []service) error {
	runs := make([]func(context.Context) error, 0, len(services)+1)
	runs = append(runs, func(ctx context.Context) error {
		return egressProxy.Run(servicelog.ContextFor(ctx, egressProxyName))
	})
	for _, service := range services {
		runs = append(runs, service.run)
	}
	return servergroup.Run(ctx, 0, nil, runs...)
}
