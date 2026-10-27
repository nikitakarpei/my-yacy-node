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
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/metrichistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/visitlinksecret"
)

const (
	brokerDirectoryName        = "nats"
	metricHistoryDirectoryName = "metric-history"
)

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
	history, err := metrichistory.Open(
		ctx,
		cfg.directoryOf(metricHistoryDirectoryName),
		cfg.MetricHistoryRetention,
		wallclock.Clock{},
	)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	defer closeHistory(ctx, history)
	overlay, err := yggdrasilOverlayFor(ctx, cfg)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	defer overlay.stop()
	egress, err := egressRouteFor(ctx, cfg, overlay.egressOverlays())
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	addresses := peerAddressesFor(broker.ClientURL(), egress)
	crawler, err := crawlerFor(ctx, cfg, addresses, getenv)
	if err != nil {
		return errors.Join(err, closeEvery([]io.Closer{ops, egress}))
	}

	services, err := servicesFor(ctx, cfg, addresses, overlay, crawler, visitLinkSecret, getenv)
	if err != nil {
		return errors.Join(err, closeEvery([]io.Closer{ops, egress}))
	}
	logStarted(ctx, addresses, crawler.apiURL, ops)
	processRegistry := opsmetrics.ProcessRegistry()
	opsMux := opsMuxFor(processRegistry, services, history)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, egress, overlay, services, func(ctx context.Context) error {
			return history.Run(ctx, metricSourcesOf(processRegistry, services))
		})
	})
}

func closeHistory(ctx context.Context, history *metrichistory.History) {
	if err := history.Close(); err != nil {
		slog.WarnContext(ctx, "metric history did not close", slog.Any("error", err))
	}
}

func closeEvery[C io.Closer](closers []C) error {
	failures := make([]error, 0, len(closers))
	for _, closer := range closers {
		failures = append(failures, closer.Close())
	}
	return errors.Join(failures...)
}

func logStarted(
	ctx context.Context,
	addresses peerAddresses,
	crawlerAPIURL string,
	ops net.Listener,
) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", addresses.natsURL),
		slog.String("egressProxyUrl", addresses.egressProxyURL.Redacted()),
		slog.String("crawlerApiUrl", crawlerAPIURL),
		slog.String("opsAddr", ops.Addr().String()),
	)
}

func opsMuxFor(
	processRegistry *prometheus.Registry,
	services []service,
	history *metrichistory.History,
) *http.ServeMux {
	mux := opsmetrics.MuxFor(processRegistry)
	for _, service := range services {
		mux.Handle(opsmetrics.PathMetrics+"/"+service.name, opsmetrics.HandlerFor(service.registry))
	}
	dashboardpage.Register(mux, history, metrichistory.SampleInterval, serviceDashboards)
	return mux
}

func metricSourcesOf(
	processRegistry *prometheus.Registry,
	services []service,
) []metrichistory.Source {
	sources := make([]metrichistory.Source, 0, len(services))
	for _, service := range services {
		sources = append(sources, metrichistory.Source{
			Gatherer: prometheus.Gatherers{processRegistry, service.registry},
			Instance: service.name,
		})
	}
	return sources
}

func runEvery(
	ctx context.Context,
	egress egressRoute,
	overlay yggdrasilOverlay,
	services []service,
	historyRun func(context.Context) error,
) error {
	runs := append(append(egress.runs(), overlay.runs()...), historyRun)
	for _, service := range services {
		runs = append(runs, service.run)
	}
	return servergroup.Run(ctx, 0, nil, runs...)
}
