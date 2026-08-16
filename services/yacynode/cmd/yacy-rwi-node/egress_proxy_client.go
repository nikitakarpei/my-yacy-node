package main

import (
	"net/http"
	"net/url"
	"time"
)

const outboundRequestTimeout = 30 * time.Second

func newEgressProxyClient(proxyURL *url.URL, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}
}
