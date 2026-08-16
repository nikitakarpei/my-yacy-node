package nodeconfiguration

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvSeedlistURLs            = "YACY_SEEDLIST_URLS"
	EnvAnnounceInterval        = "YACY_ANNOUNCE_INTERVAL"
	EnvPeerContactConcurrency  = "YACY_PEER_CONTACT_CONCURRENCY"
	EnvKnownRosterCapacity     = "YACY_KNOWN_ROSTER_CAPACITY"
	EnvReachableRosterCapacity = "YACY_REACHABLE_ROSTER_CAPACITY"

	DefaultAnnounceInterval        = 10 * time.Minute
	DefaultPeerContactConcurrency  = 16
	DefaultKnownRosterCapacity     = 4096
	DefaultReachableRosterCapacity = 256
)

type PeerExchangeConfig struct {
	SeedlistURLs            []string
	AnnounceInterval        time.Duration
	PeerContactConcurrency  int
	KnownRosterCapacity     int
	ReachableRosterCapacity int
}

func (c PeerExchangeConfig) Announcing() bool {
	return len(c.SeedlistURLs) > 0
}

func loadPeerExchangeConfig(getenv func(string) string) (PeerExchangeConfig, error) {
	announceInterval, err := envconfig.Duration(
		getenv,
		EnvAnnounceInterval,
		DefaultAnnounceInterval,
	)
	if err != nil {
		return PeerExchangeConfig{}, err
	}

	peerContactConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvPeerContactConcurrency,
		DefaultPeerContactConcurrency,
	)
	if err != nil {
		return PeerExchangeConfig{}, err
	}

	knownRosterCapacity, err := envconfig.PositiveInt(
		getenv,
		EnvKnownRosterCapacity,
		DefaultKnownRosterCapacity,
	)
	if err != nil {
		return PeerExchangeConfig{}, err
	}

	reachableRosterCapacity, err := envconfig.PositiveInt(
		getenv,
		EnvReachableRosterCapacity,
		DefaultReachableRosterCapacity,
	)
	if err != nil {
		return PeerExchangeConfig{}, err
	}

	return PeerExchangeConfig{
		SeedlistURLs:            envconfig.List(getenv, EnvSeedlistURLs),
		AnnounceInterval:        announceInterval,
		PeerContactConcurrency:  peerContactConcurrency,
		KnownRosterCapacity:     knownRosterCapacity,
		ReachableRosterCapacity: reachableRosterCapacity,
	}, nil
}
