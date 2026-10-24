package yacypeerassembly

import (
	"context"
	"errors"
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
)

type peerAddresses struct {
	egressProxyURL *url.URL
	crawlerAPI     net.Listener
	search         net.Listener
	natsURL        string
}

func peerAddressesFor(
	ctx context.Context,
	natsURL string,
	egress egressRoute,
	getenv func(string) string,
) (peerAddresses, error) {
	crawlerAPI, err := loopback.Listen(ctx)
	if err != nil {
		return peerAddresses{}, err
	}
	search, err := searchListenerFor(ctx, getenv)
	if err != nil {
		return peerAddresses{}, errors.Join(err, crawlerAPI.Close())
	}
	return peerAddresses{
		egressProxyURL: egress.proxyURL,
		crawlerAPI:     crawlerAPI,
		search:         search,
		natsURL:        natsURL,
	}, nil
}

func searchListenerFor(ctx context.Context, getenv func(string) string) (net.Listener, error) {
	searchListenAddr, err := searchListenAddrFor(getenv)
	if err != nil {
		return nil, err
	}
	return servergroup.Listen(ctx, searchListenAddr)
}

func (a peerAddresses) crawlerAPIAddress() string {
	return a.crawlerAPI.Addr().String()
}

func (a peerAddresses) crawlerAPIURL() string {
	return "http://" + a.crawlerAPIAddress()
}

func (a peerAddresses) searchURL() string {
	host, port, _ := net.SplitHostPort(a.search.Addr().String())
	if net.ParseIP(host).IsUnspecified() {
		host = loopback.Host
	}
	return "http://" + net.JoinHostPort(host, port)
}
