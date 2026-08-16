package main

import (
	"net/http"
	"net/url"
	"time"
)

const outboundRequestTimeout = 30 * time.Second

func newEgressProxyClient(proxyURL *url.URL) *http.Client {
	return &http.Client{
		Timeout:   outboundRequestTimeout,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}
}
