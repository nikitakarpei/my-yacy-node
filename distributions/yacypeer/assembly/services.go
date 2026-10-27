package yacypeerassembly

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	crawlconsoleassembly "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/assembly"
	crawlscrapebridgeassembly "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	searchpageassembly "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	visitcrawlassembly "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/assembly"
	yacycrawlerassembly "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/assembly"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

var unreadSettings = []string{
	yacynodeassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvServeProfiler,
	corpustextassembly.EnvOpsAddr,
	yacycrawlerassembly.EnvOpsAddr,
	pagescrapeassembly.EnvOpsAddr,
	crawlconsoleassembly.EnvOpsAddr,
	crawlscrapebridgeassembly.EnvOpsAddr,
	visitcrawlassembly.EnvOpsAddr,
	searchpageassembly.EnvOpsAddr,
}

var localListenAddresses = serviceenvironment.Defaults{
	yacydhtsearchassembly.EnvListenAddr: loopback.Host + yacydhtsearchassembly.DefaultListenAddr,
	corpustextassembly.EnvSearchAddr:    loopback.Host + corpustextassembly.DefaultSearchAddr,
	crawlconsoleassembly.EnvListenAddr:  loopback.Host + crawlconsoleassembly.DefaultListenAddr,
	visitcrawlassembly.EnvListenAddr:    loopback.Host + visitcrawlassembly.DefaultListenAddr,
	searchpageassembly.EnvListenAddr:    loopback.Host + searchpageassembly.DefaultListenAddr,
}

const (
	yacynodeService          = "yacynode"
	yacydhtsearchService     = "yacydhtsearch"
	corpustextService        = "corpustext"
	yacycrawlerService       = "yacycrawler"
	pagescrapeService        = "pagescrape"
	crawlconsoleService      = "crawlconsole"
	crawlscrapebridgeService = "crawlscrapebridge"
	visitcrawlService        = "visitcrawl"
	searchpageService        = "searchpage"
)

type service struct {
	registry  *prometheus.Registry
	listeners io.Closer
	run       func(context.Context) error
	name      string
}

func (s service) Close() error {
	if err := s.listeners.Close(); err != nil {
		return fmt.Errorf("%s: %w", s.name, err)
	}
	return nil
}

type serviceAssembly func(ctx context.Context, getenv func(string) string) (service, error)

func servicesFor(
	ctx context.Context,
	cfg PeerConfig,
	addresses peerAddresses,
	crawler crawler,
	visitLinkSecret string,
	getenv func(string) string,
) ([]service, error) {
	assemblies := serviceAssembliesFor(cfg, addresses, crawler.apiURL)
	services := append(make([]service, 0, len(assemblies)+1), crawler.service)
	for _, assembly := range assemblies {
		service, err := assembly(ctx, getenv)
		if err != nil {
			return nil, errors.Join(err, closeEvery(services))
		}
		services = append(services, service)
	}
	searchPageServices, err := searchPageServicesFor(ctx, cfg, addresses, visitLinkSecret, getenv)
	if err != nil {
		return nil, errors.Join(err, closeEvery(services))
	}
	return append(services, searchPageServices...), nil
}

func serviceAssembliesFor(
	cfg PeerConfig,
	addresses peerAddresses,
	crawlerAPIURL string,
) []serviceAssembly {
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
			yacynodeConfigLoaderFor(cfg),
			yacynodeassembly.ListenersFor,
			yacynodeassembly.RunService,
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
				crawlconsoleassembly.EnvCrawlerAPIURL: crawlerAPIURL,
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

func yacynodeConfigLoaderFor(
	cfg PeerConfig,
) func(func(string) string) (yacynodeassembly.ServiceConfig, error) {
	if !cfg.namesAnEgressProxy() {
		return yacynodeassembly.LoadServiceConfig
	}
	return func(getenv func(string) string) (yacynodeassembly.ServiceConfig, error) {
		if _, err := envconfig.Required(getenv, yacynodeassembly.EnvAdvertiseHost); err != nil {
			return yacynodeassembly.ServiceConfig{}, fmt.Errorf(
				"%w when %s is set",
				err,
				EnvEgressProxyURL,
			)
		}
		return yacynodeassembly.LoadServiceConfig(getenv)
	}
}

func (cfg PeerConfig) directoryOf(serviceName string) string {
	return filepath.Join(cfg.DataDirectory, serviceName)
}

func serviceAssemblyOf[C any, L io.Closer](
	name string,
	wiring serviceenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
) serviceAssembly {
	return func(ctx context.Context, getenv func(string) string) (service, error) {
		assembled, err := assembledServiceOf(
			ctx, name, wiring, loadServiceConfig, listenersFor, runService, getenv,
		)
		if err != nil {
			return service{}, err
		}
		return assembled.service(), nil
	}
}
