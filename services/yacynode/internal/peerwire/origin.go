package peerwire

import (
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Scheme string

const (
	HTTP  Scheme = "http"
	HTTPS Scheme = "https"
)

type Origin struct {
	Scheme Scheme
	Host   yacymodel.Host
	Port   yacymodel.Port
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
