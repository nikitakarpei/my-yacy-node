package yacypeerassembly

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

type serviceURLs map[string]string

type reachableServiceAssembly func(
	ctx context.Context,
	getenv func(string) string,
) (service, string, error)

func reachableServicesFor(
	ctx context.Context,
	assemblies []reachableServiceAssembly,
	getenv func(string) string,
) ([]service, serviceURLs, error) {
	services := make([]service, 0, len(assemblies))
	urls := make(serviceURLs, len(assemblies))
	for _, assembly := range assemblies {
		service, url, err := assembly(ctx, getenv)
		if err != nil {
			return nil, nil, errors.Join(err, closeEvery(services))
		}
		services = append(services, service)
		urls[service.name] = url
	}
	return services, urls, nil
}

func reachableServiceAssemblyOf[C any, L io.Closer](
	name string,
	wiring serviceenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
	reachedListenerOf func(L) net.Listener,
) reachableServiceAssembly {
	return func(ctx context.Context, getenv func(string) string) (service, string, error) {
		assembled, err := assembledServiceOf(
			ctx, name, wiring, loadServiceConfig, listenersFor, runService, getenv,
		)
		if err != nil {
			return service{}, "", err
		}
		return assembled.service(), loopback.URLOf(reachedListenerOf(assembled.listeners)), nil
	}
}
