package yacypeerassembly

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	corpusmarkdownassembly "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/assembly"
	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	crawlconsoleassembly "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/assembly"
	crawlscrapebridgeassembly "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	yacycrawlerassembly "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/assembly"
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
	yacycrawlerassembly.EnvOpsAddr,
	pagescrapeassembly.EnvOpsAddr,
	crawlconsoleassembly.EnvOpsAddr,
	crawlscrapebridgeassembly.EnvOpsAddr,
	yacycrawlerassembly.EnvAPIAddr,
}

const (
	yacynodeService          = "yacynode"
	yacydhtsearchService     = "yacydhtsearch"
	corpustextService        = "corpustext"
	corpusmarkdownService    = "corpusmarkdown"
	yacycrawlerService       = "yacycrawler"
	pagescrapeService        = "pagescrape"
	crawlconsoleService      = "crawlconsole"
	crawlscrapebridgeService = "crawlscrapebridge"
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
	addresses peerAddresses,
	getenv func(string) string,
) ([]service, error) {
	assemblies := append(
		searchServiceAssembliesFor(cfg, addresses),
		crawlServiceAssembliesFor(cfg, addresses)...,
	)
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

func searchServiceAssembliesFor(cfg PeerConfig, addresses peerAddresses) []serviceAssembly {
	return []serviceAssembly{
		serviceAssemblyOf(
			yacynodeService,
			serviceenvironment.Wiring{
				yacynodeassembly.EnvPageOfferNATSURL:  addresses.natsURL,
				yacynodeassembly.EnvPageOfferDurable:  yacynodeService,
				yacynodeassembly.EnvAcceptRemoteIndex: strconv.FormatBool(false),
				yacynodeassembly.EnvDataDir:           cfg.directoryOf(yacynodeService),
				yacynodeassembly.EnvNetworkName:       cfg.NetworkName,
				yacynodeassembly.EnvSeedlistURLs:      cfg.SeedlistURLs,
				yacynodeassembly.EnvEgressProxyURL:    addresses.egressProxyURL.String(),
			},
			yacynodeassembly.LoadServiceConfig,
			yacynodeassembly.ListenersFor,
			yacynodeassembly.RunService,
		),
		serviceAssemblyOf(
			yacydhtsearchService,
			serviceenvironment.Wiring{
				yacydhtsearchassembly.EnvNetworkName:    cfg.NetworkName,
				yacydhtsearchassembly.EnvSeedlistURLs:   cfg.SeedlistURLs,
				yacydhtsearchassembly.EnvEgressProxyURL: addresses.egressProxyURL.String(),
			},
			yacydhtsearchassembly.LoadServiceConfig,
			yacydhtsearchassembly.ListenersFor,
			yacydhtsearchassembly.RunService,
		),
		serviceAssemblyOf(
			corpustextService,
			serviceenvironment.Wiring{
				corpustextassembly.EnvPageOfferNATSURL: addresses.natsURL,
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
				corpusmarkdownassembly.EnvPageOfferNATSURL: addresses.natsURL,
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

func crawlServiceAssembliesFor(cfg PeerConfig, addresses peerAddresses) []serviceAssembly {
	return []serviceAssembly{
		serviceAssemblyOf(
			yacycrawlerService,
			serviceenvironment.Wiring{
				yacycrawlerassembly.EnvCrawlNATSURL:  addresses.natsURL,
				yacycrawlerassembly.EnvCrawlStateDir: cfg.directoryOf(yacycrawlerService),
				yacycrawlerassembly.EnvProxyURL:      addresses.egressProxyURL.String(),
				yacycrawlerassembly.EnvProxyDialMode: yacycrawlerassembly.DefaultProxyDialMode,
			},
			yacycrawlerassembly.LoadServiceConfig,
			crawlerListenersOn(addresses.crawlerAPI),
			yacycrawlerassembly.RunService,
		),
		serviceAssemblyOf(
			pagescrapeService,
			serviceenvironment.Wiring{
				pagescrapeassembly.EnvScrapeNATSURL: addresses.natsURL,
				pagescrapeassembly.EnvProxyURL:      addresses.egressProxyURL.String(),
				pagescrapeassembly.EnvProxyDialMode: pagescrapeassembly.DefaultProxyDialMode,
			},
			pagescrapeassembly.LoadServiceConfig,
			pagescrapeassembly.ListenersFor,
			pagescrapeassembly.RunService,
		),
		serviceAssemblyOf(
			crawlconsoleService,
			serviceenvironment.Wiring{
				crawlconsoleassembly.EnvCrawlerAPIURL: addresses.crawlerAPIURL(),
			},
			crawlconsoleassembly.LoadServiceConfig,
			crawlconsoleassembly.ListenersFor,
			crawlconsoleassembly.RunService,
		),
		serviceAssemblyOf(
			crawlscrapebridgeService,
			serviceenvironment.Wiring{
				crawlscrapebridgeassembly.EnvCrawlNATSURL:         addresses.natsURL,
				crawlscrapebridgeassembly.EnvScrapeRequestNATSURL: addresses.natsURL,
			},
			crawlscrapebridgeassembly.LoadServiceConfig,
			crawlscrapebridgeassembly.ListenersFor,
			crawlscrapebridgeassembly.RunService,
		),
	}
}

func crawlerListenersOn(
	api net.Listener,
) func(context.Context, yacycrawlerassembly.ServiceConfig) (yacycrawlerassembly.Listeners, error) {
	return func(context.Context, yacycrawlerassembly.ServiceConfig) (yacycrawlerassembly.Listeners, error) {
		return yacycrawlerassembly.Listeners{API: api}, nil
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
