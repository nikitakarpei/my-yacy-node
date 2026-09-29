// Package tunnel sends each request through the egress proxy, in a CONNECT
// tunnel for https and in absolute-URL form for http.
package tunnel

import (
	"crypto/tls"
	"net/http"
	"net/url"
)

func New(proxyURL *url.URL, tlsConfig *tls.Config) *http.Transport {
	return &http.Transport{
		Proxy:              http.ProxyURL(proxyURL),
		TLSClientConfig:    tlsConfig,
		DisableCompression: true,
		DisableKeepAlives:  true,
	}
}
