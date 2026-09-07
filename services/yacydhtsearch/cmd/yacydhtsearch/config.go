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
	EnvPeerCallBudget       = "YACYDHTSEARCH_PEER_CALL_BUDGET"
	EnvPeerSearchCooldown   = "YACYDHTSEARCH_PEER_SEARCH_COOLDOWN"
	EnvPeerCallsInFlight    = "YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT"
	EnvDirectoryCapacity    = "YACYDHTSEARCH_DIRECTORY_CAPACITY"
	EnvRefreshInterval      = "YACYDHTSEARCH_REFRESH_INTERVAL"
	EnvProbeBudget          = "YACYDHTSEARCH_PROBE_BUDGET"
	EnvPartitionExponent    = "YACYDHTSEARCH_PARTITION_EXPONENT"
	EnvPeerRedundancy       = "YACYDHTSEARCH_PEER_REDUNDANCY"
	EnvMaxResponseBytes     = "YACYDHTSEARCH_MAX_RESPONSE_BYTES"
	EnvPeerItemsCeiling     = "YACYDHTSEARCH_PEER_ITEMS_CEILING"
	EnvRankedItemsCeiling   = "YACYDHTSEARCH_RANKED_ITEMS_CEILING"
	EnvNATSURL              = "YACYDHTSEARCH_NATS_URL"
	EnvRankingCacheCapacity = "YACYDHTSEARCH_RANKING_CACHE_CAPACITY"
	EnvRankingLifetime      = "YACYDHTSEARCH_RANKING_LIFETIME"

	DefaultListenAddr           = ":8080"
	DefaultOpsAddr              = ":9090"
	DefaultQueryBudget          = 5 * time.Second
	DefaultPeerCallBudget       = 4 * time.Second
	DefaultPeerSearchCooldown   = 5 * time.Second
	DefaultPeerCallsInFlight    = 24
	DefaultDirectoryCapacity    = 4096
	DefaultRefreshInterval      = 5 * time.Minute
	DefaultProbeBudget          = 3 * time.Second
	DefaultPartitionExponent    = 4
	DefaultPeerRedundancy       = 3
	DefaultMaxResponseBytes     = 4 * 1024 * 1024
	DefaultPeerItemsCeiling     = 10
	DefaultRankedItemsCeiling   = 50
	DefaultRankingCacheCapacity = 1024
	DefaultRankingLifetime      = 2 * time.Minute

	peerBudgetCeiling = 3 * time.Second
)

type ServiceConfig struct {
	ListenAddr         string
	OpsAddr            string
	NetworkName        string
	SeedlistURLs       []string
	EgressProxyURL     *url.URL
	QueryBudget        time.Duration
	PeerCallBudget     time.Duration
	PeerSearchCooldown time.Duration
	PeerCallsInFlight  int
	DirectoryCapacity  int
	RefreshInterval    time.Duration
	ProbeBudget        time.Duration
	Partitions         yacymodel.DHTRingPartitions
	PeerRedundancy     int
	MaxResponseBytes   int64
	PeerItemsCeiling   int
	RankedItemsCeiling int
	NATSURL            string
	RankingCache       int
	RankingLifetime    time.Duration
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

	return ServiceConfig{
		ListenAddr:         envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:            envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		NetworkName:        envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork),
		SeedlistURLs:       seedlistURLs,
		EgressProxyURL:     egressProxyURL,
		QueryBudget:        durations.queryBudget,
		PeerCallBudget:     durations.peerCallBudget,
		PeerSearchCooldown: durations.peerSearchCooldown,
		PeerCallsInFlight:  counts.peerCallsInFlight,
		DirectoryCapacity:  counts.directoryCapacity,
		RefreshInterval:    durations.refreshInterval,
		ProbeBudget:        durations.probeBudget,
		Partitions:         partitions,
		PeerRedundancy:     counts.peerRedundancy,
		MaxResponseBytes:   maxResponseBytes,
		PeerItemsCeiling:   counts.peerItemsCeiling,
		RankedItemsCeiling: counts.rankedItemsCeiling,
		NATSURL:            strings.TrimSpace(getenv(EnvNATSURL)),
		RankingCache:       counts.rankingCacheCapacity,
		RankingLifetime:    durations.rankingLifetime,
	}, nil
}

type configuredDurations struct {
	queryBudget        time.Duration
	peerCallBudget     time.Duration
	peerSearchCooldown time.Duration
	refreshInterval    time.Duration
	probeBudget        time.Duration
	rankingLifetime    time.Duration
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
		{EnvPeerCallBudget, DefaultPeerCallBudget, &durations.peerCallBudget},
		{EnvPeerSearchCooldown, DefaultPeerSearchCooldown, &durations.peerSearchCooldown},
		{EnvRefreshInterval, DefaultRefreshInterval, &durations.refreshInterval},
		{EnvProbeBudget, DefaultProbeBudget, &durations.probeBudget},
		{EnvRankingLifetime, DefaultRankingLifetime, &durations.rankingLifetime},
	} {
		if *field.into, err = envconfig.Duration(getenv, field.key, field.fallback); err != nil {
			return configuredDurations{}, err
		}
	}

	return durations, nil
}

type configuredCounts struct {
	peerCallsInFlight    int
	directoryCapacity    int
	peerRedundancy       int
	peerItemsCeiling     int
	rankedItemsCeiling   int
	rankingCacheCapacity int
}

func countsOf(getenv func(string) string) (configuredCounts, error) {
	var counts configuredCounts
	var err error
	for _, field := range []struct {
		key      string
		fallback int
		into     *int
	}{
		{EnvPeerCallsInFlight, DefaultPeerCallsInFlight, &counts.peerCallsInFlight},
		{EnvDirectoryCapacity, DefaultDirectoryCapacity, &counts.directoryCapacity},
		{EnvPeerRedundancy, DefaultPeerRedundancy, &counts.peerRedundancy},
		{EnvPeerItemsCeiling, DefaultPeerItemsCeiling, &counts.peerItemsCeiling},
		{EnvRankedItemsCeiling, DefaultRankedItemsCeiling, &counts.rankedItemsCeiling},
		{EnvRankingCacheCapacity, DefaultRankingCacheCapacity, &counts.rankingCacheCapacity},
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
