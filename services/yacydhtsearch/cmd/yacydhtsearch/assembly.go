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
	peeranswerhistoryobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistoryobservers/applog"
	peercallobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/applog"
	peercallobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	peerchoiceobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoiceobservers/applog"
	peerchoiceobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoiceobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	peerdirectoryobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryobservers/applog"
	peerdirectoryobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryrefresh"
	peerlivenessobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenessobservers/applog"
	peerlivenessobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenessobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	peermatchedobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peermatchedobservers/applog"
	peermatchedobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peermatchedobservers/prometheus"
	peerpresencejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/jetstream"
	peerpresencememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/memory"
	peerpresenceobserversjetstreamapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresenceobservers/jetstream/applog"
	peerpresenceobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresenceobservers/jetstream/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	presenceaccrualobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrualobservers/applog"
	presenceaccrualobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrualobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankings"
	queryrankingsobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/applog"
	queryrankingsobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/bywordcount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	rankingcachejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/jetstream"
	rankingcachememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/memory"
	rankingcacheobserversjetstreamapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/jetstream/applog"
	rankingcacheobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/jetstream/prometheus"
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
	presence, err := peerPresenceFor(ctx, cfg, registry)
	if err != nil {
		return err
	}

	reliability := peerreliability.New(
		presence,
		peerreliability.ReliabilityWeights{
			MaturationDuration: cfg.MaturationDuration,
			StalenessHorizon:   cfg.StalenessHorizon,
		},
		time.Now,
	)
	directory := peerdirectory.New(
		peerdirectory.DirectoryLimits{
			Capacity:      cfg.DirectoryCapacity,
			Cooldown:      cfg.PeerChoiceCooldown,
			NewcomerShare: cfg.NewcomerShare,
		},
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
		cfg.NetworkRedundancy,
		reliability,
		directory,
		peerchoice.PeerChoiceObservers{
			peerchoiceobserversapplog.PeerChoiceLog{},
			peerchoiceobserversprometheus.New(registry),
		},
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
	rankingCacheMetrics := rankingcacheobserversjetstreamprometheus.New(registry)
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
		peerlivenesswire.New(outbound, cfg.NetworkName, peerlivenesswire.PeerLivenessObservers{
			peerlivenessobserversapplog.PeerLivenessLog{},
			peerlivenessobserversprometheus.New(registry),
		}),
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
	return bywordcount.New(
		wordjoined.New(
			peers,
			choice,
			cfg.RankedItemsCeiling,
			cfg.PeerItemsCeiling,
			yacymodel.PeersHoldingOneWordOf(cfg.Partitions, cfg.NetworkRedundancy),
			wordjoined.WordJoinedSpreadObservers{
				wordjoinedobserversapplog.WordJoinedSpreadLog{},
				wordjoinedobserversprometheus.New(registry, cfg.QueryBudget),
			},
		),
		peermatched.New(
			peers,
			choice,
			cfg.PeerItemsCeiling,
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
	registry *prometheus.Registry,
) (peerPresence, error) {
	accrualLimits := presenceaccrual.PresenceAccrualLimits{
		Capacity:        cfg.DirectoryCapacity,
		ContinuityLimit: cfg.ContinuityLimit,
	}
	accrualObservers := presenceaccrual.PresenceAccrualObservers{
		presenceaccrualobserversapplog.PresenceAccrualLog{},
		presenceaccrualobserversprometheus.New(registry),
	}
	if cfg.NATSURL == "" {
		return peerpresencememory.New(accrualLimits, accrualObservers), nil
	}

	answers, err := peerAnswerHistoryAt(ctx, cfg)
	if err != nil {
		return nil, err
	}
	snapshots, err := peerPresenceBucketAt(ctx, cfg)
	if err != nil {
		return nil, err
	}
	presence := peerpresencejetstream.New(
		answers,
		snapshots,
		peerpresencejetstream.PeerPresenceLimits{
			AccrualLimits:    accrualLimits,
			SnapshotInterval: cfg.SnapshotInterval,
		},
		accrualObservers,
		peerpresencejetstream.PeerPresenceObservers{
			peerpresenceobserversjetstreamapplog.PeerPresenceLog{},
			peerpresenceobserversjetstreamprometheus.New(registry),
		},
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
		peeranswerhistory.HistoryObservers{
			peeranswerhistoryobserversapplog.PeerAnswerHistoryLog{},
		},
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
	metrics *rankingcacheobserversjetstreamprometheus.RankingMetrics,
) (queryrankings.RankingCache, error) {
	if cfg.NATSURL == "" {
		return rankingcachememory.New(cfg.RankingCache, cfg.RankingLifetime), nil
	}

	bucket, err := rankingBucketAt(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return rankingcachejetstream.New(bucket, rankingcachejetstream.RankingCacheObservers{
		rankingcacheobserversjetstreamapplog.RankingLog{},
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
