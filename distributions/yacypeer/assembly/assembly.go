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
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/metrichistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/visitlinksecret"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
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
	egress, err := egressproxy.Listen(ctx, cfg.EgressProxyURL)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	peer := peer{
		egressProxyURL:  egress.URL(),
		cfg:             cfg,
		natsURL:         broker.ClientURL(),
		visitLinkSecret: visitLinkSecret,
		webUIURLs:       webUIURLsFor(ops, getenv),
	}
	services, err := peer.services(ctx, getenv)
	if err != nil {
		return errors.Join(err, closeEvery([]io.Closer{ops, egress}))
	}
	logStarted(ctx, peer, ops)
	processRegistry := opsmetrics.ProcessRegistry()
	opsMux := opsMuxFor(processRegistry, services, history, peer.webUIURLs)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, services,
			func(ctx context.Context) error {
				return egress.Run(servicelog.ContextFor(ctx, egressproxy.Name))
			},
			func(ctx context.Context) error {
				return history.Run(ctx, metricSourcesOf(processRegistry, services))
			},
		)
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
	peer peer,
	ops net.Listener,
) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", peer.natsURL),
		slog.String("egressProxyUrl", peer.egressProxyURL.Redacted()),
		slog.String("opsAddr", ops.Addr().String()),
	)
}

func opsMuxFor(
	processRegistry *prometheus.Registry,
	services []service,
	history *metrichistory.History,
	webUIURLs webuitabs.URLs,
) *http.ServeMux {
	mux := opsmetrics.MuxFor(processRegistry)
	for _, service := range services {
		mux.Handle(opsmetrics.PathMetrics+"/"+service.name, opsmetrics.HandlerFor(service.registry))
	}
	dashboardpage.Register(
		mux,
		history,
		metrichistory.SampleInterval,
		serviceDashboards,
		webUIURLs.TabBarFor(webuitabs.Metrics),
	)
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
	services []service,
	workers ...func(context.Context) error,
) error {
	runs := workers
	for _, service := range services {
		runs = append(runs, service.run)
	}
	return servergroup.Run(ctx, 0, nil, runs...)
}
