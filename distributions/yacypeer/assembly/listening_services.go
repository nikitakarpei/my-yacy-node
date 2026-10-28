package yacypeerassembly

import (
	"context"
	"io"
	"net"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

type serviceURLs map[string]*url.URL

type listeningService struct {
	service service
	url     *url.URL
}

func (s listeningService) Close() error {
	return s.service.Close()
}

func urlsOf(listeningServices []listeningService) serviceURLs {
	urls := make(serviceURLs, len(listeningServices))
	for _, listeningService := range listeningServices {
		urls[listeningService.service.name] = listeningService.url
	}
	return urls
}

func servicesOf(listeningServices []listeningService) []service {
	services := make([]service, 0, len(listeningServices))
	for _, listeningService := range listeningServices {
		services = append(services, listeningService.service)
	}
	return services
}

func listeningServiceAssemblyFrom[C any, L io.Closer](
	name string,
	wiring serviceenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
	listenerOf func(L) net.Listener,
) serviceAssembly[listeningService] {
	return func(ctx context.Context, getenv func(string) string) (listeningService, error) {
		assembledService, err := assembledServiceFrom(
			ctx, name, wiring, loadServiceConfig, listenersFor, runService, getenv,
		)
		if err != nil {
			return listeningService{}, err
		}
		return listeningService{
			service: assembledService.service(),
			url:     loopback.URLOf(listenerOf(assembledService.listeners)),
		}, nil
	}
}
