package peerwire

import (
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	hostsPerPeer   = 4
	originsPerPeer = 5
	httpsPort      = yacymodel.Port(443)
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

type Origin struct {
	Scheme Scheme
	Host   yacymodel.Host
	Port   yacymodel.Port
}

func OriginsFrom(addresses yacymodel.PeerAddresses) []Origin {
	var origins []Origin
	for _, host := range addresses.Hosts[:min(len(addresses.Hosts), hostsPerPeer)] {
		if _, isAddr := host.Addr(); !isAddr {
			origins = append(origins, Origin{Scheme: HTTPS, Host: host, Port: httpsPort})
			if securePort, announced := addresses.SecurePort.Get(); announced &&
				securePort != httpsPort {
				origins = append(origins, Origin{Scheme: HTTPS, Host: host, Port: securePort})
			}
		}
		origins = append(origins, Origin{Scheme: HTTP, Host: host, Port: addresses.Port})
	}

	return origins[:min(len(origins), originsPerPeer)]
}

func (o Origin) String() string {
	return o.urlOf("")
}

func (o Origin) urlOf(path string) string {
	target := url.URL{
		Scheme: string(o.Scheme),
		Host:   net.JoinHostPort(o.Host.String(), o.Port.String()),
		Path:   path,
	}

	return target.String()
}
