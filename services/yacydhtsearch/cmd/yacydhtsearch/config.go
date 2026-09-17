package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvListenAddr                   = "YACYDHTSEARCH_LISTEN_ADDR"
	EnvOpsAddr                      = "YACYDHTSEARCH_OPS_ADDR"
	EnvNetworkName                  = "YACYDHTSEARCH_NETWORK_NAME"
	EnvSeedlistURLs                 = "YACYDHTSEARCH_SEEDLIST_URLS"
	EnvEgressProxyURL               = "EGRESS_PROXY_URL"
	EnvQueryBudget                  = "YACYDHTSEARCH_QUERY_BUDGET"
	EnvNetworkRedundancy            = "YACYDHTSEARCH_NETWORK_REDUNDANCY"
	EnvReplicasCoveringAPartition   = "YACYDHTSEARCH_REPLICAS_COVERING_A_PARTITION"
	EnvHedgeDelay                   = "YACYDHTSEARCH_HEDGE_DELAY"
	EnvPeerCallsInFlight            = "YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT"
	EnvPeerCallBudget               = "YACYDHTSEARCH_PEER_CALL_BUDGET"
	EnvProbesInFlight               = "YACYDHTSEARCH_PROBES_IN_FLIGHT"
	EnvDirectoryCapacity            = "YACYDHTSEARCH_DIRECTORY_CAPACITY"
	EnvRefreshInterval              = "YACYDHTSEARCH_REFRESH_INTERVAL"
	EnvNewcomerShare                = "YACYDHTSEARCH_DIRECTORY_NEWCOMER_SHARE"
	EnvProbeBudget                  = "YACYDHTSEARCH_PROBE_BUDGET"
	EnvContinuityLimit              = "YACYDHTSEARCH_PEER_PRESENCE_CONTINUITY_LIMIT"
	EnvProbeAnswerHistoryKeptFor    = "YACYDHTSEARCH_PROBE_ANSWER_HISTORY_KEPT_FOR"
	EnvMaturationDuration           = "YACYDHTSEARCH_PEER_RELIABILITY_MATURATION_DURATION"
	EnvStalenessHorizon             = "YACYDHTSEARCH_PEER_RELIABILITY_STALENESS_HORIZON"
	EnvSnapshotInterval             = "YACYDHTSEARCH_PEER_PRESENCE_SNAPSHOT_INTERVAL"
	EnvPartitionExponent            = "YACYDHTSEARCH_PARTITION_EXPONENT"
	EnvMaxResponseBytes             = "YACYDHTSEARCH_MAX_RESPONSE_BYTES"
	EnvPeerItemsCeiling             = "YACYDHTSEARCH_PEER_ITEMS_CEILING"
	EnvCrossCheckedDocumentsCeiling = "YACYDHTSEARCH_CROSS_CHECKED_DOCUMENTS_CEILING"
	EnvAsksForCrossCheckedDocuments = "YACYDHTSEARCH_ASKS_FOR_CROSS_CHECKED_DOCUMENTS"
	EnvRankedItemsCeiling           = "YACYDHTSEARCH_RANKED_ITEMS_CEILING"
	EnvNATSURL                      = "YACYDHTSEARCH_NATS_URL"
	EnvRankingCacheCapacity         = "YACYDHTSEARCH_RANKING_CACHE_CAPACITY"
	EnvRankingLifetime              = "YACYDHTSEARCH_RANKING_LIFETIME"
	EnvPagesReadPerQuery            = "YACYDHTSEARCH_PAGES_READ_PER_QUERY"
	EnvPageReadBudget               = "YACYDHTSEARCH_PAGE_READ_BUDGET"
	EnvPageByteCeiling              = "YACYDHTSEARCH_PAGE_BYTE_CEILING"
	EnvSnippetLengthCeiling         = "YACYDHTSEARCH_SNIPPET_LENGTH_CEILING"

	DefaultListenAddr                   = ":8080"
	DefaultOpsAddr                      = ":9090"
	DefaultQueryBudget                  = 10 * time.Second
	DefaultNetworkRedundancy            = 3
	DefaultHedgeDelay                   = 500 * time.Millisecond
	DefaultPeerCallsInFlight            = 48
	DefaultPeerCallBudget               = 3 * time.Second
	DefaultProbesInFlight               = 24
	DefaultDirectoryCapacity            = 4096
	DefaultRefreshInterval              = 5 * time.Minute
	DefaultNewcomerShare                = 0.05
	DefaultProbeBudget                  = 3 * time.Second
	DefaultContinuityLimit              = 15 * time.Minute
	DefaultProbeAnswerHistoryKeptFor    = 24 * time.Hour
	DefaultSnapshotInterval             = 10 * time.Minute
	DefaultPartitionExponent            = 4
	DefaultMaxResponseBytes             = 4 * 1024 * 1024
	DefaultPeerItemsCeiling             = 10
	DefaultCrossCheckedDocumentsCeiling = 1000
	DefaultAsksForCrossCheckedDocuments = false
	DefaultRankedItemsCeiling           = 50
	DefaultRankingCacheCapacity         = 1024
	DefaultRankingLifetime              = 2 * time.Minute
	DefaultPagesReadPerQuery            = 50
	DefaultPageReadBudget               = 3 * time.Second
	DefaultPageByteCeiling              = 4 * 1024 * 1024
	DefaultSnippetLengthCeiling         = 300
)

type ServiceConfig struct {
	ListenAddr                   string
	OpsAddr                      string
	NetworkName                  string
	SeedlistURLs                 []string
	EgressProxyURL               *url.URL
	QueryBudget                  time.Duration
	NetworkRedundancy            int
	ReplicasCoveringAPartition   int
	HedgeDelay                   time.Duration
	PeerCallsInFlight            int
	PeerCallBudget               time.Duration
	ProbesInFlight               int
	DirectoryCapacity            int
	NewcomerShare                float64
	RefreshInterval              time.Duration
	ProbeBudget                  time.Duration
	ContinuityLimit              time.Duration
	ProbeAnswerHistoryKeptFor    time.Duration
	MaturationDuration           time.Duration
	StalenessHorizon             time.Duration
	SnapshotInterval             time.Duration
	Partitions                   yacymodel.DHTRingPartitions
	MaxResponseBytes             int64
	PeerItemsCeiling             int
	CrossCheckedDocumentsCeiling int
	AsksForCrossCheckedDocuments bool
	RankedItemsCeiling           int
	NATSURL                      string
	RankingCache                 int
	RankingLifetime              time.Duration

	PagesReadPerQuery    int
	PageReadBudget       time.Duration
	PageByteCeiling      int64
	SnippetLengthCeiling int
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
	asksForCrossCheckedDocuments, err := envconfig.Bool(
		getenv, EnvAsksForCrossCheckedDocuments, DefaultAsksForCrossCheckedDocuments,
	)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		ListenAddr: envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:    envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		NetworkName: envconfig.String(
			getenv,
			EnvNetworkName,
			yacyproto.DefaultNetwork,
		),
		SeedlistURLs:                 seedlistURLs,
		EgressProxyURL:               egressProxyURL,
		QueryBudget:                  durations.queryBudget,
		NetworkRedundancy:            counts.networkRedundancy,
		ReplicasCoveringAPartition:   replicasCoveringAPartition,
		HedgeDelay:                   durations.hedgeDelay,
		PeerCallsInFlight:            counts.peerCallsInFlight,
		PeerCallBudget:               durations.peerCallBudget,
		ProbesInFlight:               counts.probesInFlight,
		DirectoryCapacity:            counts.directoryCapacity,
		NewcomerShare:                newcomerShare,
		RefreshInterval:              durations.refreshInterval,
		ProbeBudget:                  durations.probeBudget,
		ContinuityLimit:              durations.continuityLimit,
		ProbeAnswerHistoryKeptFor:    durations.probeAnswerHistoryKeptFor,
		MaturationDuration:           durations.maturationDuration,
		StalenessHorizon:             durations.stalenessHorizon,
		SnapshotInterval:             durations.snapshotInterval,
		Partitions:                   partitions,
		MaxResponseBytes:             maxResponseBytes,
		PeerItemsCeiling:             counts.peerItemsCeiling,
		CrossCheckedDocumentsCeiling: counts.crossCheckedDocumentsCeiling,
		AsksForCrossCheckedDocuments: asksForCrossCheckedDocuments,
		RankedItemsCeiling:           counts.rankedItemsCeiling,
		NATSURL:                      strings.TrimSpace(getenv(EnvNATSURL)),
		RankingCache:                 counts.rankingCacheCapacity,
		RankingLifetime:              durations.rankingLifetime,

		PagesReadPerQuery:    counts.pagesReadPerQuery,
		PageReadBudget:       durations.pageReadBudget,
		PageByteCeiling:      pageByteCeiling,
		SnippetLengthCeiling: counts.snippetLengthCeiling,
	}, nil
}

type configuredDurations struct {
	queryBudget               time.Duration
	hedgeDelay                time.Duration
	peerCallBudget            time.Duration
	refreshInterval           time.Duration
	probeBudget               time.Duration
	continuityLimit           time.Duration
	probeAnswerHistoryKeptFor time.Duration
	maturationDuration        time.Duration
	stalenessHorizon          time.Duration
	snapshotInterval          time.Duration
	rankingLifetime           time.Duration
	pageReadBudget            time.Duration
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
		{EnvPeerCallBudget, DefaultPeerCallBudget, &durations.peerCallBudget},
		{EnvRefreshInterval, DefaultRefreshInterval, &durations.refreshInterval},
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
		{EnvPageReadBudget, DefaultPageReadBudget, &durations.pageReadBudget},
	} {
		if *field.into, err = envconfig.Duration(getenv, field.key, field.fallback); err != nil {
			return configuredDurations{}, err
		}
	}

	return durations, nil
}

type configuredCounts struct {
	networkRedundancy            int
	peerCallsInFlight            int
	probesInFlight               int
	directoryCapacity            int
	peerItemsCeiling             int
	crossCheckedDocumentsCeiling int
	rankedItemsCeiling           int
	rankingCacheCapacity         int
	pagesReadPerQuery            int
	snippetLengthCeiling         int
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
		{EnvCrossCheckedDocumentsCeiling, DefaultCrossCheckedDocumentsCeiling, &counts.crossCheckedDocumentsCeiling},
		{EnvRankedItemsCeiling, DefaultRankedItemsCeiling, &counts.rankedItemsCeiling},
		{EnvRankingCacheCapacity, DefaultRankingCacheCapacity, &counts.rankingCacheCapacity},
		{EnvPagesReadPerQuery, DefaultPagesReadPerQuery, &counts.pagesReadPerQuery},
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
	replicas, err := envconfig.PositiveInt(getenv, EnvReplicasCoveringAPartition, networkRedundancy)
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
