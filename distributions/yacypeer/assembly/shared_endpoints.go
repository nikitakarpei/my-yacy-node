package yacypeerassembly

import (
	"context"
	"net"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

type sharedEndpoints struct {
	builtInEgressProxy *egressproxy.EgressProxy
	egressProxyURL     *url.URL
	crawlerAPI         net.Listener
	natsURL            string
}

func sharedEndpointsFor(
	ctx context.Context,
	cfg PeerConfig,
	natsURL string,
) (sharedEndpoints, error) {
	crawlerAPI, err := loopback.Listen(ctx)
	if err != nil {
		return sharedEndpoints{}, err
	}
	endpoints := sharedEndpoints{
		egressProxyURL: cfg.EgressProxyURL,
		crawlerAPI:     crawlerAPI,
		natsURL:        natsURL,
	}
	if cfg.namesAnEgressProxy() {
		return endpoints, nil
	}
	return endpoints.withBuiltInEgressProxy(ctx)
}

func (e sharedEndpoints) withBuiltInEgressProxy(ctx context.Context) (sharedEndpoints, error) {
	egressProxy, err := egressproxy.Listen(ctx)
	if err != nil {
		return sharedEndpoints{}, err
	}
	e.builtInEgressProxy = egressProxy
	e.egressProxyURL = egressProxy.URL()
	return e, nil
}

func (e sharedEndpoints) crawlerAPIAddress() string {
	return e.crawlerAPI.Addr().String()
}

func (e sharedEndpoints) crawlerAPIURL() string {
	return "http://" + e.crawlerAPIAddress()
}

func (e sharedEndpoints) runs() []func(context.Context) error {
	if e.builtInEgressProxy == nil {
		return nil
	}
	return []func(context.Context) error{func(ctx context.Context) error {
		return e.builtInEgressProxy.Run(servicelog.ContextFor(ctx, egressproxy.Name))
	}}
}
