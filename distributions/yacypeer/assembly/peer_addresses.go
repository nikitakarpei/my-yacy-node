package yacypeerassembly

import "net/url"

type peerAddresses struct {
	egressProxyURL *url.URL
	natsURL        string
}

func peerAddressesFor(natsURL string, egress egressRoute) peerAddresses {
	return peerAddresses{egressProxyURL: egress.proxyURL(), natsURL: natsURL}
}
