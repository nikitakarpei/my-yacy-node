package yacypeerassembly

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/visitlinksecret"
)

const brokerDirectoryName = "nats"

func RunPeer(
	ctx context.Context,
	cfg PeerConfig,
	ops net.Listener,
	getenv func(string) string,
) error {
	visitLinkSecret, err := visitlinksecret.Settle(cfg.directoryOf(visitcrawlService))
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	broker, err := natsbroker.Start(cfg.directoryOf(brokerDirectoryName))
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	defer broker.Shutdown()
	egress, err := egressRouteFor(ctx, cfg)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	addresses, err := peerAddressesFor(ctx, broker.ClientURL(), egress)
	if err != nil {
		return errors.Join(err, closeEvery([]io.Closer{ops, egress}))
	}

	services, err := servicesFor(ctx, cfg, addresses, visitLinkSecret, getenv)
	if err != nil {
		return errors.Join(err, closeEvery([]io.Closer{ops, egress, addresses}))
	}
	logStarted(ctx, addresses, ops)
	opsMux := opsMuxFor(opsmetrics.ProcessRegistry(), services)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, egress, services)
	})
}

func closeEvery[C io.Closer](closers []C) error {
	failures := make([]error, 0, len(closers))
	for _, closer := range closers {
		failures = append(failures, closer.Close())
	}
	return errors.Join(failures...)
}

func logStarted(ctx context.Context, addresses peerAddresses, ops net.Listener) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", addresses.natsURL),
		slog.String("egressProxyUrl", addresses.egressProxyURL.Redacted()),
		slog.String("crawlerApiAddr", addresses.crawlerAPIAddress()),
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

func runEvery(ctx context.Context, egress egressRoute, services []service) error {
	runs := egress.runs()
	for _, service := range services {
		runs = append(runs, service.run)
	}
	return servergroup.Run(ctx, 0, nil, runs...)
}
