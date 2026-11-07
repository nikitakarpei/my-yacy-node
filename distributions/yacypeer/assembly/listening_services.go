package yacypeerassembly

import (
	"context"
	"io"
	"net"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
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
	runService func(context.Context, C, L, dial, *prometheus.Registry) error,
	listenersOf func(net.Listener) L,
	network *processnetwork.Network,
) serviceAssembly[listeningService] {
	return func(ctx context.Context, getenv func(string) string) (listeningService, error) {
		assembledService, err := assembledServiceFrom(
			ctx,
			name,
			wiring,
			loadServiceConfig,
			func(context.Context, C) (L, error) {
				return listenersOf(network.ListenerAt(hostOf(name))), nil
			},
			runService,
			network.DialContext,
			getenv,
		)
		if err != nil {
			return listeningService{}, err
		}
		return listeningService{
			service: assembledService.service(),
			url:     &url.URL{Scheme: "http", Host: hostOf(name)},
		}, nil
	}
}
