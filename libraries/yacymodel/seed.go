package yacymodel

import (
	"errors"
	"fmt"
	"net/url"
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
	Version        Optional[SoftwareVersion]
	Tags           PeerTags
	SolrAvailable  Optional[bool]

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
	if s.PrimaryAddress.Present() {
		return true
	}
	if hosts, ok := s.AdditionalAddresses.Get(); ok && len(hosts) > 0 {
		return true
	}
	return false
}

func (s Seed) NetworkAddress() (NetworkAddress, bool) {
	host, ok := s.PrimaryAddress.Get()
	if !ok {
		return NetworkAddress{}, false
	}
	port, ok := s.Port.Get()
	if !ok {
		return NetworkAddress{}, false
	}
	address, err := NetworkAddressOf(host, port)
	if err != nil {
		return NetworkAddress{}, false
	}

	return address, true
}

func (s Seed) HTTPEndpoint(path string) (*url.URL, error) {
	address, ok := s.NetworkAddress()
	if !ok {
		return nil, fmt.Errorf("%w: no reachable address", ErrBadSeed)
	}

	return &url.URL{
		Scheme: "http",
		Host:   address.String(),
		Path:   path,
	}, nil
}
