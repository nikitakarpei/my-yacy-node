// Package tunnel sends each request through the egress proxy, in a CONNECT
// tunnel for https and in absolute-URL form for http.
package tunnel

import (
	"net/http"
	"net/url"
)

func New(proxyURL *url.URL) *http.Transport {
	return &http.Transport{
		Proxy:              http.ProxyURL(proxyURL),
		DisableCompression: true,
	}
}
