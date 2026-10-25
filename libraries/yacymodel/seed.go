package yacymodel

import (
	"errors"
	"slices"
	"time"
)

var ErrBadSeed = errors.New("bad seed")

// Seed is a peer's self-description as broadcast to the network: who the peer
// is, how to reach it, what it will do for others, and what it has indexed.
type Seed struct {
	Hash     Hash
	Name     PeerName
	PeerType PeerType

	PrimaryAddress      Optional[Host]
	AdditionalAddresses Optional[[]Host]
	Port                Optional[Port]
	SecurePort          Optional[Port]
	SeedListAddress     Optional[SeedListURL]

	RemotePeerType Optional[PeerType]
	Capabilities   Optional[PeerCapabilities]
	SolrAvailable  Optional[bool]
	Version        Optional[SoftwareVersion]
	Tags           PeerTags

	FirstSeen      Optional[time.Time]
	LastSeen       Optional[time.Time]
	DisconnectedAt Optional[time.Time]
	UTCOffset      Optional[UTCOffset]
	Uptime         time.Duration

	IndexingSpeed     int
	RetrievalSpeed    int
	UplinkSpeed       int
	ClientConnectRate float64

	IndexedWords    int
	StoredURLs      int
	NoticedURLs     int
	RemoteCrawlURLs int
	StoredSeeds     int

	WordsSent     int
	WordsReceived int
	URLsSent      int
	URLsReceived  int

	News Optional[PeerNews]
}

// IsAddressable reports whether the peer published any address a peer can reach.
func (s Seed) IsAddressable() bool {
	_, addressable := s.Addresses()

	return addressable
}

func (s Seed) AcceptsRemoteIndex() bool {
	capabilities, known := s.Capabilities.Get()

	return known && capabilities.AcceptRemoteIndex
}

func (s Seed) Addresses() (PeerAddresses, bool) {
	port, ok := s.Port.Get()
	if !ok || !port.inRange() {
		return PeerAddresses{}, false
	}
	hosts := s.hosts()
	if len(hosts) == 0 {
		return PeerAddresses{}, false
	}

	return PeerAddresses{Hosts: hosts, Port: port, SecurePort: s.announcedSecurePort()}, true
}

func (s Seed) hosts() []Host {
	candidates := s.AdditionalAddresses.OrElse(nil)
	if primary, ok := s.PrimaryAddress.Get(); ok {
		candidates = append([]Host{primary}, candidates...)
	}

	var hosts []Host
	for _, host := range candidates {
		if host.String() != "" && !slices.Contains(hosts, host) {
			hosts = append(hosts, host)
		}
	}

	return hosts
}

func (s Seed) announcedSecurePort() Optional[Port] {
	capabilities, known := s.Capabilities.Get()
	if !known || !capabilities.SSLAvailable {
		return None[Port]()
	}

	return s.SecurePort
}
