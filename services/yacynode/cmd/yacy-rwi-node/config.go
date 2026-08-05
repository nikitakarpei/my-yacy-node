package main

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/bytesize"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	envPeerHash                = "YACY_PEER_HASH"
	envPeerName                = "YACY_PEER_NAME"
	envNetworkName             = "YACY_NETWORK_NAME"
	envPeerAddr                = "YACY_PEER_ADDR"
	envOpsAddr                 = "YACY_OPS_ADDR"
	envAdvertiseHost           = "YACY_ADVERTISE_HOST"
	envAdvertisePort           = "YACY_ADVERTISE_PORT"
	envDataDir                 = "YACY_DATA_DIR"
	envStorageQuota            = "YACY_STORAGE_QUOTA"
	envTrustedProxies          = "YACY_TRUSTED_PROXIES"
	envSeedlistURLs            = "YACY_SEEDLIST_URLS"
	envAnnounceInterval        = "YACY_ANNOUNCE_INTERVAL"
	envPeerContactConcurrency  = "YACY_PEER_CONTACT_CONCURRENCY"
	envKnownRosterCapacity     = "YACY_KNOWN_ROSTER_CAPACITY"
	envReachableRosterCapacity = "YACY_REACHABLE_ROSTER_CAPACITY"

	envDistributionEnabled               = "YACY_DISTRIBUTION_ENABLED"
	envDistributionRedundancy            = "YACY_DISTRIBUTION_REDUNDANCY"
	envDistributionPartitionExponent     = "YACY_DISTRIBUTION_PARTITION_EXPONENT"
	envDistributionPostingsPerBatch      = "YACY_DISTRIBUTION_POSTINGS_PER_BATCH"
	envDistributionCycleInterval         = "YACY_DISTRIBUTION_CYCLE_INTERVAL"
	envDistributionDrainBudget           = "YACY_DISTRIBUTION_DRAIN_BUDGET"
	envDistributionLongestOfferInterval  = "YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL"
	envDistributionShortestOfferInterval = "YACY_DISTRIBUTION_SHORTEST_OFFER_INTERVAL"
	envDistributionRecipientCooldown     = "YACY_DISTRIBUTION_RECIPIENT_COOLDOWN"
	envDistributionMinReachablePeers     = "YACY_DISTRIBUTION_MIN_REACHABLE_PEERS"
	envDistributionURLMetadataBatchSize  = "YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE"

	defaultPeerAddr                = ":8090"
	defaultOpsAddr                 = ":9090"
	defaultDataDir                 = "./data"
	defaultQuota                   = "1GB"
	defaultAnnounceInterval        = 10 * time.Minute
	defaultPeerContactConcurrency  = 16
	defaultKnownRosterCapacity     = 4096
	defaultReachableRosterCapacity = 256

	defaultDistributionEnabled               = false
	defaultDistributionRedundancy            = 3
	defaultDistributionPartitionExponent     = 4
	defaultDistributionPostingsPerBatch      = 1000
	defaultDistributionCycleInterval         = time.Minute
	defaultDistributionDrainBudget           = time.Minute
	defaultDistributionLongestOfferInterval  = 24 * time.Hour
	defaultDistributionShortestOfferInterval = 5 * time.Minute
	defaultDistributionRecipientCooldown     = 10 * time.Minute
	defaultDistributionMinReachablePeers     = 32
	defaultDistributionURLMetadataBatchSize  = 50

	storageFileName = "yacy-rwipostings.db"
)

type nodeConfig struct {
	Hash                    yacymodel.Hash
	NetworkName             string
	Name                    yacymodel.PeerName
	AdvertiseHost           string
	AdvertisePort           int
	Flags                   yacymodel.PeerCapabilities
	PeerAddr                string
	OpsAddr                 string
	StoragePath             string
	StorageQuotaByte        int64
	TrustedProxies          []*net.IPNet
	ProxyURL                *url.URL
	SeedlistURLs            []string
	AnnounceInterval        time.Duration
	PeerContactConcurrency  int
	KnownRosterCapacity     int
	ReachableRosterCapacity int
	Distribution            distributionConfig
	Crawl                   crawlConfig
}

type distributionConfig struct {
	Enabled              bool
	Redundancy           int
	PartitionExponent    uint
	PostingsPerBatch     int
	CycleInterval        time.Duration
	DrainBudget          time.Duration
	OfferInterval        offerIntervalConfig
	RecipientCooldown    time.Duration
	MinReachablePeers    int
	URLMetadataBatchSize int
}

func loadNodeConfig(getenv func(string) string) (nodeConfig, error) {
	hash, err := yacymodel.ParseHash(strings.TrimSpace(getenv(envPeerHash)))
	if err != nil {
		return nodeConfig{}, fmt.Errorf("%s: %w", envPeerHash, err)
	}

	rawName, err := requiredEnv(getenv, envPeerName)
	if err != nil {
		return nodeConfig{}, err
	}
	name, err := yacymodel.ParsePeerName(rawName)
	if err != nil {
		return nodeConfig{}, fmt.Errorf("%s: %w", envPeerName, err)
	}

	peerAddr := envconfig.String(getenv, envPeerAddr, defaultPeerAddr)

	seedlistURLs := envconfig.List(getenv, envSeedlistURLs)

	peering, err := loadPeeringConfig(getenv)
	if err != nil {
		return nodeConfig{}, err
	}

	host, err := advertiseHost(getenv, len(seedlistURLs) > 0)
	if err != nil {
		return nodeConfig{}, err
	}

	port, err := advertisePort(getenv, peerAddr)
	if err != nil {
		return nodeConfig{}, err
	}

	quota, err := bytesize.Parse(envconfig.String(getenv, envStorageQuota, defaultQuota))
	if err != nil {
		return nodeConfig{}, fmt.Errorf("%s: %w", envStorageQuota, err)
	}

	proxies, err := parseTrustedProxies(getenv(envTrustedProxies))
	if err != nil {
		return nodeConfig{}, fmt.Errorf("%s: %w", envTrustedProxies, err)
	}

	proxyURL, err := egressProxyURL(getenv)
	if err != nil {
		return nodeConfig{}, err
	}

	dataDir := envconfig.String(getenv, envDataDir, defaultDataDir)

	return nodeConfig{
		Hash:                    hash,
		NetworkName:             envconfig.String(getenv, envNetworkName, yacyproto.DefaultNetwork),
		Name:                    name,
		AdvertiseHost:           host,
		AdvertisePort:           port,
		Flags:                   seniorFlags(),
		PeerAddr:                peerAddr,
		OpsAddr:                 envconfig.String(getenv, envOpsAddr, defaultOpsAddr),
		StoragePath:             filepath.Join(dataDir, storageFileName),
		StorageQuotaByte:        quota,
		TrustedProxies:          proxies,
		ProxyURL:                proxyURL,
		SeedlistURLs:            seedlistURLs,
		AnnounceInterval:        peering.AnnounceInterval,
		PeerContactConcurrency:  peering.PeerContactConcurrency,
		KnownRosterCapacity:     peering.KnownRosterCapacity,
		ReachableRosterCapacity: peering.ReachableRosterCapacity,
		Distribution:            peering.Distribution,
	}, nil
}

type peeringConfig struct {
	AnnounceInterval        time.Duration
	PeerContactConcurrency  int
	KnownRosterCapacity     int
	ReachableRosterCapacity int
	Distribution            distributionConfig
}

func loadPeeringConfig(getenv func(string) string) (peeringConfig, error) {
	announceInterval, err := envconfig.Duration(
		getenv,
		envAnnounceInterval,
		defaultAnnounceInterval,
	)
	if err != nil {
		return peeringConfig{}, err
	}

	peerContactConcurrency, err := envconfig.PositiveInt(
		getenv,
		envPeerContactConcurrency,
		defaultPeerContactConcurrency,
	)
	if err != nil {
		return peeringConfig{}, err
	}

	knownRosterCapacity, err := envconfig.PositiveInt(
		getenv,
		envKnownRosterCapacity,
		defaultKnownRosterCapacity,
	)
	if err != nil {
		return peeringConfig{}, err
	}

	reachableRosterCapacity, err := envconfig.PositiveInt(
		getenv,
		envReachableRosterCapacity,
		defaultReachableRosterCapacity,
	)
	if err != nil {
		return peeringConfig{}, err
	}

	distribution, err := loadDistributionConfig(getenv)
	if err != nil {
		return peeringConfig{}, err
	}

	return peeringConfig{
		AnnounceInterval:        announceInterval,
		PeerContactConcurrency:  peerContactConcurrency,
		KnownRosterCapacity:     knownRosterCapacity,
		ReachableRosterCapacity: reachableRosterCapacity,
		Distribution:            distribution,
	}, nil
}

func loadDistributionConfig(getenv func(string) string) (distributionConfig, error) {
	enabled, err := envconfig.Bool(getenv, envDistributionEnabled, defaultDistributionEnabled)
	if err != nil {
		return distributionConfig{}, err
	}

	redundancy, err := envconfig.PositiveInt(
		getenv, envDistributionRedundancy, defaultDistributionRedundancy,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	partitionExponent, err := envconfig.PositiveInt(
		getenv, envDistributionPartitionExponent, defaultDistributionPartitionExponent,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	postingsPerBatch, err := envconfig.PositiveInt(
		getenv, envDistributionPostingsPerBatch, defaultDistributionPostingsPerBatch,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	cycleInterval, err := envconfig.Duration(
		getenv, envDistributionCycleInterval, defaultDistributionCycleInterval,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	drainBudget, err := envconfig.Duration(
		getenv, envDistributionDrainBudget, defaultDistributionDrainBudget,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	offerInterval, err := loadOfferIntervalConfig(getenv)
	if err != nil {
		return distributionConfig{}, err
	}

	recipientCooldown, err := envconfig.Duration(
		getenv, envDistributionRecipientCooldown, defaultDistributionRecipientCooldown,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	minReachablePeers, err := envconfig.PositiveInt(
		getenv, envDistributionMinReachablePeers, defaultDistributionMinReachablePeers,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	urlMetadataBatchSize, err := envconfig.PositiveInt(
		getenv, envDistributionURLMetadataBatchSize, defaultDistributionURLMetadataBatchSize,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	return distributionConfig{
		Enabled:              enabled,
		Redundancy:           redundancy,
		PartitionExponent:    uint(partitionExponent),
		PostingsPerBatch:     postingsPerBatch,
		CycleInterval:        cycleInterval,
		DrainBudget:          drainBudget,
		OfferInterval:        offerInterval,
		RecipientCooldown:    recipientCooldown,
		MinReachablePeers:    minReachablePeers,
		URLMetadataBatchSize: urlMetadataBatchSize,
	}, nil
}

type offerIntervalConfig struct {
	Longest  time.Duration
	Shortest time.Duration
}

func loadOfferIntervalConfig(getenv func(string) string) (offerIntervalConfig, error) {
	longest, err := envconfig.Duration(
		getenv, envDistributionLongestOfferInterval, defaultDistributionLongestOfferInterval,
	)
	if err != nil {
		return offerIntervalConfig{}, err
	}

	shortest, err := envconfig.Duration(
		getenv, envDistributionShortestOfferInterval, defaultDistributionShortestOfferInterval,
	)
	if err != nil {
		return offerIntervalConfig{}, err
	}

	return offerIntervalConfig{Longest: longest, Shortest: shortest}, nil
}

func advertiseHost(getenv func(string) string, announcing bool) (string, error) {
	host := strings.TrimSpace(getenv(envAdvertiseHost))
	if host == "" && announcing {
		return "", fmt.Errorf("%s: must be set when announcing to the network", envAdvertiseHost)
	}

	return host, nil
}

func advertisePort(getenv func(string) string, peerAddr string) (int, error) {
	if raw := strings.TrimSpace(getenv(envAdvertisePort)); raw != "" {
		return positiveInt(envAdvertisePort, raw)
	}

	_, portPart, err := net.SplitHostPort(peerAddr)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", envPeerAddr, err)
	}

	return positiveInt(envPeerAddr, portPart)
}

func seniorFlags() yacymodel.PeerCapabilities {
	return yacymodel.PeerCapabilities{
		DirectConnect:     true,
		AcceptRemoteIndex: true,
	}
}

func requiredEnv(getenv func(string) string, key string) (string, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s: must be set", key)
	}

	return value, nil
}

func positiveInt(key, raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: must be positive", key)
	}

	return value, nil
}
