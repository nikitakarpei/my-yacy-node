package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvListenAddr                       = "YACYDHTSEARCH_LISTEN_ADDR"
	EnvOpsAddr                          = "YACYDHTSEARCH_OPS_ADDR"
	EnvServeProfiler                    = "YACYDHTSEARCH_SERVE_PROFILER"
	EnvNetworkName                      = "YACYDHTSEARCH_NETWORK_NAME"
	EnvSeedlistURLs                     = "YACYDHTSEARCH_SEEDLIST_URLS"
	EnvSeedlistReadBudget               = "YACYDHTSEARCH_SEEDLIST_READ_BUDGET"
	EnvEgressProxyURL                   = "EGRESS_PROXY_URL"
	EnvPageReadProxyURL                 = "YACYDHTSEARCH_PAGE_READ_PROXY_URL"
	EnvPageReadProxyDialMode            = "YACYDHTSEARCH_PAGE_READ_PROXY_DIAL_MODE"
	EnvQueryBudget                      = "YACYDHTSEARCH_QUERY_BUDGET"
	EnvNetworkRedundancy                = "YACYDHTSEARCH_NETWORK_REDUNDANCY"
	EnvReplicasCoveringAPartition       = "YACYDHTSEARCH_REPLICAS_COVERING_A_PARTITION"
	EnvHedgeDelay                       = "YACYDHTSEARCH_HEDGE_DELAY"
	EnvPeerCallsInFlight                = "YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT"
	EnvURLMetadataCallBudget            = "YACYDHTSEARCH_URL_METADATA_CALL_BUDGET"
	EnvSearchCallBudget                 = "YACYDHTSEARCH_SEARCH_CALL_BUDGET"
	EnvProbesInFlight                   = "YACYDHTSEARCH_PROBES_IN_FLIGHT"
	EnvDirectoryCapacity                = "YACYDHTSEARCH_DIRECTORY_CAPACITY"
	EnvRefreshInterval                  = "YACYDHTSEARCH_REFRESH_INTERVAL"
	EnvNewcomerShare                    = "YACYDHTSEARCH_DIRECTORY_NEWCOMER_SHARE"
	EnvProbeBudget                      = "YACYDHTSEARCH_PROBE_BUDGET"
	EnvContinuityLimit                  = "YACYDHTSEARCH_PEER_PRESENCE_CONTINUITY_LIMIT"
	EnvProbeAnswerHistoryKeptFor        = "YACYDHTSEARCH_PROBE_ANSWER_HISTORY_KEPT_FOR"
	EnvMaturationDuration               = "YACYDHTSEARCH_PEER_RELIABILITY_MATURATION_DURATION"
	EnvStalenessHorizon                 = "YACYDHTSEARCH_PEER_RELIABILITY_STALENESS_HORIZON"
	EnvSnapshotInterval                 = "YACYDHTSEARCH_PEER_PRESENCE_SNAPSHOT_INTERVAL"
	EnvPartitionExponent                = "YACYDHTSEARCH_PARTITION_EXPONENT"
	EnvMaxResponseBytes                 = "YACYDHTSEARCH_MAX_RESPONSE_BYTES"
	EnvPeerItemsCeiling                 = "YACYDHTSEARCH_PEER_ITEMS_CEILING"
	EnvDocumentsToMatchCeiling          = "YACYDHTSEARCH_DOCUMENTS_TO_MATCH_CEILING"
	EnvURLMetadataAskDocumentsCeiling   = "YACYDHTSEARCH_URL_METADATA_ASK_DOCUMENTS_CEILING"
	EnvURLMetadataAskDocumentsFloor     = "YACYDHTSEARCH_URL_METADATA_ASK_DOCUMENTS_FLOOR"
	EnvURLMetadataAskTargetTime         = "YACYDHTSEARCH_URL_METADATA_ASK_TARGET_TIME"
	EnvRankedItemsCeiling               = "YACYDHTSEARCH_RANKED_ITEMS_CEILING"
	EnvNATSURL                          = "YACYDHTSEARCH_NATS_URL"
	EnvRankingCacheCapacity             = "YACYDHTSEARCH_RANKING_CACHE_CAPACITY"
	EnvRankingLifetime                  = "YACYDHTSEARCH_RANKING_LIFETIME"
	EnvQueryWordDocumentAmountLifetime  = "YACYDHTSEARCH_QUERY_WORD_DOCUMENT_AMOUNT_LIFETIME"
	EnvQueryWordDocumentAmountsCapacity = "YACYDHTSEARCH_QUERY_WORD_DOCUMENT_AMOUNTS_CAPACITY"
	EnvPagesReadPerQuery                = "YACYDHTSEARCH_PAGES_READ_PER_QUERY"
	EnvPagesReadPerSite                 = "YACYDHTSEARCH_PAGES_READ_PER_SITE"
	EnvCompoundWordsCeiling             = "YACYDHTSEARCH_COMPOUND_WORDS_CEILING"
	EnvPageReadBudget                   = "YACYDHTSEARCH_PAGE_READ_BUDGET"
	EnvPageReadCutoffPercent            = "YACYDHTSEARCH_PAGE_READ_CUTOFF_PERCENT"
	EnvPageReadCutoffGrace              = "YACYDHTSEARCH_PAGE_READ_CUTOFF_GRACE"
	EnvURLMetadataLookupCutoffPercent   = "YACYDHTSEARCH_URL_METADATA_LOOKUP_CUTOFF_PERCENT"
	EnvURLMetadataLookupCutoffGrace     = "YACYDHTSEARCH_URL_METADATA_LOOKUP_CUTOFF_GRACE"
	EnvPageByteCeiling                  = "YACYDHTSEARCH_PAGE_BYTE_CEILING"
	EnvPageReadMaxRedirectHops          = "YACYDHTSEARCH_PAGE_READ_MAX_REDIRECT_HOPS"
	EnvSnippetLengthCeiling             = "YACYDHTSEARCH_SNIPPET_LENGTH_CEILING"

	DefaultListenAddr                       = ":8080"
	DefaultOpsAddr                          = ":9090"
	DefaultServeProfiler                    = false
	DefaultQueryBudget                      = 10 * time.Second
	DefaultNetworkRedundancy                = 3
	DefaultHedgeDelay                       = 500 * time.Millisecond
	DefaultReplicasCoveringAPartition       = 1
	DefaultPeerCallsInFlight                = 160
	DefaultURLMetadataCallBudget            = 3 * time.Second
	DefaultSearchCallBudget                 = 2 * time.Second
	DefaultProbesInFlight                   = 24
	DefaultDirectoryCapacity                = 4096
	DefaultRefreshInterval                  = 5 * time.Minute
	DefaultSeedlistReadBudget               = 10 * time.Second
	DefaultNewcomerShare                    = 0.05
	DefaultProbeBudget                      = 3 * time.Second
	DefaultContinuityLimit                  = 15 * time.Minute
	DefaultProbeAnswerHistoryKeptFor        = 24 * time.Hour
	DefaultSnapshotInterval                 = 10 * time.Minute
	DefaultPartitionExponent                = 4
	DefaultMaxResponseBytes                 = 4 * 1024 * 1024
	DefaultPeerItemsCeiling                 = 10
	DefaultDocumentsToMatchCeiling          = 1000
	DefaultURLMetadataAskDocumentsCeiling   = 1000
	DefaultURLMetadataAskDocumentsFloor     = 25
	DefaultURLMetadataAskTargetTime         = time.Second
	DefaultRankedItemsCeiling               = 50
	DefaultCompoundWordsCeiling             = 4
	DefaultRankingCacheCapacity             = 1024
	DefaultRankingLifetime                  = 2 * time.Minute
	DefaultQueryWordDocumentAmountLifetime  = 6 * time.Hour
	DefaultQueryWordDocumentAmountsCapacity = 100000
	DefaultPagesReadPerQuery                = 50
	DefaultPagesReadPerSite                 = 3
	DefaultPageReadBudget                   = 3 * time.Second
	DefaultPageReadCutoffPercent            = 90
	DefaultPageReadCutoffGrace              = 250 * time.Millisecond
	DefaultURLMetadataLookupCutoffPercent   = 90
	DefaultURLMetadataLookupCutoffGrace     = 250 * time.Millisecond
	DefaultPageByteCeiling                  = 4 * 1024 * 1024
	DefaultPageReadMaxRedirectHops          = 3
	DefaultSnippetLengthCeiling             = 300
	DefaultPageReadProxyDialMode            = "tunnel"
)

type ServiceConfig struct {
	ListenAddr                       string
	OpsAddr                          string
	ServeProfiler                    bool
	NetworkName                      string
	SeedlistURLs                     []string
	SeedlistReadBudget               time.Duration
	EgressProxyURL                   *url.URL
	PageReadProxyURL                 *url.URL
	PageReadProxyDialMode            pagefetchershttp.ProxyDialMode
	QueryBudget                      time.Duration
	NetworkRedundancy                int
	ReplicasCoveringAPartition       int
	HedgeDelay                       time.Duration
	PeerCallsInFlight                int
	URLMetadataCallBudget            time.Duration
	SearchCallBudget                 time.Duration
	ProbesInFlight                   int
	DirectoryCapacity                int
	NewcomerShare                    float64
	RefreshInterval                  time.Duration
	ProbeBudget                      time.Duration
	ContinuityLimit                  time.Duration
	ProbeAnswerHistoryKeptFor        time.Duration
	MaturationDuration               time.Duration
	StalenessHorizon                 time.Duration
	SnapshotInterval                 time.Duration
	Partitions                       yacymodel.DHTRingPartitions
	MaxResponseBytes                 int64
	PeerItemsCeiling                 int
	DocumentsToMatchCeiling          int
	URLMetadataAskDocumentsCeiling   int
	URLMetadataAskDocumentsFloor     int
	URLMetadataAskTargetTime         time.Duration
	URLMetadataLookupCutoff          wordjoined.URLMetadataLookupCutoff
	RankedItemsCeiling               int
	NATSURL                          string
	RankingCache                     int
	RankingLifetime                  time.Duration
	QueryWordDocumentAmountLifetime  time.Duration
	QueryWordDocumentAmountsCapacity int

	PagesReadPerQuery       int
	PagesReadPerSite        int
	CompoundWordsCeiling    int
	PageReadBudget          time.Duration
	PageReadCutoff          pagereading.PageReadCutoff
	PageByteCeiling         int64
	PageReadMaxRedirectHops int
	SnippetLengthCeiling    int
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	seedlistURLs, err := seedlistURLsOf(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	egressProxyURL, err := requiredProxyURL(getenv, EnvEgressProxyURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageReadProxyURL, err := pageReadProxyURLOf(getenv, egressProxyURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageReadProxyDialMode, err := pageReadProxyDialModeOf(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	partitions, err := partitionsOf(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	durations, err := durationsOf(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	counts, err := countsOf(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	replicasCoveringAPartition, err := replicasCoveringAPartitionOf(
		getenv,
		counts.networkRedundancy,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	newcomerShare, err := envconfig.Share(getenv, EnvNewcomerShare, DefaultNewcomerShare)
	if err != nil {
		return ServiceConfig{}, err
	}
	maxResponseBytes, err := envconfig.PositiveInt64(
		getenv, EnvMaxResponseBytes, DefaultMaxResponseBytes,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageByteCeiling, err := envconfig.PositiveInt64(
		getenv, EnvPageByteCeiling, DefaultPageByteCeiling,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	serveProfiler, err := envconfig.Bool(getenv, EnvServeProfiler, DefaultServeProfiler)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		ListenAddr:    envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:       envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		ServeProfiler: serveProfiler,
		NetworkName: envconfig.String(
			getenv,
			EnvNetworkName,
			yacyproto.DefaultNetwork,
		),
		SeedlistURLs:                   seedlistURLs,
		SeedlistReadBudget:             durations.seedlistReadBudget,
		EgressProxyURL:                 egressProxyURL,
		PageReadProxyURL:               pageReadProxyURL,
		PageReadProxyDialMode:          pageReadProxyDialMode,
		QueryBudget:                    durations.queryBudget,
		NetworkRedundancy:              counts.networkRedundancy,
		ReplicasCoveringAPartition:     replicasCoveringAPartition,
		HedgeDelay:                     durations.hedgeDelay,
		PeerCallsInFlight:              counts.peerCallsInFlight,
		URLMetadataCallBudget:          durations.urlMetadataCallBudget,
		SearchCallBudget:               durations.searchCallBudget,
		ProbesInFlight:                 counts.probesInFlight,
		DirectoryCapacity:              counts.directoryCapacity,
		NewcomerShare:                  newcomerShare,
		RefreshInterval:                durations.refreshInterval,
		ProbeBudget:                    durations.probeBudget,
		ContinuityLimit:                durations.continuityLimit,
		ProbeAnswerHistoryKeptFor:      durations.probeAnswerHistoryKeptFor,
		MaturationDuration:             durations.maturationDuration,
		StalenessHorizon:               durations.stalenessHorizon,
		SnapshotInterval:               durations.snapshotInterval,
		Partitions:                     partitions,
		MaxResponseBytes:               maxResponseBytes,
		PeerItemsCeiling:               counts.peerItemsCeiling,
		DocumentsToMatchCeiling:        counts.documentsToMatchCeiling,
		URLMetadataAskDocumentsCeiling: counts.urlMetadataAskDocumentsCeiling,
		URLMetadataAskDocumentsFloor:   counts.urlMetadataAskDocumentsFloor,
		URLMetadataAskTargetTime:       durations.urlMetadataAskTargetTime,
		URLMetadataLookupCutoff: wordjoined.URLMetadataLookupCutoff{
			PercentOfDocuments: counts.urlMetadataLookupCutoffPercent,
			Grace:              durations.urlMetadataLookupCutoffGrace,
		},
		RankedItemsCeiling:               counts.rankedItemsCeiling,
		NATSURL:                          strings.TrimSpace(getenv(EnvNATSURL)),
		RankingCache:                     counts.rankingCacheCapacity,
		RankingLifetime:                  durations.rankingLifetime,
		QueryWordDocumentAmountLifetime:  durations.queryWordDocumentAmountLifetime,
		QueryWordDocumentAmountsCapacity: counts.queryWordDocumentAmountsCapacity,

		PagesReadPerQuery:    counts.pagesReadPerQuery,
		PagesReadPerSite:     counts.pagesReadPerSite,
		CompoundWordsCeiling: counts.compoundWordsCeiling,
		PageReadBudget:       durations.pageReadBudget,
		PageReadCutoff: pagereading.PageReadCutoff{
			PercentOfPages: counts.pageReadCutoffPercent,
			Grace:          durations.pageReadCutoffGrace,
		},
		PageByteCeiling:         pageByteCeiling,
		PageReadMaxRedirectHops: counts.pageReadMaxRedirectHops,
		SnippetLengthCeiling:    counts.snippetLengthCeiling,
	}, nil
}

type configuredDurations struct {
	queryBudget                     time.Duration
	hedgeDelay                      time.Duration
	urlMetadataCallBudget           time.Duration
	searchCallBudget                time.Duration
	refreshInterval                 time.Duration
	seedlistReadBudget              time.Duration
	probeBudget                     time.Duration
	continuityLimit                 time.Duration
	probeAnswerHistoryKeptFor       time.Duration
	maturationDuration              time.Duration
	stalenessHorizon                time.Duration
	snapshotInterval                time.Duration
	rankingLifetime                 time.Duration
	queryWordDocumentAmountLifetime time.Duration
	pageReadBudget                  time.Duration
	pageReadCutoffGrace             time.Duration
	urlMetadataLookupCutoffGrace    time.Duration
	urlMetadataAskTargetTime        time.Duration
}

func durationsOf(getenv func(string) string) (configuredDurations, error) {
	var durations configuredDurations
	var err error
	for _, field := range []struct {
		key      string
		fallback time.Duration
		into     *time.Duration
	}{
		{EnvQueryBudget, DefaultQueryBudget, &durations.queryBudget},
		{EnvHedgeDelay, DefaultHedgeDelay, &durations.hedgeDelay},
		{EnvURLMetadataCallBudget, DefaultURLMetadataCallBudget, &durations.urlMetadataCallBudget},
		{EnvSearchCallBudget, DefaultSearchCallBudget, &durations.searchCallBudget},
		{EnvRefreshInterval, DefaultRefreshInterval, &durations.refreshInterval},
		{EnvSeedlistReadBudget, DefaultSeedlistReadBudget, &durations.seedlistReadBudget},
		{EnvProbeBudget, DefaultProbeBudget, &durations.probeBudget},
		{EnvContinuityLimit, DefaultContinuityLimit, &durations.continuityLimit},
		{EnvProbeAnswerHistoryKeptFor, DefaultProbeAnswerHistoryKeptFor, &durations.probeAnswerHistoryKeptFor},
		{
			EnvMaturationDuration,
			peerreliability.DefaultReliabilityWeights().MaturationDuration,
			&durations.maturationDuration,
		},
		{
			EnvStalenessHorizon,
			peerreliability.DefaultReliabilityWeights().StalenessHorizon,
			&durations.stalenessHorizon,
		},
		{EnvSnapshotInterval, DefaultSnapshotInterval, &durations.snapshotInterval},
		{EnvRankingLifetime, DefaultRankingLifetime, &durations.rankingLifetime},
		{EnvQueryWordDocumentAmountLifetime, DefaultQueryWordDocumentAmountLifetime, &durations.queryWordDocumentAmountLifetime},
		{EnvPageReadBudget, DefaultPageReadBudget, &durations.pageReadBudget},
		{EnvPageReadCutoffGrace, DefaultPageReadCutoffGrace, &durations.pageReadCutoffGrace},
		{
			EnvURLMetadataLookupCutoffGrace,
			DefaultURLMetadataLookupCutoffGrace,
			&durations.urlMetadataLookupCutoffGrace,
		},
		{EnvURLMetadataAskTargetTime, DefaultURLMetadataAskTargetTime, &durations.urlMetadataAskTargetTime},
	} {
		if *field.into, err = envconfig.Duration(getenv, field.key, field.fallback); err != nil {
			return configuredDurations{}, err
		}
	}

	return durations, nil
}

type configuredCounts struct {
	networkRedundancy                int
	peerCallsInFlight                int
	probesInFlight                   int
	directoryCapacity                int
	peerItemsCeiling                 int
	documentsToMatchCeiling          int
	urlMetadataAskDocumentsCeiling   int
	urlMetadataAskDocumentsFloor     int
	rankedItemsCeiling               int
	rankingCacheCapacity             int
	queryWordDocumentAmountsCapacity int
	pagesReadPerQuery                int
	pagesReadPerSite                 int
	compoundWordsCeiling             int
	pageReadMaxRedirectHops          int
	pageReadCutoffPercent            int
	urlMetadataLookupCutoffPercent   int
	snippetLengthCeiling             int
}

func countsOf(getenv func(string) string) (configuredCounts, error) {
	var counts configuredCounts
	var err error
	for _, field := range []struct {
		key      string
		fallback int
		into     *int
	}{
		{EnvNetworkRedundancy, DefaultNetworkRedundancy, &counts.networkRedundancy},
		{EnvPeerCallsInFlight, DefaultPeerCallsInFlight, &counts.peerCallsInFlight},
		{EnvProbesInFlight, DefaultProbesInFlight, &counts.probesInFlight},
		{EnvDirectoryCapacity, DefaultDirectoryCapacity, &counts.directoryCapacity},
		{EnvPeerItemsCeiling, DefaultPeerItemsCeiling, &counts.peerItemsCeiling},
		{EnvDocumentsToMatchCeiling, DefaultDocumentsToMatchCeiling, &counts.documentsToMatchCeiling},
		{EnvURLMetadataAskDocumentsCeiling, DefaultURLMetadataAskDocumentsCeiling, &counts.urlMetadataAskDocumentsCeiling},
		{EnvURLMetadataAskDocumentsFloor, DefaultURLMetadataAskDocumentsFloor, &counts.urlMetadataAskDocumentsFloor},
		{EnvRankedItemsCeiling, DefaultRankedItemsCeiling, &counts.rankedItemsCeiling},
		{EnvRankingCacheCapacity, DefaultRankingCacheCapacity, &counts.rankingCacheCapacity},
		{EnvQueryWordDocumentAmountsCapacity, DefaultQueryWordDocumentAmountsCapacity, &counts.queryWordDocumentAmountsCapacity},
		{EnvPagesReadPerQuery, DefaultPagesReadPerQuery, &counts.pagesReadPerQuery},
		{EnvPagesReadPerSite, DefaultPagesReadPerSite, &counts.pagesReadPerSite},
		{EnvCompoundWordsCeiling, DefaultCompoundWordsCeiling, &counts.compoundWordsCeiling},
		{EnvPageReadMaxRedirectHops, DefaultPageReadMaxRedirectHops, &counts.pageReadMaxRedirectHops},
		{EnvPageReadCutoffPercent, DefaultPageReadCutoffPercent, &counts.pageReadCutoffPercent},
		{
			EnvURLMetadataLookupCutoffPercent,
			DefaultURLMetadataLookupCutoffPercent,
			&counts.urlMetadataLookupCutoffPercent,
		},
		{EnvSnippetLengthCeiling, DefaultSnippetLengthCeiling, &counts.snippetLengthCeiling},
	} {
		if *field.into, err = envconfig.PositiveInt(getenv, field.key, field.fallback); err != nil {
			return configuredCounts{}, err
		}
	}

	return counts, nil
}

func replicasCoveringAPartitionOf(
	getenv func(string) string,
	networkRedundancy int,
) (int, error) {
	replicas, err := envconfig.PositiveInt(
		getenv,
		EnvReplicasCoveringAPartition,
		DefaultReplicasCoveringAPartition,
	)
	if err != nil {
		return 0, err
	}
	if replicas > networkRedundancy {
		return 0, fmt.Errorf(
			"%s: must not be above %s",
			EnvReplicasCoveringAPartition,
			EnvNetworkRedundancy,
		)
	}

	return replicas, nil
}

func partitionsOf(getenv func(string) string) (yacymodel.DHTRingPartitions, error) {
	exponent, err := envconfig.PositiveInt(getenv, EnvPartitionExponent, DefaultPartitionExponent)
	if err != nil {
		return 0, err
	}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(uint(exponent))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", EnvPartitionExponent, err)
	}

	return partitions, nil
}

func seedlistURLsOf(getenv func(string) string) ([]string, error) {
	raw := strings.TrimSpace(getenv(EnvSeedlistURLs))
	if raw == "" {
		return nil, fmt.Errorf("%s: must be set", EnvSeedlistURLs)
	}

	var addresses []string
	for _, candidate := range strings.Split(raw, ",") {
		address := strings.TrimSpace(candidate)
		if address == "" {
			continue
		}
		if _, err := url.Parse(address); err != nil {
			return nil, fmt.Errorf("%s: %w", EnvSeedlistURLs, err)
		}
		addresses = append(addresses, address)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("%s: must name at least one seedlist", EnvSeedlistURLs)
	}

	return addresses, nil
}

func pageReadProxyURLOf(
	getenv func(string) string,
	egressProxyURL *url.URL,
) (*url.URL, error) {
	if strings.TrimSpace(getenv(EnvPageReadProxyURL)) == "" {
		return egressProxyURL, nil
	}

	return requiredProxyURL(getenv, EnvPageReadProxyURL)
}

func pageReadProxyDialModeOf(
	getenv func(string) string,
) (pagefetchershttp.ProxyDialMode, error) {
	mode, err := pagefetchershttp.ProxyDialModeNamed(
		envconfig.String(getenv, EnvPageReadProxyDialMode, DefaultPageReadProxyDialMode),
	)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", EnvPageReadProxyDialMode, err)
	}

	return mode, nil
}

func requiredProxyURL(getenv func(string) string, key string) (*url.URL, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return nil, fmt.Errorf("%s: must be set", key)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s: scheme must be http or https", key)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("%s: must include a host", key)
	}

	return parsed, nil
}
