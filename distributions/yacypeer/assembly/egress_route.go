package yacypeerassembly

import (
	"context"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

type egressRoute struct {
	builtInProxy *egressproxy.EgressProxy
}

func egressRouteFor(ctx context.Context, cfg PeerConfig) (egressRoute, error) {
	builtInProxy, err := egressproxy.Listen(ctx, cfg.EgressProxyURL)
	if err != nil {
		return egressRoute{}, err
	}
	return egressRoute{builtInProxy: builtInProxy}, nil
}

func (r egressRoute) proxyURL() *url.URL {
	return r.builtInProxy.URL()
}

func (r egressRoute) Close() error {
	return r.builtInProxy.Close()
}

func (r egressRoute) runs() []func(context.Context) error {
	return []func(context.Context) error{func(ctx context.Context) error {
		return r.builtInProxy.Run(servicelog.ContextFor(ctx, egressproxy.Name))
	}}
}
