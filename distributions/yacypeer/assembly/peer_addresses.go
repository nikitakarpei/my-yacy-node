package yacypeerassembly

import (
	"context"
	"errors"
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	visitcrawlassembly "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/assembly"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
)

type peerAddresses struct {
	egressProxyURL *url.URL
	crawlerAPI     net.Listener
	search         net.Listener
	visitIntake    net.Listener
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
	search, err := listenerOf(
		ctx,
		yacydhtsearchService,
		yacydhtsearchassembly.EnvListenAddr,
		getenv,
	)
	if err != nil {
		return peerAddresses{}, errors.Join(err, crawlerAPI.Close())
	}
	visitIntake, err := listenerOf(ctx, visitcrawlService, visitcrawlassembly.EnvListenAddr, getenv)
	if err != nil {
		return peerAddresses{}, errors.Join(err, crawlerAPI.Close(), search.Close())
	}
	return peerAddresses{
		egressProxyURL: egress.proxyURL,
		crawlerAPI:     crawlerAPI,
		search:         search,
		visitIntake:    visitIntake,
		natsURL:        natsURL,
	}, nil
}

func listenerOf(
	ctx context.Context,
	service string,
	listenAddrSetting string,
	getenv func(string) string,
) (net.Listener, error) {
	listenAddr, err := listenAddrOf(service, listenAddrSetting, getenv)
	if err != nil {
		return nil, err
	}
	return servergroup.Listen(ctx, listenAddr)
}

func (a peerAddresses) crawlerAPIAddress() string {
	return a.crawlerAPI.Addr().String()
}

func (a peerAddresses) crawlerAPIURL() string {
	return "http://" + a.crawlerAPIAddress()
}

func (a peerAddresses) searchURL() string {
	return localURLOf(a.search)
}

func (a peerAddresses) visitIntakeURL() string {
	return localURLOf(a.visitIntake)
}

func localURLOf(listener net.Listener) string {
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	if net.ParseIP(host).IsUnspecified() {
		host = loopback.Host
	}
	return "http://" + net.JoinHostPort(host, port)
}
