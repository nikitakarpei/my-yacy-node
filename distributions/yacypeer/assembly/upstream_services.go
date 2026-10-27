package yacypeerassembly

import (
	"context"
	"io"
	"net"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

type serviceURLs map[string]string

type upstreamService struct {
	service service
	url     string
}

func (s upstreamService) Close() error {
	return s.service.Close()
}

func urlsOf(upstreamServices []upstreamService) serviceURLs {
	urls := make(serviceURLs, len(upstreamServices))
	for _, upstreamService := range upstreamServices {
		urls[upstreamService.service.name] = upstreamService.url
	}
	return urls
}

func servicesOf(upstreamServices []upstreamService) []service {
	services := make([]service, 0, len(upstreamServices))
	for _, upstreamService := range upstreamServices {
		services = append(services, upstreamService.service)
	}
	return services
}

func upstreamServiceAssemblyFrom[C any, L io.Closer](
	name string,
	wiring serviceenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
	upstreamListenerOf func(L) net.Listener,
) serviceAssembly[upstreamService] {
	return func(ctx context.Context, getenv func(string) string) (upstreamService, error) {
		assembledService, err := assembledServiceFrom(
			ctx, name, wiring, loadServiceConfig, listenersFor, runService, getenv,
		)
		if err != nil {
			return upstreamService{}, err
		}
		return upstreamService{
			service: assembledService.service(),
			url:     loopback.URLOf(upstreamListenerOf(assembledService.listeners)),
		}, nil
	}
}
