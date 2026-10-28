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

	PrimaryHost     Optional[Host]
	AdditionalHosts Optional[[]Host]
	Port            Optional[Port]
	SecurePort      Optional[Port]
	SeedListAddress Optional[SeedListURL]

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

	Identity         Optional[PeerIdentity]
	AddressSignature Optional[AddressSignature]
}

// IsAddressable reports whether the peer published any address a peer can reach.
func (s Seed) IsAddressable() bool {
	_, addressable := s.Addresses()

	return addressable
}

func (s Seed) VerifiedIdentityIn(networkName string) (PeerIdentity, bool) {
	identity, identified := s.Identity.Get()
	signature, signed := s.AddressSignature.Get()
	if !identified || !signed || identity.Hash() != s.Hash {
		return PeerIdentity{}, false
	}
	if !identity.verifies(
		addressClaimOf(s, networkName, signature.IssuedAt),
		signature.Signature[:],
	) {
		return PeerIdentity{}, false
	}

	return identity, true
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
	candidates := s.AdditionalHosts.OrElse(nil)
	if primary, ok := s.PrimaryHost.Get(); ok {
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
