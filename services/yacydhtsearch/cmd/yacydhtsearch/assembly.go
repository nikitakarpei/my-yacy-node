package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	dhtdistanceobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/dhtdistanceobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	networksearchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearchobservers/applog"
	networksearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearchobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	peerdirectoryobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryobservers/applog"
	peerdirectoryobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryrefresh"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearch"
	peersearchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchobservers/applog"
	peersearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerselections/dhtdistance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankings"
	queryrankingsobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/applog"
	queryrankingsobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/prometheus"
	rankingcachejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/jetstream"
	rankingcachememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/memory"
	rankingcacheobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/applog"
	rankingcacheobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalepeersources/leastrecentlyanswered"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacysearchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
	yacyseedlistobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlistobservers/applog"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	shutdownLimit      = 15 * time.Second
	rankingBucket      = "yacydhtsearch-rankings"
	rankingByteCeiling = 32 * 1024
	msgServiceStarted  = "yacydhtsearch started"
	msgServiceStopped  = "yacydhtsearch stopped"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) error {
	outbound := outboundClient(cfg)
	directory := peerdirectory.New(
		cfg.DirectoryCapacity,
		cfg.PeerSearchCooldown,
		time.Now,
		leastrecentlyanswered.New(),
		peerdirectory.DirectoryObservers{
			peerdirectoryobserversapplog.DirectoryLog{},
			peerdirectoryobserversprometheus.New(registry),
		},
	)
	network := networksearch.New(
		cfg.NetworkName,
		directory,
		dhtdistance.New(
			cfg.Partitions,
			cfg.PeerRedundancy,
			dhtdistanceobserversprometheus.New(registry),
		),
		peersearch.New(
			peersearchwire.New(
				outbound,
				cfg.MaxResponseBytes,
				peersearchwire.PeerSearchObservers{
					peersearchobserversapplog.PeerSearchLog{},
					peersearchobserversprometheus.New(
						registry,
						cfg.PeerCallBudget,
						cfg.PeerItemsCeiling,
					),
				},
			),
			cfg.PeerCallsInFlight,
			cfg.PeerCallBudget,
		),
		cfg.QueryBudget,
		peerBudgetCeiling,
		cfg.PeerItemsCeiling,
		cfg.RankedItemsCeiling,
		cfg.Partitions,
		networksearch.NetworkSearchObservers{
			networksearchobserversapplog.NetworkSearchLog{},
			networksearchobserversprometheus.New(registry, cfg.QueryBudget),
		},
	)
	rankingMetrics := rankingcacheobserversprometheus.New(registry)
	cache, err := rankingCacheFor(ctx, cfg, rankingMetrics)
	if err != nil {
		return err
	}
	rankings := queryrankings.New(cache, network, queryrankings.RankingObservers{
		queryrankingsobserversapplog.QueryRankingLog{},
		queryrankingsobserversprometheus.New(registry),
	})
	refresh := peerdirectoryrefresh.New(
		yacyseedlist.New(
			outbound,
			cfg.SeedlistURLs,
			cfg.MaxResponseBytes,
			yacyseedlistobserversapplog.SeedlistLog{},
		),
		directory,
		peerlivenesswire.New(outbound, cfg.NetworkName),
		cfg.RefreshInterval,
		cfg.ProbeBudget,
		cfg.PeerCallsInFlight,
	)
	go refresh.Run(ctx)

	searchServer := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: httpobservation.NewHandler(
			yacysearchendpoint.NewMux(rankings),
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "yacydhtsearch"),
		),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", cfg.ListenAddr),
		slog.String("networkName", cfg.NetworkName),
		slog.Int("seedlists", len(cfg.SeedlistURLs)),
	)
	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "search", Server: searchServer},
		{Name: "ops", Server: opsServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)

	return err
}

func rankingCacheFor(
	ctx context.Context,
	cfg ServiceConfig,
	metrics *rankingcacheobserversprometheus.RankingMetrics,
) (queryrankings.RankingCache, error) {
	if cfg.NATSURL == "" {
		return rankingcachememory.New(cfg.RankingCache, cfg.RankingLifetime), nil
	}

	bucket, err := rankingBucketAt(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return rankingcachejetstream.New(bucket, rankingcachejetstream.RankingCacheObservers{
		rankingcacheobserversapplog.RankingLog{},
		metrics,
	}), nil
}

func rankingBucketAt(ctx context.Context, cfg ServiceConfig) (natsjetstream.KeyValue, error) {
	stream, _, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvNATSURL, err)
	}

	bucket, err := stream.CreateOrUpdateKeyValue(ctx, natsjetstream.KeyValueConfig{
		Bucket:       rankingBucket,
		TTL:          cfg.RankingLifetime,
		MaxBytes:     int64(cfg.RankingCache) * rankingByteCeiling,
		MaxValueSize: rankingByteCeiling,
		History:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("open bucket %s: %w", rankingBucket, err)
	}

	return bucket, nil
}

func outboundClient(cfg ServiceConfig) *http.Client {
	return &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(cfg.EgressProxyURL)},
	}
}
