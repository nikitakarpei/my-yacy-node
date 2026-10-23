package yacypeerassembly

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	corpusmarkdownassembly "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/assembly"
	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

var unreadSettings = []string{
	yacynodeassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvServeProfiler,
	corpustextassembly.EnvOpsAddr,
	corpusmarkdownassembly.EnvOpsAddr,
}

const (
	yacynodeService       = "yacynode"
	yacydhtsearchService  = "yacydhtsearch"
	corpustextService     = "corpustext"
	corpusmarkdownService = "corpusmarkdown"
)

type service struct {
	registry *prometheus.Registry
	run      func(context.Context) error
	name     string
}

type serviceAssembly func(ctx context.Context, getenv func(string) string) (service, error)

func servicesFor(
	ctx context.Context,
	cfg PeerConfig,
	natsURL string,
	getenv func(string) string,
) ([]service, error) {
	assemblies := serviceAssembliesFor(cfg, natsURL)
	services := make([]service, 0, len(assemblies))
	for _, assembly := range assemblies {
		service, err := assembly(ctx, getenv)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, nil
}

func serviceAssembliesFor(cfg PeerConfig, natsURL string) []serviceAssembly {
	return []serviceAssembly{
		serviceAssemblyOf(
			yacynodeService,
			serviceenvironment.Wiring{
				yacynodeassembly.EnvPageOfferNATSURL:  natsURL,
				yacynodeassembly.EnvPageOfferDurable:  yacynodeService,
				yacynodeassembly.EnvAcceptRemoteIndex: strconv.FormatBool(false),
				yacynodeassembly.EnvDataDir:           cfg.directoryOf(yacynodeService),
				yacynodeassembly.EnvNetworkName:       cfg.NetworkName,
				yacynodeassembly.EnvSeedlistURLs:      cfg.SeedlistURLs,
			},
			yacynodeassembly.LoadServiceConfig,
			yacynodeassembly.ListenersFor,
			yacynodeassembly.RunService,
		),
		serviceAssemblyOf(
			yacydhtsearchService,
			serviceenvironment.Wiring{
				yacydhtsearchassembly.EnvNetworkName:  cfg.NetworkName,
				yacydhtsearchassembly.EnvSeedlistURLs: cfg.SeedlistURLs,
			},
			yacydhtsearchassembly.LoadServiceConfig,
			yacydhtsearchassembly.ListenersFor,
			yacydhtsearchassembly.RunService,
		),
		serviceAssemblyOf(
			corpustextService,
			serviceenvironment.Wiring{
				corpustextassembly.EnvPageOfferNATSURL: natsURL,
				corpustextassembly.EnvPageOfferDurable: corpustextService,
				corpustextassembly.EnvBleveDirectory:   cfg.directoryOf(corpustextService),
			},
			corpustextassembly.LoadServiceConfig,
			corpustextassembly.ListenersFor,
			corpustextassembly.RunService,
		),
		serviceAssemblyOf(
			corpusmarkdownService,
			serviceenvironment.Wiring{
				corpusmarkdownassembly.EnvPageOfferNATSURL: natsURL,
				corpusmarkdownassembly.EnvPageOfferDurable: corpusmarkdownService,
				corpusmarkdownassembly.EnvPageMarkdownDirectory: cfg.directoryOf(
					corpusmarkdownService,
				),
			},
			corpusmarkdownassembly.LoadServiceConfig,
			corpusmarkdownassembly.ListenersFor,
			corpusmarkdownassembly.RunService,
		),
	}
}

func (cfg PeerConfig) directoryOf(serviceName string) string {
	return filepath.Join(cfg.DataDirectory, serviceName)
}

func serviceAssemblyOf[C, L any](
	name string,
	wiring serviceenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
) serviceAssembly {
	return func(ctx context.Context, getenv func(string) string) (service, error) {
		environment, err := serviceenvironment.For(name, wiring, unreadSettings, getenv)
		if err != nil {
			return service{}, err
		}
		cfg, err := loadServiceConfig(environment.Lookup)
		if err != nil {
			return service{}, fmt.Errorf("%s: %w", name, err)
		}
		listeners, err := listenersFor(ctx, cfg)
		if err != nil {
			return service{}, fmt.Errorf("%s: %w", name, err)
		}
		registry := prometheus.NewRegistry()
		return service{
			registry: registry,
			run: func(ctx context.Context) error {
				return runService(servicelog.ContextFor(ctx, name), cfg, listeners, registry)
			},
			name: name,
		}, nil
	}
}
