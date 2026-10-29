package peerwire

import (
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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
