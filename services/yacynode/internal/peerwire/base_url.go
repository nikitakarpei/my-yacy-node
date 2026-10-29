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
	HTTP  Scheme = "http"
	HTTPS Scheme = "https"
)

func ParseScheme(text string) (Scheme, error) {
	switch scheme := Scheme(text); scheme {
	case HTTP, HTTPS:
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
	identity yacymodel.Optional[yacymodel.PeerIdentity],
) []BaseURL {
	if !identity.Present() {
		return unidentifiedBaseURLsFrom(addresses)
	}
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

func (o BaseURL) String() string {
	return o.urlOf("")
}

func (o BaseURL) urlOf(path string) string {
	target := url.URL{
		Scheme: string(o.Scheme),
		Host:   net.JoinHostPort(o.Host.String(), o.Port.String()),
		Path:   path,
	}

	return target.String()
}
