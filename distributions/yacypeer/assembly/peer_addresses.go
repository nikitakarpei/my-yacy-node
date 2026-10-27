package yacypeerassembly

import "net/url"

type peerAddresses struct {
	egressProxyURL *url.URL
	natsURL        string
}

func peerAddressesFor(natsURL string, egressProxyURL *url.URL) peerAddresses {
	return peerAddresses{egressProxyURL: egressProxyURL, natsURL: natsURL}
}
