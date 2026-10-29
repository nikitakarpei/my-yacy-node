package peerwire

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

const (
	hostsPerPeer    = 4
	baseURLsPerPeer = 5
	httpsPort       = yacymodel.Port(443)
)

type Callee struct {
	hash     yacymodel.Hash
	identity yacymodel.Optional[yacymodel.PeerIdentity]
}

func IdentifiedCallee(identity yacymodel.PeerIdentity) Callee {
	return Callee{hash: identity.Hash(), identity: yacymodel.Some(identity)}
}

func UnidentifiedCallee(peerHash yacymodel.Hash) Callee {
	return Callee{hash: peerHash}
}

func (c Callee) Hash() yacymodel.Hash {
	return c.hash
}

func (c Callee) Identity() yacymodel.Optional[yacymodel.PeerIdentity] {
	return c.identity
}

func (c Callee) BaseURLsFrom(addresses yacymodel.PeerAddresses) []BaseURL {
	if !c.identity.Present() {
		return unidentifiedBaseURLsFrom(addresses)
	}

	return identifiedBaseURLsFrom(addresses)
}

func unidentifiedBaseURLsFrom(addresses yacymodel.PeerAddresses) []BaseURL {
	var baseURLs []BaseURL
	for _, host := range addresses.Hosts[:min(len(addresses.Hosts), hostsPerPeer)] {
		if _, isAddr := host.Addr(); !isAddr {
			baseURLs = append(baseURLs, BaseURL{Scheme: HTTPS, Host: host, Port: httpsPort})
			if securePort, announced := addresses.SecurePort.Get(); announced &&
				securePort != httpsPort {
				baseURLs = append(baseURLs, BaseURL{Scheme: HTTPS, Host: host, Port: securePort})
			}
		}
		baseURLs = append(baseURLs, BaseURL{Scheme: HTTP, Host: host, Port: addresses.Port})
	}

	return baseURLs[:min(len(baseURLs), baseURLsPerPeer)]
}

func identifiedBaseURLsFrom(addresses yacymodel.PeerAddresses) []BaseURL {
	if securePort, announced := addresses.SecurePort.Get(); announced {
		return baseURLsAt(HTTPS, addresses.Hosts, securePort)
	}

	return baseURLsAt(HTTP, addresses.Hosts, addresses.Port)
}

func baseURLsAt(scheme Scheme, hosts []yacymodel.Host, port yacymodel.Port) []BaseURL {
	baseURLs := make([]BaseURL, 0, hostsPerPeer)
	for _, host := range hosts[:min(len(hosts), hostsPerPeer)] {
		baseURLs = append(baseURLs, BaseURL{Scheme: scheme, Host: host, Port: port})
	}

	return baseURLs
}
