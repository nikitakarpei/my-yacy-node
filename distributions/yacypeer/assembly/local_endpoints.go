package yacypeerassembly

import (
	"context"
	"net"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

type localEndpoints struct {
	egressProxy *egressproxy.EgressProxy
	crawlerAPI  net.Listener
	natsURL     string
}

func localEndpointsOn(ctx context.Context, natsURL string) (localEndpoints, error) {
	egressProxy, err := egressproxy.Listen(ctx)
	if err != nil {
		return localEndpoints{}, err
	}
	crawlerAPI, err := loopback.Listen(ctx)
	if err != nil {
		return localEndpoints{}, err
	}
	return localEndpoints{egressProxy: egressProxy, crawlerAPI: crawlerAPI, natsURL: natsURL}, nil
}

func (e localEndpoints) egressProxyURL() string {
	return e.egressProxy.URL()
}

func (e localEndpoints) crawlerAPIAddress() string {
	return e.crawlerAPI.Addr().String()
}

func (e localEndpoints) crawlerAPIURL() string {
	return "http://" + e.crawlerAPIAddress()
}

func (e localEndpoints) runEgressProxy(ctx context.Context) error {
	return e.egressProxy.Run(servicelog.ContextFor(ctx, egressproxy.Name))
}
