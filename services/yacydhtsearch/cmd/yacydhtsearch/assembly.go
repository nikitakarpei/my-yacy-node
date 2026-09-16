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

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/hostdiscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	networksearchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearchobservers/applog"
	networksearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearchobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	pagereadingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereadingobservers/applog"
	pagereadingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereadingobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	peercallobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/applog"
	peercallobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	peerchoiceobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoiceobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	peerdirectoryobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryobservers/applog"
	peerdirectoryobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryrefresh"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	peermatchedobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peermatchedobservers/applog"
	peermatchedobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peermatchedobservers/prometheus"
	peerpresencesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresences/jetstream"
	peerpresencesmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresences/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankings"
	queryrankingsobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/applog"
	queryrankingsobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/bywordcount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	rankingcachejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/jetstream"
	rankingcachememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/memory"
	rankingcacheobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/applog"
	rankingcacheobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalepeersources/leastreliable"
	wordjoinedobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordjoinedobservers/applog"
	wordjoinedobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordjoinedobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacysearchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
	yacyseedlistobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlistobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	opsReadHeaderLimit      = 10 * time.Second
	shutdownLimit           = 15 * time.Second
	rankingBucket           = "yacydhtsearch-rankings"
	rankingByteCeiling      = 32 * 1024
	peerAnswerHistoryStream = "yacydhtsearch-peer-answer-history"
	peerPresenceBucket      = "yacydhtsearch-peer-presence"
	presenceByteCeiling     = 512
	msgServiceStarted       = "yacydhtsearch started"
	msgServiceStopped       = "yacydhtsearch stopped"
	pageFetchUserAgent      = "yacydhtsearch (+https://yacy.net)"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) error {
	outbound := outboundClient(cfg)
	presence, err := peerPresenceFor(ctx, cfg)
	if err != nil {
		return err
	}

	reliability := peerreliability.New(
		presence, peerreliability.DefaultReliabilityWeights(), time.Now,
	)
	directory := peerdirectory.New(
		cfg.DirectoryCapacity,
		cfg.PeerChoiceCooldown,
		time.Now,
		leastreliable.New(reliability, cfg.RefreshInterval, time.Now),
		peerdirectory.DirectoryObservers{
			peerdirectoryobserversapplog.DirectoryLog{},
			peerdirectoryobserversprometheus.New(registry),
			presence,
		},
	)
	peers := peercallwire.New(
		outbound,
		peercallwire.SearchedNetwork{Name: cfg.NetworkName, RingPartitions: cfg.Partitions},
		peercallwire.PeerCallLimits{
			MaxResponseBytes:  cfg.MaxResponseBytes,
			PeerCallsInFlight: cfg.PeerCallsInFlight,
			PeerCallBudget:    cfg.PeerCallBudget,
		},
		peercallwire.PeerCallObservers{
			peercallobserversapplog.PeerCallLog{},
			peercallobserversprometheus.New(registry, cfg.QueryBudget),
		},
	)
	choice := peerchoice.New(
		cfg.Partitions,
		reliability,
		directory,
		peerchoiceobserversprometheus.New(registry),
	)
	pageReading, err := pageReadingFor(cfg, registry)
	if err != nil {
		return err
	}
	network := networksearch.New(
		directory,
		querySpreadFor(cfg, peers, choice, registry),
		pageReading,
		itemsOrderingOfTheService(),
		cfg.QueryBudget,
		cfg.PageReadBudget,
		cfg.PagesReadPerQuery,
		cfg.RankedItemsCeiling,
		networksearch.NetworkSearchObservers{
			networksearchobserversapplog.NetworkSearchLog{},
			networksearchobserversprometheus.New(registry, cfg.QueryBudget),
		},
	)
	rankingCacheMetrics := rankingcacheobserversprometheus.New(registry)
	cache, err := rankingCacheFor(ctx, cfg, rankingCacheMetrics)
	if err != nil {
		return err
	}
	rankings := queryrankings.New(cache, network, queryrankings.QueryRankingObservers{
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
		peerlivenesswire.New(outbound, cfg.NetworkName, peerlivenesswire.PeerLivenessObservers{}),
		presence,
		peerdirectoryrefresh.ProbeLimits{
			ProbeBudget:    cfg.ProbeBudget,
			ProbesInFlight: cfg.ProbesInFlight,
		},
	)
	go refresh.Run(ctx, cfg.RefreshInterval)

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

func querySpreadFor(
	cfg ServiceConfig,
	peers peercallwire.Wire,
	choice peerchoice.Choice,
	registry *prometheus.Registry,
) networksearch.QuerySpread {
	amountOfPeersHoldingOneWord := yacymodel.PeersHoldingOneWordOf(
		cfg.Partitions, cfg.NetworkRedundancy,
	)
	return bywordcount.New(
		wordjoined.New(
			peers,
			choice,
			cfg.RankedItemsCeiling,
			cfg.PeerItemsCeiling,
			amountOfPeersHoldingOneWord,
			wordjoined.WordJoinedSpreadObservers{
				wordjoinedobserversapplog.WordJoinedSpreadLog{},
				wordjoinedobserversprometheus.New(registry, cfg.QueryBudget),
			},
		),
		peermatched.New(
			peers,
			choice,
			cfg.PeerItemsCeiling,
			amountOfPeersHoldingOneWord,
			peermatched.PeerMatchedSpreadObservers{
				peermatchedobserversapplog.PeerMatchedSpreadLog{},
				peermatchedobserversprometheus.New(registry, cfg.QueryBudget),
			},
		),
	)
}

func pageReadingFor(
	cfg ServiceConfig,
	registry *prometheus.Registry,
) (networksearch.PageReading, error) {
	formatDerivations, err := pageformats.New()
	if err != nil {
		return nil, fmt.Errorf("page format derivations: %w", err)
	}

	return pagereading.New(
		pagefetchershttp.New(
			cfg.EgressProxyURL,
			pagefetchershttp.ProxyDialTunnel,
			pageFetchUserAgent,
			cfg.PageByteCeiling,
			cfg.PageReadBudget,
		),
		formatDerivations,
		cfg.PageReadBudget,
		cfg.SnippetLengthCeiling,
		pagereading.PageReadingObservers{
			pagereadingobserversapplog.PageReadingLog{},
			pagereadingobserversprometheus.New(registry, cfg.PageReadBudget),
		},
	), nil
}

func itemsOrderingOfTheService() networksearch.ItemsOrdering {
	return hostdiscount.New(documentrelevance.New(documentrelevance.DefaultScoreWeights()))
}

type peerPresence interface {
	peerdirectory.DirectoryObserver
	peerreliability.PeerPresence
	peerdirectoryrefresh.PeerPresence
}

func peerPresenceFor(
	ctx context.Context,
	cfg ServiceConfig,
) (peerPresence, error) {
	accrualLimits := presenceaccrual.PresenceAccrualLimits{
		Capacity:        cfg.DirectoryCapacity,
		ContinuityLimit: cfg.ContinuityLimit,
	}
	if cfg.NATSURL == "" {
		return peerpresencesmemory.New(
			accrualLimits,
			presenceaccrual.PresenceAccrualObservers{},
		), nil
	}

	answers, err := peerAnswerHistoryAt(ctx, cfg)
	if err != nil {
		return nil, err
	}
	snapshots, err := peerPresenceBucketAt(ctx, cfg)
	if err != nil {
		return nil, err
	}
	presence := peerpresencesjetstream.New(
		answers,
		snapshots,
		accrualLimits,
		presenceaccrual.PresenceAccrualObservers{},
		peerpresencesjetstream.PeerPresenceObservers{},
	)
	go presence.FoldThePeerAnswerHistory(ctx)

	return presence, nil
}

func peerAnswerHistoryAt(
	ctx context.Context,
	cfg ServiceConfig,
) (peeranswerhistory.History, error) {
	stream, _, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return peeranswerhistory.History{}, fmt.Errorf("%s: %w", EnvNATSURL, err)
	}

	_, err = stream.CreateOrUpdateStream(ctx, natsjetstream.StreamConfig{
		Name: peerAnswerHistoryStream,
		Subjects: []string{
			peeranswerhistory.SubjectOfEveryPeerAnswerIn(cfg.NetworkName),
		},
		MaxAge: cfg.PeerAnswerHistoryKeptFor,
	})
	if err != nil {
		return peeranswerhistory.History{},
			fmt.Errorf("open stream %s: %w", peerAnswerHistoryStream, err)
	}

	return peeranswerhistory.New(
		stream,
		peerAnswerHistoryStream,
		cfg.NetworkName,
		peeranswerhistory.HistoryObservers{},
	), nil
}

func peerPresenceBucketAt(
	ctx context.Context,
	cfg ServiceConfig,
) (natsjetstream.KeyValue, error) {
	stream, _, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvNATSURL, err)
	}

	bucket, err := stream.CreateOrUpdateKeyValue(ctx, natsjetstream.KeyValueConfig{
		Bucket:       peerPresenceBucket,
		MaxBytes:     int64(cfg.DirectoryCapacity) * presenceByteCeiling,
		MaxValueSize: presenceByteCeiling,
		History:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("open bucket %s: %w", peerPresenceBucket, err)
	}

	return bucket, nil
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
