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
	EnvPeerHash                = "YACY_PEER_HASH"
	EnvPeerName                = "YACY_PEER_NAME"
	EnvNetworkName             = "YACY_NETWORK_NAME"
	EnvPeerAddr                = "YACY_PEER_ADDR"
	EnvOpsAddr                 = "YACY_OPS_ADDR"
	EnvAdvertiseHost           = "YACY_ADVERTISE_HOST"
	EnvAdvertisePort           = "YACY_ADVERTISE_PORT"
	EnvDataDir                 = "YACY_DATA_DIR"
	EnvStorageQuota            = "YACY_STORAGE_QUOTA"
	EnvEscrowPostingCapacity   = "YACY_ESCROW_POSTING_CAPACITY"
	EnvTrustedProxies          = "YACY_TRUSTED_PROXIES"
	EnvSeedlistURLs            = "YACY_SEEDLIST_URLS"
	EnvAnnounceInterval        = "YACY_ANNOUNCE_INTERVAL"
	EnvPeerContactConcurrency  = "YACY_PEER_CONTACT_CONCURRENCY"
	EnvKnownRosterCapacity     = "YACY_KNOWN_ROSTER_CAPACITY"
	EnvReachableRosterCapacity = "YACY_REACHABLE_ROSTER_CAPACITY"

	EnvDistributionEnabled               = "YACY_DISTRIBUTION_ENABLED"
	EnvDistributionRedundancy            = "YACY_DISTRIBUTION_REDUNDANCY"
	EnvDistributionPartitionExponent     = "YACY_DISTRIBUTION_PARTITION_EXPONENT"
	EnvDistributionPostingsPerBatch      = "YACY_DISTRIBUTION_POSTINGS_PER_BATCH"
	EnvDistributionCycleInterval         = "YACY_DISTRIBUTION_CYCLE_INTERVAL"
	EnvDistributionDrainBudget           = "YACY_DISTRIBUTION_DRAIN_BUDGET"
	EnvDistributionLongestOfferInterval  = "YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL"
	EnvDistributionShortestOfferInterval = "YACY_DISTRIBUTION_SHORTEST_OFFER_INTERVAL"
	EnvDistributionRecipientCooldown     = "YACY_DISTRIBUTION_RECIPIENT_COOLDOWN"
	EnvDistributionMinReachablePeers     = "YACY_DISTRIBUTION_MIN_REACHABLE_PEERS"
	EnvDistributionURLMetadataBatchSize  = "YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE"

	DefaultPeerAddr                = ":8090"
	DefaultOpsAddr                 = ":9090"
	DefaultDataDir                 = "./data"
	DefaultQuota                   = "1GB"
	DefaultEscrowPostingCapacity   = 8192
	DefaultAnnounceInterval        = 10 * time.Minute
	DefaultPeerContactConcurrency  = 16
	DefaultKnownRosterCapacity     = 4096
	DefaultReachableRosterCapacity = 256

	DefaultDistributionEnabled               = false
	DefaultDistributionRedundancy            = 3
	DefaultDistributionPartitionExponent     = 4
	DefaultDistributionPostingsPerBatch      = 1000
	DefaultDistributionCycleInterval         = time.Minute
	DefaultDistributionDrainBudget           = time.Minute
	DefaultDistributionLongestOfferInterval  = 24 * time.Hour
	DefaultDistributionShortestOfferInterval = 5 * time.Minute
	DefaultDistributionRecipientCooldown     = 10 * time.Minute
	DefaultDistributionMinReachablePeers     = 32
	DefaultDistributionURLMetadataBatchSize  = 50

	StorageFileName = "yacy-rwipostings.db"
)

type NodeConfig struct {
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
	TrustedProxyNetworks    []*net.IPNet
	ProxyURL                *url.URL
	SeedlistURLs            []string
	AnnounceInterval        time.Duration
	PeerContactConcurrency  int
	KnownRosterCapacity     int
	ReachableRosterCapacity int
	Escrow                  EscrowConfig
	Distribution            DistributionConfig
	Crawl                   CrawlConfig
}

type EscrowConfig struct {
	PostingCapacity int
}

type DistributionConfig struct {
	Enabled              bool
	Redundancy           int
	PartitionExponent    uint
	PostingsPerBatch     int
	CycleInterval        time.Duration
	DrainBudget          time.Duration
	OfferInterval        OfferIntervalConfig
	RecipientCooldown    time.Duration
	MinReachablePeers    int
	URLMetadataBatchSize int
}

func LoadNodeConfig(getenv func(string) string) (NodeConfig, error) {
	hash, err := yacymodel.ParseHash(strings.TrimSpace(getenv(EnvPeerHash)))
	if err != nil {
		return NodeConfig{}, fmt.Errorf("%s: %w", EnvPeerHash, err)
	}

	rawName, err := requiredEnv(getenv, EnvPeerName)
	if err != nil {
		return NodeConfig{}, err
	}
	name, err := yacymodel.ParsePeerName(rawName)
	if err != nil {
		return NodeConfig{}, fmt.Errorf("%s: %w", EnvPeerName, err)
	}

	peerAddr := envconfig.String(getenv, EnvPeerAddr, DefaultPeerAddr)

	seedlistURLs := envconfig.List(getenv, EnvSeedlistURLs)

	peering, err := loadPeeringConfig(getenv)
	if err != nil {
		return NodeConfig{}, err
	}

	host, err := advertiseHost(getenv, len(seedlistURLs) > 0)
	if err != nil {
		return NodeConfig{}, err
	}

	port, err := advertisePort(getenv, peerAddr)
	if err != nil {
		return NodeConfig{}, err
	}

	quota, err := bytesize.Parse(envconfig.String(getenv, EnvStorageQuota, DefaultQuota))
	if err != nil {
		return NodeConfig{}, fmt.Errorf("%s: %w", EnvStorageQuota, err)
	}

	escrow, err := loadEscrowConfig(getenv)
	if err != nil {
		return NodeConfig{}, err
	}

	trustedProxyNetworks, err := trustedProxyNetworksFrom(getenv(EnvTrustedProxies))
	if err != nil {
		return NodeConfig{}, fmt.Errorf("%s: %w", EnvTrustedProxies, err)
	}

	proxyURL, err := egressProxyURL(getenv)
	if err != nil {
		return NodeConfig{}, err
	}

	dataDir := envconfig.String(getenv, EnvDataDir, DefaultDataDir)

	return NodeConfig{
		Hash:                    hash,
		NetworkName:             envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork),
		Name:                    name,
		AdvertiseHost:           host,
		AdvertisePort:           port,
		Flags:                   seniorFlags(),
		PeerAddr:                peerAddr,
		OpsAddr:                 envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		StoragePath:             filepath.Join(dataDir, StorageFileName),
		StorageQuotaByte:        quota,
		Escrow:                  escrow,
		TrustedProxyNetworks:    trustedProxyNetworks,
		ProxyURL:                proxyURL,
		SeedlistURLs:            seedlistURLs,
		AnnounceInterval:        peering.AnnounceInterval,
		PeerContactConcurrency:  peering.PeerContactConcurrency,
		KnownRosterCapacity:     peering.KnownRosterCapacity,
		ReachableRosterCapacity: peering.ReachableRosterCapacity,
		Distribution:            peering.Distribution,
		Crawl:                   loadCrawlConfig(getenv),
	}, nil
}

type peeringConfig struct {
	AnnounceInterval        time.Duration
	PeerContactConcurrency  int
	KnownRosterCapacity     int
	ReachableRosterCapacity int
	Distribution            DistributionConfig
}

func loadPeeringConfig(getenv func(string) string) (peeringConfig, error) {
	announceInterval, err := envconfig.Duration(
		getenv,
		EnvAnnounceInterval,
		DefaultAnnounceInterval,
	)
	if err != nil {
		return peeringConfig{}, err
	}

	peerContactConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvPeerContactConcurrency,
		DefaultPeerContactConcurrency,
	)
	if err != nil {
		return peeringConfig{}, err
	}

	knownRosterCapacity, err := envconfig.PositiveInt(
		getenv,
		EnvKnownRosterCapacity,
		DefaultKnownRosterCapacity,
	)
	if err != nil {
		return peeringConfig{}, err
	}

	reachableRosterCapacity, err := envconfig.PositiveInt(
		getenv,
		EnvReachableRosterCapacity,
		DefaultReachableRosterCapacity,
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

func loadDistributionConfig(getenv func(string) string) (DistributionConfig, error) {
	enabled, err := envconfig.Bool(getenv, EnvDistributionEnabled, DefaultDistributionEnabled)
	if err != nil {
		return DistributionConfig{}, err
	}

	redundancy, err := envconfig.PositiveInt(
		getenv, EnvDistributionRedundancy, DefaultDistributionRedundancy,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	partitionExponent, err := envconfig.PositiveInt(
		getenv, EnvDistributionPartitionExponent, DefaultDistributionPartitionExponent,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	postingsPerBatch, err := envconfig.PositiveInt(
		getenv, EnvDistributionPostingsPerBatch, DefaultDistributionPostingsPerBatch,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	cycleInterval, err := envconfig.Duration(
		getenv, EnvDistributionCycleInterval, DefaultDistributionCycleInterval,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	drainBudget, err := envconfig.Duration(
		getenv, EnvDistributionDrainBudget, DefaultDistributionDrainBudget,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	offerInterval, err := loadOfferIntervalConfig(getenv)
	if err != nil {
		return DistributionConfig{}, err
	}

	recipientCooldown, err := envconfig.Duration(
		getenv, EnvDistributionRecipientCooldown, DefaultDistributionRecipientCooldown,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	minReachablePeers, err := envconfig.PositiveInt(
		getenv, EnvDistributionMinReachablePeers, DefaultDistributionMinReachablePeers,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	urlMetadataBatchSize, err := envconfig.PositiveInt(
		getenv, EnvDistributionURLMetadataBatchSize, DefaultDistributionURLMetadataBatchSize,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	return DistributionConfig{
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

type OfferIntervalConfig struct {
	Longest  time.Duration
	Shortest time.Duration
}

func loadOfferIntervalConfig(getenv func(string) string) (OfferIntervalConfig, error) {
	longest, err := envconfig.Duration(
		getenv, EnvDistributionLongestOfferInterval, DefaultDistributionLongestOfferInterval,
	)
	if err != nil {
		return OfferIntervalConfig{}, err
	}

	shortest, err := envconfig.Duration(
		getenv, EnvDistributionShortestOfferInterval, DefaultDistributionShortestOfferInterval,
	)
	if err != nil {
		return OfferIntervalConfig{}, err
	}

	return OfferIntervalConfig{Longest: longest, Shortest: shortest}, nil
}

func advertiseHost(getenv func(string) string, announcing bool) (string, error) {
	host := strings.TrimSpace(getenv(EnvAdvertiseHost))
	if host == "" && announcing {
		return "", fmt.Errorf("%s: must be set when announcing to the network", EnvAdvertiseHost)
	}

	return host, nil
}

func advertisePort(getenv func(string) string, peerAddr string) (int, error) {
	if raw := strings.TrimSpace(getenv(EnvAdvertisePort)); raw != "" {
		return positiveInt(EnvAdvertisePort, raw)
	}

	_, portPart, err := net.SplitHostPort(peerAddr)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", EnvPeerAddr, err)
	}

	return positiveInt(EnvPeerAddr, portPart)
}

func loadEscrowConfig(getenv func(string) string) (EscrowConfig, error) {
	postingCapacity, err := envconfig.PositiveInt(
		getenv,
		EnvEscrowPostingCapacity,
		DefaultEscrowPostingCapacity,
	)
	if err != nil {
		return EscrowConfig{}, err
	}

	return EscrowConfig{PostingCapacity: postingCapacity}, nil
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
