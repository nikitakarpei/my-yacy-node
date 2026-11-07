package yacypeerassembly

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/cachedpages"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/metrichistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pagecacheproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/peerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/visitlinksecret"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuiproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

const (
	brokerName                 = "nats"
	brokerDirectoryName        = brokerName
	metricHistoryDirectoryName = "metric-history"
	peerEgressProxyName        = "peeregressproxy"
	pageFetchEgressProxyName   = "pagefetchegressproxy"
	cachedPagesDirectoryName   = "cached-pages"
	cachedPageRetention        = 30 * time.Minute
	cachedPagesMaxBytes        = 512 << 20
	pageMaxBodyBytes           = 4 << 20
	processHostSuffix          = ".internal"
)

func RunPeer(
	ctx context.Context,
	cfg PeerConfig,
	web net.Listener,
	getenv func(string) string,
) error {
	visitLinkSecret, err := visitlinksecret.Settle(cfg.directoryOf(visitcrawlService))
	if err != nil {
		return errors.Join(err, web.Close())
	}
	peerKey, err := peerkey.Settle(ctx, cfg.DataDirectory, cfg.PeerKeySearchTime, newPeerKey)
	if err != nil {
		return errors.Join(err, web.Close())
	}
	network := processnetwork.New()
	broker, companionParts, err := startBroker(cfg)
	if err != nil {
		return errors.Join(err, web.Close())
	}
	network.Attach(hostOf(brokerName), broker.Connection)
	defer closeCompanions(ctx, companionParts)
	defer broker.Shutdown()
	history, err := metrichistory.Open(
		ctx,
		cfg.directoryOf(metricHistoryDirectoryName),
		cfg.MetricHistoryRetention,
		wallclock.Clock{},
	)
	if err != nil {
		return errors.Join(err, web.Close())
	}
	defer closeHistory(ctx, history)
	peerEgress := egressproxy.New(network.ListenerAt(hostOf(peerEgressProxyName)), cfg.PeerProxyURL)
	pageFetchEgress := egressproxy.New(
		network.ListenerAt(hostOf(pageFetchEgressProxyName)),
		cfg.PageFetchProxyURL,
	)
	processRegistry := opsmetrics.ProcessRegistry()
	cachedPages, err := cachedpages.Open(
		servicelog.ContextFor(ctx, cachedpages.Name),
		cfg.directoryOf(cachedPagesDirectoryName),
		cachedpages.Limits{Retention: cachedPageRetention, MaxBytes: cachedPagesMaxBytes},
		wallclock.Clock{},
		processRegistry,
	)
	if err != nil {
		return errors.Join(err, closeEvery([]io.Closer{web, peerEgress, pageFetchEgress}))
	}
	cacheProxy := pagecacheproxy.New(
		network.ListenerAt(hostOf(pagecacheproxy.Name)),
		cachedPages,
		&http.Transport{
			Proxy:       http.ProxyURL(pageFetchEgress.URL()),
			DialContext: network.DialContext,
		},
		pageMaxBodyBytes,
		processRegistry,
	)
	peer := peer{
		peerEgressProxyURL: peerEgress.URL(),
		pageCacheProxyURL:  cacheProxy.URL(),
		cfg:                cfg,
		natsURL:            (&url.URL{Scheme: "nats", Host: hostOf(brokerName)}).String(),
		network:            network,
		visitLinkSecret:    visitLinkSecret,
		peerKey:            peerKey,
	}
	services, urls, err := peer.services(ctx, getenv)
	if err != nil {
		return errors.Join(
			err,
			closeEvery([]io.Closer{web, peerEgress, pageFetchEgress, cacheProxy}),
		)
	}
	logStarted(ctx, peer, web)
	webMux := webMuxFor(processRegistry, services, history, urls, network)
	companionServers := companionParts.servers(broker, peer.natsURL, network)
	return opsmetrics.ServeAlongside(ctx, web, webMux, func(ctx context.Context) error {
		return runEvery(ctx, services, append(companionServers,
			func(ctx context.Context) error {
				return peerEgress.Run(servicelog.ContextFor(ctx, peerEgressProxyName))
			},
			func(ctx context.Context) error {
				return pageFetchEgress.Run(servicelog.ContextFor(ctx, pageFetchEgressProxyName))
			},
			func(ctx context.Context) error {
				return cacheProxy.Run(servicelog.ContextFor(ctx, pagecacheproxy.Name))
			},
			func(ctx context.Context) error {
				return cachedPages.Run(servicelog.ContextFor(ctx, cachedpages.Name))
			},
			func(ctx context.Context) error {
				return history.Run(ctx, metricSourcesOf(processRegistry, services))
			},
		)...)
	})
}

func newPeerKey(ctx context.Context, searchTime time.Duration) (yacymodel.PeerKey, error) {
	return yacymodel.NewPeerKey(ctx, searchTime, wallclock.Clock{})
}

func hostOf(part string) string {
	return part + processHostSuffix
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
	web net.Listener,
) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", peer.natsURL),
		slog.String("peerEgressProxyUrl", peer.peerEgressProxyURL.Redacted()),
		slog.String("webAddr", web.Addr().String()),
	)
}

func webMuxFor(
	processRegistry *prometheus.Registry,
	services []service,
	history *metrichistory.History,
	urls serviceURLs,
	network *processnetwork.Network,
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
	mux.Handle(
		crawlConsolePathPrefix+"/",
		webuiproxy.To(urls[crawlconsoleService], network.DialContext),
	)
	mux.Handle("/", webuiproxy.To(urls[searchpageService], network.DialContext))
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
