package peerwire

import (
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	hostsPerPeer    = 4
	baseURLsPerPeer = 5
	httpsPort       = yacymodel.Port(443)
)

var errUnknownScheme = errors.New("unknown scheme")

type Scheme string

const (
	HTTP      Scheme = "http"
	HTTPS     Scheme = "https"
	PeerHTTPS Scheme = "peer-https"
)

func ParseScheme(text string) (Scheme, error) {
	switch scheme := Scheme(text); scheme {
	case HTTP, HTTPS, PeerHTTPS:
		return scheme, nil
	default:
		return "", fmt.Errorf("%w: %q", errUnknownScheme, text)
	}
}

type BaseURL struct {
	Scheme Scheme
	Host   yacymodel.Host
	Port   yacymodel.Port
}

func BaseURLsFrom(
	addresses yacymodel.PeerAddresses,
	signedIdentity yacymodel.Optional[yacymodel.SignedIdentity],
) []BaseURL {
	securePort, announced := addresses.SecurePort.Get()
	if _, signed := signedIdentity.Get(); signed && announced {
		return peerHTTPSBaseURLsFrom(addresses.Hosts, securePort)
	}

	return unsignedBaseURLsFrom(addresses)
}

func peerHTTPSBaseURLsFrom(hosts []yacymodel.Host, securePort yacymodel.Port) []BaseURL {
	baseURLs := make([]BaseURL, 0, hostsPerPeer)
	for _, host := range hosts[:min(len(hosts), hostsPerPeer)] {
		baseURLs = append(baseURLs, BaseURL{Scheme: PeerHTTPS, Host: host, Port: securePort})
	}

	return baseURLs
}

func unsignedBaseURLsFrom(addresses yacymodel.PeerAddresses) []BaseURL {
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

func (o BaseURL) String() string {
	return o.urlOf("")
}

func (o BaseURL) urlOf(path string) string {
	target := url.URL{
		Scheme: o.Scheme.urlScheme(),
		Host:   net.JoinHostPort(o.Host.String(), o.Port.String()),
		Path:   path,
	}

	return target.String()
}

func (s Scheme) urlScheme() string {
	if s == PeerHTTPS {
		return string(HTTPS)
	}

	return string(s)
}
