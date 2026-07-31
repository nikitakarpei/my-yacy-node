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
	envGreetsPerCycle          = "YACY_GREETS_PER_CYCLE"
	envKnownRosterCapacity     = "YACY_KNOWN_ROSTER_CAPACITY"
	envReachableRosterCapacity = "YACY_REACHABLE_ROSTER_CAPACITY"

	envDistributionEnabled           = "YACY_DISTRIBUTION_ENABLED"
	envDistributionRedundancy        = "YACY_DISTRIBUTION_REDUNDANCY"
	envDistributionPartitionExponent = "YACY_DISTRIBUTION_PARTITION_EXPONENT"
	envDistributionPostingsPerCycle  = "YACY_DISTRIBUTION_POSTINGS_PER_CYCLE"
	envDistributionCycleInterval     = "YACY_DISTRIBUTION_CYCLE_INTERVAL"
	envDistributionRefreshInterval   = "YACY_DISTRIBUTION_REFRESH_INTERVAL"
	envDistributionRetryInterval     = "YACY_DISTRIBUTION_RETRY_INTERVAL"
	envDistributionMinReachablePeers = "YACY_DISTRIBUTION_MIN_REACHABLE_PEERS"

	defaultPeerAddr                = ":8090"
	defaultOpsAddr                 = ":9090"
	defaultDataDir                 = "./data"
	defaultQuota                   = "1GB"
	defaultAnnounceInterval        = 10 * time.Minute
	defaultGreetsPerCycle          = 16
	defaultKnownRosterCapacity     = 4096
	defaultReachableRosterCapacity = 256

	defaultDistributionEnabled           = false
	defaultDistributionRedundancy        = 3
	defaultDistributionPartitionExponent = 4
	defaultDistributionPostingsPerCycle  = 1000
	defaultDistributionCycleInterval     = time.Minute
	defaultDistributionRefreshInterval   = 24 * time.Hour
	defaultDistributionRetryInterval     = 5 * time.Minute
	defaultDistributionMinReachablePeers = 32

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
	GreetsPerCycle          int
	KnownRosterCapacity     int
	ReachableRosterCapacity int
	Distribution            distributionConfig
	Crawl                   crawlConfig
}

type distributionConfig struct {
	Enabled           bool
	Redundancy        int
	PartitionExponent uint
	PostingsPerCycle  int
	CycleInterval     time.Duration
	RefreshInterval   time.Duration
	RetryInterval     time.Duration
	MinReachablePeers int
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
		GreetsPerCycle:          peering.GreetsPerCycle,
		KnownRosterCapacity:     peering.KnownRosterCapacity,
		ReachableRosterCapacity: peering.ReachableRosterCapacity,
		Distribution:            peering.Distribution,
	}, nil
}

type peeringConfig struct {
	AnnounceInterval        time.Duration
	GreetsPerCycle          int
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

	greetsPerCycle, err := envconfig.PositiveInt(getenv, envGreetsPerCycle, defaultGreetsPerCycle)
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
		GreetsPerCycle:          greetsPerCycle,
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

	postingsPerCycle, err := envconfig.PositiveInt(
		getenv, envDistributionPostingsPerCycle, defaultDistributionPostingsPerCycle,
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

	refreshInterval, err := envconfig.Duration(
		getenv, envDistributionRefreshInterval, defaultDistributionRefreshInterval,
	)
	if err != nil {
		return distributionConfig{}, err
	}

	retryInterval, err := envconfig.Duration(
		getenv, envDistributionRetryInterval, defaultDistributionRetryInterval,
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

	return distributionConfig{
		Enabled:           enabled,
		Redundancy:        redundancy,
		PartitionExponent: uint(partitionExponent),
		PostingsPerCycle:  postingsPerCycle,
		CycleInterval:     cycleInterval,
		RefreshInterval:   refreshInterval,
		RetryInterval:     retryInterval,
		MinReachablePeers: minReachablePeers,
	}, nil
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
