package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/pprof"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/sitediscount"
	hedgedelaysconstant "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/hedgedelays/constant"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	networksearchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearchobservers/applog"
	networksearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearchobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	pagereadingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereadingobservers/applog"
	pagereadingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereadingobservers/prometheus"
	peercallobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/applog"
	peercallobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/prometheus"
	peercallobserversurlmetadataaskceilings "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/urlmetadataaskceilings"
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
	peerpresencejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/jetstream"
	peerpresencememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/memory"
	peerpresenceobserversjetstreamapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresenceobservers/jetstream/applog"
	peerpresenceobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresenceobservers/jetstream/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	presenceaccrualobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrualobservers/applog"
	presenceaccrualobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrualobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	probeanswerhistoryobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistoryobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankings"
	queryrankingsobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/applog"
	queryrankingsobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankingsobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/bywordcount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	queryspreadsobserverspeermatchedapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/peermatched/applog"
	queryspreadsobserverspeermatchedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/peermatched/prometheus"
	queryspreadsobserverswordjoinedapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/applog"
	queryspreadsobserverswordjoinedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/prometheus"
	queryworddocumentamountsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamounts/jetstream"
	queryworddocumentamountsmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamounts/memory"
	queryworddocumentamountsobserversjetstreamapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamountsobservers/jetstream/applog"
	queryworddocumentamountsobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamountsobservers/jetstream/prometheus"
	rankingcachejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/jetstream"
	rankingcachememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/memory"
	rankingcacheobserversjetstreamapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/jetstream/applog"
	rankingcacheobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/jetstream/prometheus"
	replicacallsyacysearch "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicacalls/yacysearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalepeersources/leastreliable"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings"
	peerpacesmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings/peerpaces/memory"
	urlmetadataaskceilingsobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilingsobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	wordpartitionasksobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasksobservers/applog"
	wordpartitionasksobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasksobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacysearchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
	yacyseedlistobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlistobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	opsReadHeaderLimit                 = 10 * time.Second
	shutdownLimit                      = 15 * time.Second
	rankingBucket                      = "yacydhtsearch-rankings"
	rankingByteCeiling                 = 32 * 1024
	queryWordDocumentAmountsBucket     = "yacydhtsearch-query-word-document-amounts"
	queryWordDocumentAmountByteCeiling = 64
	probeAnswerHistoryStream           = "yacydhtsearch-probe-answer-history"
	peerPresenceBucket                 = "yacydhtsearch-peer-presence"
	presenceByteCeiling                = 512
	msgServiceStarted                  = "yacydhtsearch started"
	msgServiceStopped                  = "yacydhtsearch stopped"
	pageFetchUserAgent                 = "yacydhtsearch (+https://yacy.net)"
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
	urlMetadataAskCeilings := urlmetadataaskceilings.New(
		cfg.URLMetadataAskDocumentsCeiling,
		cfg.URLMetadataAskDocumentsFloor,
		cfg.URLMetadataAskTargetTime,
		peerpacesmemory.New(),
		urlmetadataaskceilingsobserversapplog.AskCeilingLog{},
	)
	peers := peercallwire.New(
		outbound,
		wallclock.Clock{},
		peercallwire.SearchedNetwork{Name: cfg.NetworkName, RingPartitions: cfg.Partitions},
		peercallwire.PeerCallLimits{
			MaxResponseBytes:         cfg.MaxResponseBytes,
			PeerCallsInFlight:        cfg.PeerCallsInFlight,
			URLMetadataCallBudget:    cfg.URLMetadataCallBudget,
			SearchCallBudget:         cfg.SearchCallBudget,
			SearchCallHeadersTimeout: cfg.SearchCallHeadersTimeout,
		},
		peercallwire.PeerCallObservers{
			peercallobserversapplog.PeerCallLog{},
			peercallobserversprometheus.New(registry, cfg.QueryBudget),
			peercallobserversurlmetadataaskceilings.New(urlMetadataAskCeilings),
		},
	)
	choice := peerchoice.New(
		cfg.Partitions,
		cfg.NetworkRedundancy,
		reliability,
		peerchoice.PeerChoiceObservers{
			peerchoiceobserversapplog.PeerChoiceLog{},
			peerchoiceobserversprometheus.New(registry),
		},
	)
	pageReading, err := pageReadingFor(cfg, registry)
	if err != nil {
		return err
	}
	queryWordDocumentAmounts, err := queryWordDocumentAmountsFor(
		ctx, cfg, queryworddocumentamountsobserversjetstreamprometheus.New(registry),
	)
	if err != nil {
		return err
	}
	network := networksearch.New(
		directory,
		choice,
		querySpreadFor(cfg, peers, queryWordDocumentAmounts, urlMetadataAskCeilings, registry),
		pageReading,
		sitediscount.New(
			documentrelevance.RelevanceScorerWeighedBy(
				documentrelevance.DefaultRelevanceWeights(),
			),
		),
		cfg.QueryBudget,
		cfg.PageReadBudget,
		cfg.PagesReadPerQuery,
		cfg.PagesReadPerSite,
		cfg.RankedItemsCeiling,
		cfg.CompoundWordsCeiling,
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
			cfg.SeedlistReadBudget,
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
	opsMux := opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	if cfg.ServeProfiler {
		serveProfilerOn(opsMux)
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsMux,
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
	queryWordDocumentAmounts wordjoined.QueryWordDocumentAmounts,
	urlMetadataAskCeilings wordjoined.URLMetadataAskCeilings,
	registry *prometheus.Registry,
) networksearch.QuerySpread {
	hedgeDelay := hedgedelaysconstant.New(cfg.HedgeDelay)
	replicaAsksObservers := wordpartitionasks.ReplicaAsksObservers{
		wordpartitionasksobserversapplog.ReplicaAsksLog{},
		wordpartitionasksobserversprometheus.New(registry, cfg.QueryBudget),
	}
	wordJoinedReplicaAsks := wordpartitionasks.New(
		replicacallsyacysearch.New(peers, replicacallsyacysearch.Wants{
			Abstract:                true,
			MatchedDocumentsCeiling: yacymodel.Some(cfg.PeerItemsCeiling),
		}),
		hedgeDelay,
		wallclock.Clock{},
		cfg.ReplicasCoveringAPartition,
		replicaAsksObservers,
	)
	peerMatchedReplicaAsks := wordpartitionasks.New(
		replicacallsyacysearch.New(peers, replicacallsyacysearch.Wants{
			Abstract:                true,
			MatchedDocumentsCeiling: yacymodel.Some(cfg.PeerItemsCeiling),
		}),
		hedgeDelay,
		wallclock.Clock{},
		cfg.ReplicasCoveringAPartition,
		replicaAsksObservers,
	)

	return bywordcount.New(
		wordjoined.New(
			wordJoinedReplicaAsks,
			peers,
			queryWordDocumentAmounts,
			cfg.URLMetadataLookupCutoff,
			rand.UintN,
			urlMetadataAskCeilings,
			cfg.DocumentsToMatchCeiling,
			cfg.Partitions,
			yacymodel.PeersHoldingOneWordOf(cfg.Partitions, cfg.NetworkRedundancy),
			wordjoined.WordJoinedSpreadObservers{
				queryspreadsobserverswordjoinedapplog.WordJoinedSpreadLog{},
				queryspreadsobserverswordjoinedprometheus.New(registry, cfg.QueryBudget),
			},
		),
		peermatched.New(
			peerMatchedReplicaAsks,
			peermatched.PeerMatchedSpreadObservers{
				queryspreadsobserverspeermatchedapplog.PeerMatchedSpreadLog{},
				queryspreadsobserverspeermatchedprometheus.New(registry, cfg.QueryBudget),
			},
		),
	)
}

func queryWordDocumentAmountsFor(
	ctx context.Context,
	cfg ServiceConfig,
	metrics *queryworddocumentamountsobserversjetstreamprometheus.QueryWordDocumentAmountsMetrics,
) (wordjoined.QueryWordDocumentAmounts, error) {
	if cfg.NATSURL == "" {
		return queryworddocumentamountsmemory.New(
			cfg.QueryWordDocumentAmountsCapacity, cfg.QueryWordDocumentAmountLifetime,
		), nil
	}

	bucket, err := queryWordDocumentAmountsBucketAt(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return queryworddocumentamountsjetstream.New(
		bucket,
		queryworddocumentamountsjetstream.QueryWordDocumentAmountsObservers{
			queryworddocumentamountsobserversjetstreamapplog.QueryWordDocumentAmountsLog{},
			metrics,
		},
	), nil
}

func queryWordDocumentAmountsBucketAt(
	ctx context.Context,
	cfg ServiceConfig,
) (natsjetstream.KeyValue, error) {
	stream, _, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvNATSURL, err)
	}

	bucket, err := stream.CreateOrUpdateKeyValue(ctx, natsjetstream.KeyValueConfig{
		Bucket: queryWordDocumentAmountsBucket,
		TTL:    cfg.QueryWordDocumentAmountLifetime,
		MaxBytes: int64(
			cfg.QueryWordDocumentAmountsCapacity,
		) * queryWordDocumentAmountByteCeiling,
		MaxValueSize: queryWordDocumentAmountByteCeiling,
		History:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("open bucket %s: %w", queryWordDocumentAmountsBucket, err)
	}

	return bucket, nil
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
		redirectfollowingfetch.New(
			pagefetchershttp.New(
				cfg.PageReadProxyURL,
				cfg.PageReadProxyDialMode,
				pageFetchUserAgent,
				cfg.PageByteCeiling,
				cfg.PageReadBudget,
			),
			cfg.PageReadMaxRedirectHops,
		),
		formatDerivations,
		cfg.PageReadBudget,
		cfg.PageReadCutoff,
		cfg.SnippetLengthCeiling,
		pagereading.PageReadingObservers{
			pagereadingobserversapplog.PageReadingLog{},
			pagereadingobserversprometheus.New(registry, cfg.PageReadBudget),
		},
	), nil
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

	answers, err := probeAnswerHistoryAt(ctx, cfg)
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
	go presence.FoldTheProbeAnswerHistory(ctx)

	return presence, nil
}

func probeAnswerHistoryAt(
	ctx context.Context,
	cfg ServiceConfig,
) (probeanswerhistory.History, error) {
	stream, _, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return probeanswerhistory.History{}, fmt.Errorf("%s: %w", EnvNATSURL, err)
	}

	_, err = stream.CreateOrUpdateStream(ctx, natsjetstream.StreamConfig{
		Name: probeAnswerHistoryStream,
		Subjects: []string{
			probeanswerhistory.SubjectOfEveryProbeAnswerIn(cfg.NetworkName),
		},
		MaxAge: cfg.ProbeAnswerHistoryKeptFor,
	})
	if err != nil {
		return probeanswerhistory.History{},
			fmt.Errorf("open stream %s: %w", probeAnswerHistoryStream, err)
	}

	return probeanswerhistory.New(
		stream,
		probeAnswerHistoryStream,
		cfg.NetworkName,
		probeanswerhistory.HistoryObservers{
			probeanswerhistoryobserversapplog.ProbeAnswerHistoryLog{},
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

func serveProfilerOn(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

func outboundClient(cfg ServiceConfig) *http.Client {
	return &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(cfg.EgressProxyURL)},
	}
}
