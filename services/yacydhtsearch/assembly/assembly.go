package yacydhtsearchassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	cachedrankingsmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/cachedrankings/memory"
	cachedrankingsredis "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/cachedrankings/redis"
	cachedrankingsobserversredisapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/cachedrankingsobservers/redis/applog"
	cachedrankingsobserversredisprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/cachedrankingsobservers/redis/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/sitediscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/spamlast"
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
	peerpresencememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/memory"
	peerpresenceredis "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/redis"
	peerpresenceobserversredisapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresenceobservers/redis/applog"
	peerpresenceobserversredisprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresenceobservers/redis/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	presenceaccrualobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrualobservers/applog"
	presenceaccrualobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrualobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	probeanswerhistoryobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistoryobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/bywordcount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	queryspreadsobserverspeermatchedapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/peermatched/applog"
	queryspreadsobserverspeermatchedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/peermatched/prometheus"
	queryspreadsobserverswordjoinedapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/applog"
	queryspreadsobserverswordjoineddocumentamountsapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/documentamounts/applog"
	queryspreadsobserverswordjoineddocumentamountsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/documentamounts/prometheus"
	queryspreadsobserverswordjoineddocumentasksapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/documentasks/applog"
	queryspreadsobserverswordjoineddocumentasksprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/documentasks/prometheus"
	queryspreadsobserverswordjoinedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/prometheus"
	queryspreadsobserverswordjoinedurlmetadataasksapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/urlmetadataasks/applog"
	queryspreadsobserverswordjoinedurlmetadataasksprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/urlmetadataasks/prometheus"
	queryworddocumentamountsmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamounts/memory"
	queryworddocumentamountsredis "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamounts/redis"
	queryworddocumentamountsobserversredisapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamountsobservers/redis/applog"
	queryworddocumentamountsobserversredisprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamountsobservers/redis/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache"
	rankingcacheobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/applog"
	rankingcacheobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/prometheus"
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
	serverReadHeaderLimit = 10 * time.Second
	shutdownLimit         = 15 * time.Second
	msgServiceStarted     = "yacydhtsearch started"
	msgServiceStopped     = "yacydhtsearch stopped"
)

type Listeners struct {
	Search net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	search, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Search: search}, nil
}

func (l Listeners) Close() error {
	if err := l.Search.Close(); err != nil {
		return fmt.Errorf("close the search listener: %w", err)
	}
	return nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "yacydhtsearch_info",
		Help: "YaCy DHT search application identity.",
	}, func() float64 { return 1 }))
	outbound := outboundClient(cfg)
	redisClient, err := openRedisClient(ctx, cfg)
	if err != nil {
		return err
	}
	if redisClient != nil {
		defer func() { _ = redisClient.Close() }()
	}
	presence := peerPresenceFor(ctx, cfg, redisClient, registry)

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
	queryWordDocumentAmounts := queryWordDocumentAmountsFor(
		cfg, redisClient, queryworddocumentamountsobserversredisprometheus.New(registry),
	)
	network := networksearch.New(
		directory,
		choice,
		querySpreadFor(cfg, peers, queryWordDocumentAmounts, urlMetadataAskCeilings, registry),
		pageReading,
		spamlast.New(
			sitediscount.New(
				documentrelevance.RelevanceScorerWeighedBy(
					documentrelevance.DefaultRelevanceWeights(),
				),
			),
		),
		cfg.HideUnreadResults,
		cfg.QueryBudget,
		cfg.PagesReadPerQueryCeiling,
		cfg.PagesReadPerSiteCeiling,
		cfg.RankedItemsCeiling,
		cfg.CompoundWordsCeiling,
		networksearch.NetworkSearchObservers{
			networksearchobserversapplog.NetworkSearchLog{},
			networksearchobserversprometheus.New(registry, cfg.QueryBudget),
		},
	)
	cachedRankingsMetrics := cachedrankingsobserversredisprometheus.New(registry)
	cachedRankings := cachedRankingsFor(cfg, redisClient, cachedRankingsMetrics)
	rankingCache := rankingcache.New(cachedRankings, network, rankingcache.RankingCacheObservers{
		rankingcacheobserversapplog.RankingCacheLog{},
		rankingcacheobserversprometheus.New(registry),
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
		Handler: httpobservation.NewHandler(
			yacysearchendpoint.NewMux(rankingCache),
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "yacydhtsearch"),
		),
		ReadHeaderTimeout: serverReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", listeners.Search.Addr().String()),
		slog.String("networkName", cfg.NetworkName),
		slog.Int("seedlists", len(cfg.SeedlistURLs)),
	)
	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "search", Listener: listeners.Search, Server: searchServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)

	return err
}

func querySpreadFor(
	cfg ServiceConfig,
	peers peercallwire.Wire,
	queryWordDocumentAmounts queryWordDocumentAmounts,
	urlMetadataAskCeilings urlmetadataasks.Ceilings,
	registry *prometheus.Registry,
) networksearch.QuerySpread {
	hedgeDelay := hedgedelaysconstant.New(cfg.HedgeDelay)
	wordPartitionAsksObservers := wordpartitionasks.WordPartitionAsksObservers{
		wordpartitionasksobserversapplog.WordPartitionAsksLog{},
		wordpartitionasksobserversprometheus.New(registry, cfg.QueryBudget),
	}
	wordJoinedWordPartitionAsks := wordpartitionasks.New(
		replicacallsyacysearch.New(peers, replicacallsyacysearch.Wants{
			Abstract:                true,
			MatchedDocumentsCeiling: yacymodel.Some(cfg.WordJoinedPeerItemsCeiling),
		}),
		hedgeDelay,
		wallclock.Clock{},
		cfg.ReplicasCoveringAPartition,
		wordPartitionAsksObservers,
	)
	peerMatchedWordPartitionAsks := wordpartitionasks.New(
		replicacallsyacysearch.New(peers, replicacallsyacysearch.Wants{
			Abstract:                true,
			MatchedDocumentsCeiling: yacymodel.Some(cfg.PeerMatchedPeerItemsCeiling),
		}),
		hedgeDelay,
		wallclock.Clock{},
		cfg.ReplicasCoveringAPartition,
		wordPartitionAsksObservers,
	)

	return bywordcount.New(
		wordjoined.New(
			documentasks.New(
				wordJoinedWordPartitionAsks,
				cfg.Partitions,
				cfg.DocumentsToMatchCeiling,
				documentasks.DocumentAsksObservers{
					queryspreadsobserverswordjoineddocumentasksapplog.DocumentAsksLog{},
					queryspreadsobserverswordjoineddocumentasksprometheus.New(registry),
				},
			),
			queryWordDocumentAmounts,
			leadingword.New(documentAmountsFor(queryWordDocumentAmounts, registry)),
			urlmetadataasks.New(
				peers,
				urlMetadataAskCeilings,
				cfg.URLMetadataLookupCutoff,
				wallclock.Clock{},
				cfg.URLMetadataAsksPerDocument,
				urlmetadataasks.URLMetadataLookupObservers{
					queryspreadsobserverswordjoinedurlmetadataasksapplog.URLMetadataLookupLog{},
					queryspreadsobserverswordjoinedurlmetadataasksprometheus.New(
						registry, cfg.QueryBudget,
					),
				},
			),
			cfg.Partitions,
			wordjoined.WordJoinedSpreadObservers{
				queryspreadsobserverswordjoinedapplog.WordJoinedSpreadLog{},
				queryspreadsobserverswordjoinedprometheus.New(registry, cfg.QueryBudget),
			},
		),
		peermatched.New(
			peerMatchedWordPartitionAsks,
			peermatched.PeerMatchedSpreadObservers{
				queryspreadsobserverspeermatchedapplog.PeerMatchedSpreadLog{},
				queryspreadsobserverspeermatchedprometheus.New(registry, cfg.QueryBudget),
			},
		),
	)
}

func documentAmountsFor(
	queryWordDocumentAmounts queryWordDocumentAmounts,
	registry *prometheus.Registry,
) documentamounts.FromCache {
	documentAmountsMetrics := queryspreadsobserverswordjoineddocumentamountsprometheus.New(registry)

	return documentamounts.NewFromCache(
		queryWordDocumentAmounts,
		documentamounts.FromCacheObservers{
			queryspreadsobserverswordjoineddocumentamountsapplog.DocumentAmountsLog{},
			documentAmountsMetrics,
		},
	)
}

type queryWordDocumentAmounts interface {
	documentamounts.CachedDocumentAmounts
	wordjoined.RememberedDocumentAmounts
}

func queryWordDocumentAmountsFor(
	cfg ServiceConfig,
	redisClient redis.UniversalClient,
	metrics *queryworddocumentamountsobserversredisprometheus.QueryWordDocumentAmountsMetrics,
) queryWordDocumentAmounts {
	if redisClient == nil {
		return queryworddocumentamountsmemory.New(
			cfg.QueryWordDocumentAmountsCapacity, cfg.QueryWordDocumentAmountLifetime,
		)
	}

	return queryworddocumentamountsredis.New(
		redisClient,
		cfg.QueryWordDocumentAmountLifetime,
		queryworddocumentamountsredis.QueryWordDocumentAmountsObservers{
			queryworddocumentamountsobserversredisapplog.QueryWordDocumentAmountsLog{},
			metrics,
		},
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

	return networkSearchPageReading{reading: pagereading.New(
		redirectfollowingfetch.New(
			pagefetchershttp.New(
				cfg.PageReadProxyURL,
				cfg.PageReadProxyDialMode,
				cfg.PageReadUserAgent,
				cfg.PageByteCeiling,
				cfg.PageReadBudget,
			),
			cfg.PageReadMaxRedirectHops,
		),
		formatDerivations,
		cfg.PageReadBudget,
		cfg.PageReadCutoff,
		wallclock.Clock{},
		cfg.SnippetLengthCeiling,
		pagereading.PageReadingObservers{
			pagereadingobserversapplog.PageReadingLog{},
			pagereadingobserversprometheus.New(registry, cfg.PageReadBudget),
		},
	)}, nil
}

type peerPresence interface {
	peerdirectory.DirectoryObserver
	peerreliability.PeerPresence
	peerdirectoryrefresh.PeerPresence
}

func openRedisClient(ctx context.Context, cfg ServiceConfig) (redis.UniversalClient, error) {
	if cfg.StateRedis == nil {
		return nil, nil
	}
	redisClient := redis.NewClient(cfg.StateRedis)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()

		return nil, fmt.Errorf("%s: %w", EnvStateRedisURL, err)
	}

	return redisClient, nil
}

func peerPresenceFor(
	ctx context.Context,
	cfg ServiceConfig,
	redisClient redis.UniversalClient,
	registry *prometheus.Registry,
) peerPresence {
	accrualLimits := presenceaccrual.PresenceAccrualLimits{
		Capacity:        cfg.DirectoryCapacity,
		ContinuityLimit: cfg.ContinuityLimit,
	}
	accrualObservers := presenceaccrual.PresenceAccrualObservers{
		presenceaccrualobserversapplog.PresenceAccrualLog{},
		presenceaccrualobserversprometheus.New(registry),
	}
	if redisClient == nil {
		return peerpresencememory.New(accrualLimits, accrualObservers)
	}

	presence := peerpresenceredis.New(
		probeAnswerHistoryOn(redisClient, cfg),
		redisClient,
		peerpresenceredis.PeerPresenceLimits{
			AccrualLimits:    accrualLimits,
			SnapshotInterval: cfg.SnapshotInterval,
		},
		accrualObservers,
		peerpresenceredis.PeerPresenceObservers{
			peerpresenceobserversredisapplog.PeerPresenceLog{},
			peerpresenceobserversredisprometheus.New(registry),
		},
	)
	go presence.FoldTheProbeAnswerHistory(ctx)

	return presence
}

func probeAnswerHistoryOn(
	redisClient redis.UniversalClient,
	cfg ServiceConfig,
) probeanswerhistory.History {
	return probeanswerhistory.New(
		redisClient,
		cfg.NetworkName,
		cfg.ProbeAnswerHistoryKeptFor,
		wallclock.Clock{},
		probeanswerhistory.HistoryObservers{
			probeanswerhistoryobserversapplog.ProbeAnswerHistoryLog{},
		},
	)
}

func cachedRankingsFor(
	cfg ServiceConfig,
	redisClient redis.UniversalClient,
	metrics *cachedrankingsobserversredisprometheus.CachedRankingsMetrics,
) rankingcache.CachedRankings {
	if redisClient == nil {
		return cachedrankingsmemory.New(cfg.RankingCacheCapacity, cfg.RankingLifetime)
	}

	return cachedrankingsredis.New(
		redisClient,
		cfg.RankingLifetime,
		cachedrankingsredis.CachedRankingsObservers{
			cachedrankingsobserversredisapplog.CachedRankingsLog{},
			metrics,
		},
	)
}

func outboundClient(cfg ServiceConfig) *http.Client {
	return &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(cfg.EgressProxyURL)},
	}
}
