package yacypeerassembly

import (
	"context"
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
)

type peerAddresses struct {
	egressProxyURL *url.URL
	crawlerAPI     net.Listener
	natsURL        string
}

func peerAddressesFor(
	ctx context.Context,
	natsURL string,
	egress egressRoute,
) (peerAddresses, error) {
	crawlerAPI, err := loopback.Listen(ctx)
	if err != nil {
		return peerAddresses{}, err
	}
	return peerAddresses{
		egressProxyURL: egress.proxyURL,
		crawlerAPI:     crawlerAPI,
		natsURL:        natsURL,
	}, nil
}

func (a peerAddresses) crawlerAPIAddress() string {
	return a.crawlerAPI.Addr().String()
}

func (a peerAddresses) crawlerAPIURL() string {
	return "http://" + a.crawlerAPIAddress()
}
