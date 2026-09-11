package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvListenAddr           = "YACYDHTSEARCH_LISTEN_ADDR"
	EnvOpsAddr              = "YACYDHTSEARCH_OPS_ADDR"
	EnvNetworkName          = "YACYDHTSEARCH_NETWORK_NAME"
	EnvSeedlistURLs         = "YACYDHTSEARCH_SEEDLIST_URLS"
	EnvEgressProxyURL       = "EGRESS_PROXY_URL"
	EnvQueryBudget          = "YACYDHTSEARCH_QUERY_BUDGET"
	EnvPeerChoiceCooldown   = "YACYDHTSEARCH_PEER_CHOICE_COOLDOWN"
	EnvNetworkRedundancy    = "YACYDHTSEARCH_NETWORK_REDUNDANCY"
	EnvPeerCallsInFlight    = "YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT"
	EnvProbesInFlight       = "YACYDHTSEARCH_PROBES_IN_FLIGHT"
	EnvDirectoryCapacity    = "YACYDHTSEARCH_DIRECTORY_CAPACITY"
	EnvRefreshInterval      = "YACYDHTSEARCH_REFRESH_INTERVAL"
	EnvProbeBudget          = "YACYDHTSEARCH_PROBE_BUDGET"
	EnvPartitionExponent    = "YACYDHTSEARCH_PARTITION_EXPONENT"
	EnvMaxResponseBytes     = "YACYDHTSEARCH_MAX_RESPONSE_BYTES"
	EnvPeerItemsCeiling     = "YACYDHTSEARCH_PEER_ITEMS_CEILING"
	EnvRankedItemsCeiling   = "YACYDHTSEARCH_RANKED_ITEMS_CEILING"
	EnvNATSURL              = "YACYDHTSEARCH_NATS_URL"
	EnvRankingCacheCapacity = "YACYDHTSEARCH_RANKING_CACHE_CAPACITY"
	EnvRankingLifetime      = "YACYDHTSEARCH_RANKING_LIFETIME"
	EnvWordJoinedSearch     = "YACYDHTSEARCH_WORD_JOINED_SEARCH"
	EnvRelevanceRanking     = "YACYDHTSEARCH_RELEVANCE_RANKING"
	EnvPagesReadPerQuery    = "YACYDHTSEARCH_PAGES_READ_PER_QUERY"
	EnvPageReadBudget       = "YACYDHTSEARCH_PAGE_READ_BUDGET"
	EnvPageByteCeiling      = "YACYDHTSEARCH_PAGE_BYTE_CEILING"
	EnvSnippetLengthCeiling = "YACYDHTSEARCH_SNIPPET_LENGTH_CEILING"

	DefaultListenAddr           = ":8080"
	DefaultOpsAddr              = ":9090"
	DefaultQueryBudget          = 8 * time.Second
	DefaultPeerChoiceCooldown   = 5 * time.Second
	DefaultNetworkRedundancy    = 3
	DefaultPeerCallsInFlight    = 48
	DefaultProbesInFlight       = 24
	DefaultDirectoryCapacity    = 4096
	DefaultRefreshInterval      = 5 * time.Minute
	DefaultProbeBudget          = 3 * time.Second
	DefaultPartitionExponent    = 4
	DefaultMaxResponseBytes     = 4 * 1024 * 1024
	DefaultPeerItemsCeiling     = 10
	DefaultRankedItemsCeiling   = 50
	DefaultRankingCacheCapacity = 1024
	DefaultRankingLifetime      = 2 * time.Minute
	DefaultPagesReadPerQuery    = 50
	DefaultPageReadBudget       = 3 * time.Second
	DefaultPageByteCeiling      = 4 * 1024 * 1024
	DefaultSnippetLengthCeiling = 300
)

type ServiceConfig struct {
	ListenAddr         string
	OpsAddr            string
	NetworkName        string
	SeedlistURLs       []string
	EgressProxyURL     *url.URL
	QueryBudget        time.Duration
	PeerChoiceCooldown time.Duration
	NetworkRedundancy  int
	PeerCallsInFlight  int
	ProbesInFlight     int
	DirectoryCapacity  int
	RefreshInterval    time.Duration
	ProbeBudget        time.Duration
	Partitions         yacymodel.DHTRingPartitions
	MaxResponseBytes   int64
	PeerItemsCeiling   int
	RankedItemsCeiling int
	NATSURL            string
	RankingCache       int
	RankingLifetime    time.Duration
	WordJoinedSearch   bool
	RelevanceRanking   bool

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
	wordJoinedSearch, err := envconfig.Bool(getenv, EnvWordJoinedSearch, false)
	if err != nil {
		return ServiceConfig{}, err
	}
	relevanceRanking, err := envconfig.Bool(getenv, EnvRelevanceRanking, false)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		ListenAddr:         envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:            envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		NetworkName:        envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork),
		SeedlistURLs:       seedlistURLs,
		EgressProxyURL:     egressProxyURL,
		QueryBudget:        durations.queryBudget,
		PeerChoiceCooldown: durations.peerChoiceCooldown,
		NetworkRedundancy:  counts.networkRedundancy,
		PeerCallsInFlight:  counts.peerCallsInFlight,
		ProbesInFlight:     counts.probesInFlight,
		DirectoryCapacity:  counts.directoryCapacity,
		RefreshInterval:    durations.refreshInterval,
		ProbeBudget:        durations.probeBudget,
		Partitions:         partitions,
		MaxResponseBytes:   maxResponseBytes,
		PeerItemsCeiling:   counts.peerItemsCeiling,
		RankedItemsCeiling: counts.rankedItemsCeiling,
		NATSURL:            strings.TrimSpace(getenv(EnvNATSURL)),
		RankingCache:       counts.rankingCacheCapacity,
		RankingLifetime:    durations.rankingLifetime,
		WordJoinedSearch:   wordJoinedSearch,
		RelevanceRanking:   relevanceRanking,

		PagesReadPerQuery:    counts.pagesReadPerQuery,
		PageReadBudget:       durations.pageReadBudget,
		PageByteCeiling:      pageByteCeiling,
		SnippetLengthCeiling: counts.snippetLengthCeiling,
	}, nil
}

type configuredDurations struct {
	queryBudget        time.Duration
	peerChoiceCooldown time.Duration
	refreshInterval    time.Duration
	probeBudget        time.Duration
	rankingLifetime    time.Duration
	pageReadBudget     time.Duration
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
		{EnvPeerChoiceCooldown, DefaultPeerChoiceCooldown, &durations.peerChoiceCooldown},
		{EnvRefreshInterval, DefaultRefreshInterval, &durations.refreshInterval},
		{EnvProbeBudget, DefaultProbeBudget, &durations.probeBudget},
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
	networkRedundancy    int
	peerCallsInFlight    int
	probesInFlight       int
	directoryCapacity    int
	peerItemsCeiling     int
	rankedItemsCeiling   int
	rankingCacheCapacity int
	pagesReadPerQuery    int
	snippetLengthCeiling int
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
