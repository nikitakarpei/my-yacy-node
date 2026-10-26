package yacypeerassembly

import (
	"context"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

type egressRoute struct {
	builtInProxy *egressproxy.EgressProxy
	proxyURL     *url.URL
}

func egressRouteFor(ctx context.Context, cfg PeerConfig) (egressRoute, error) {
	if cfg.namesAnEgressProxy() {
		return egressRoute{proxyURL: cfg.EgressProxyURL}, nil
	}
	builtInProxy, err := egressproxy.Listen(ctx)
	if err != nil {
		return egressRoute{}, err
	}
	return egressRoute{builtInProxy: builtInProxy, proxyURL: builtInProxy.URL()}, nil
}

func (r egressRoute) Close() error {
	if r.builtInProxy == nil {
		return nil
	}
	return r.builtInProxy.Close()
}

func (r egressRoute) runs() []func(context.Context) error {
	if r.builtInProxy == nil {
		return nil
	}
	return []func(context.Context) error{func(ctx context.Context) error {
		return r.builtInProxy.Run(servicelog.ContextFor(ctx, egressproxy.Name))
	}}
}
