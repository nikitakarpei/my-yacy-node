package yacypeerassembly

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"slices"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	crawlconsoleassembly "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/assembly"
	crawlscrapebridgeassembly "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/assembly"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	searchpageassembly "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	visitcrawlassembly "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/assembly"
	yacycrawlerassembly "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/assembly"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
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
	pageFetchUserAgent       = "yacypeer (+https://yacy.net)"
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

type peer struct {
	peerEgressProxyURL *url.URL
	pageCacheProxyURL  string
	cfg                PeerConfig
	natsURL            string
	visitLinkSecret    string
	webUIURLs          webuitabs.URLs
}

type serviceAssembly[S io.Closer] func(ctx context.Context, getenv func(string) string) (S, error)

func (p peer) services(ctx context.Context, getenv func(string) string) ([]service, error) {
	upstreamServices, err := servicesFrom(ctx, p.upstreamServiceAssemblies(), getenv)
	if err != nil {
		return nil, err
	}
	streamOnlyServices, err := servicesFrom(ctx, p.streamOnlyServiceAssemblies(), getenv)
	if err != nil {
		return nil, errors.Join(err, closeEvery(upstreamServices))
	}
	downstreamServices, err := servicesFrom(
		ctx,
		p.downstreamServiceAssembliesWith(urlsOf(upstreamServices)),
		getenv,
	)
	if err != nil {
		return nil, errors.Join(
			err,
			closeEvery(upstreamServices),
			closeEvery(streamOnlyServices),
		)
	}
	return slices.Concat(
		servicesOf(upstreamServices),
		streamOnlyServices,
		downstreamServices,
	), nil
}

func (p peer) upstreamServiceAssemblies() []serviceAssembly[upstreamService] {
	return []serviceAssembly[upstreamService]{
		upstreamServiceAssemblyFrom(
			yacycrawlerService,
			serviceenvironment.Wiring{
				yacycrawlerassembly.EnvAPIAddr:       loopback.AnyFreeAddress,
				yacycrawlerassembly.EnvCrawlNATSURL:  p.natsURL,
				yacycrawlerassembly.EnvCrawlStateDir: p.cfg.directoryOf(yacycrawlerService),
				yacycrawlerassembly.EnvProxyURL:      p.pageCacheProxyURL,
				yacycrawlerassembly.EnvProxyDialMode: pagefetchershttp.ProxyDialAbsoluteURLName,
				yacycrawlerassembly.EnvUserAgent:     pageFetchUserAgent,
			},
			yacycrawlerassembly.LoadServiceConfig,
			yacycrawlerassembly.ListenersFor,
			yacycrawlerassembly.RunService,
			func(listeners yacycrawlerassembly.Listeners) net.Listener { return listeners.API },
		),
		upstreamServiceAssemblyFrom(
			yacydhtsearchService,
			serviceenvironment.Wiring{
				yacydhtsearchassembly.EnvNetworkName:           p.cfg.NetworkName,
				yacydhtsearchassembly.EnvSeedlistURLs:          p.cfg.SeedlistURLs,
				yacydhtsearchassembly.EnvEgressProxyURL:        p.peerEgressProxyURL.String(),
				yacydhtsearchassembly.EnvPageReadProxyURL:      p.pageCacheProxyURL,
				yacydhtsearchassembly.EnvPageReadProxyDialMode: pagefetchershttp.ProxyDialAbsoluteURLName,
				yacydhtsearchassembly.EnvPageReadUserAgent:     pageFetchUserAgent,
			},
			yacydhtsearchassembly.LoadServiceConfig,
			yacydhtsearchassembly.ListenersFor,
			yacydhtsearchassembly.RunService,
			func(listeners yacydhtsearchassembly.Listeners) net.Listener { return listeners.Search },
		),
		upstreamServiceAssemblyFrom(
			corpustextService,
			serviceenvironment.Wiring{
				corpustextassembly.EnvPageOfferNATSURL: p.natsURL,
				corpustextassembly.EnvPageOfferDurable: corpustextService,
				corpustextassembly.EnvBleveDirectory:   p.cfg.directoryOf(corpustextService),
			},
			corpustextassembly.LoadServiceConfig,
			corpustextassembly.ListenersFor,
			corpustextassembly.RunService,
			func(listeners corpustextassembly.Listeners) net.Listener { return listeners.Search },
		),
		upstreamServiceAssemblyFrom(
			visitcrawlService,
			serviceenvironment.Wiring{
				visitcrawlassembly.EnvCrawlNATSURL: p.natsURL,
				visitcrawlassembly.EnvLinkSecret:   p.visitLinkSecret,
			},
			visitcrawlassembly.LoadServiceConfig,
			visitcrawlassembly.ListenersFor,
			visitcrawlassembly.RunService,
			func(listeners visitcrawlassembly.Listeners) net.Listener { return listeners.Intake },
		),
	}
}

func servicesFrom[S io.Closer](
	ctx context.Context,
	assemblies []serviceAssembly[S],
	getenv func(string) string,
) ([]S, error) {
	services := make([]S, 0, len(assemblies))
	for _, assembly := range assemblies {
		service, err := assembly(ctx, getenv)
		if err != nil {
			return nil, errors.Join(err, closeEvery(services))
		}
		services = append(services, service)
	}
	return services, nil
}

func (p peer) streamOnlyServiceAssemblies() []serviceAssembly[service] {
	return []serviceAssembly[service]{
		serviceAssemblyFrom(
			yacynodeService,
			serviceenvironment.Wiring{
				yacynodeassembly.EnvPageOfferNATSURL:  p.natsURL,
				yacynodeassembly.EnvPageOfferDurable:  yacynodeService,
				yacynodeassembly.EnvAcceptRemoteIndex: strconv.FormatBool(false),
				yacynodeassembly.EnvDataDir:           p.cfg.directoryOf(yacynodeService),
				yacynodeassembly.EnvNetworkName:       p.cfg.NetworkName,
				yacynodeassembly.EnvSeedlistURLs:      p.cfg.SeedlistURLs,
				yacynodeassembly.EnvEgressProxyURL:    p.peerEgressProxyURL.String(),
			},
			yacynodeConfigLoaderFor(p.cfg),
			yacynodeassembly.ListenersFor,
			yacynodeassembly.RunService,
		),
		serviceAssemblyFrom(
			pagescrapeService,
			serviceenvironment.Wiring{
				pagescrapeassembly.EnvScrapeNATSURL: p.natsURL,
				pagescrapeassembly.EnvProxyURL:      p.pageCacheProxyURL,
				pagescrapeassembly.EnvProxyDialMode: pagefetchershttp.ProxyDialAbsoluteURLName,
				pagescrapeassembly.EnvUserAgent:     pageFetchUserAgent,
			},
			pagescrapeassembly.LoadServiceConfig,
			pagescrapeassembly.ListenersFor,
			pagescrapeassembly.RunService,
		),
		serviceAssemblyFrom(
			crawlscrapebridgeService,
			serviceenvironment.Wiring{
				crawlscrapebridgeassembly.EnvCrawlNATSURL:         p.natsURL,
				crawlscrapebridgeassembly.EnvScrapeRequestNATSURL: p.natsURL,
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
	if !cfg.namesAPeerProxy() {
		return yacynodeassembly.LoadServiceConfig
	}
	return func(getenv func(string) string) (yacynodeassembly.ServiceConfig, error) {
		if _, err := envconfig.Required(getenv, yacynodeassembly.EnvAdvertiseHost); err != nil {
			return yacynodeassembly.ServiceConfig{}, fmt.Errorf(
				"%w when %s is set",
				err,
				EnvPeerProxyURL,
			)
		}
		return yacynodeassembly.LoadServiceConfig(getenv)
	}
}

func serviceAssemblyFrom[C any, L io.Closer](
	name string,
	wiring serviceenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
) serviceAssembly[service] {
	return func(ctx context.Context, getenv func(string) string) (service, error) {
		assembledService, err := assembledServiceFrom(
			ctx, name, wiring, loadServiceConfig, listenersFor, runService, getenv,
		)
		if err != nil {
			return service{}, err
		}
		return assembledService.service(), nil
	}
}

func (p peer) downstreamServiceAssembliesWith(
	upstreamURLs serviceURLs,
) []serviceAssembly[service] {
	return []serviceAssembly[service]{
		serviceAssemblyFrom(
			crawlconsoleService,
			serviceenvironment.Wiring{
				crawlconsoleassembly.EnvCrawlerAPIURL: upstreamURLs[yacycrawlerService],
				crawlconsoleassembly.EnvTabBar:        p.webUIURLs.SettingFor(webuitabs.Crawls),
			},
			crawlconsoleassembly.LoadServiceConfig,
			crawlconsoleassembly.ListenersFor,
			crawlconsoleassembly.RunService,
		),
		serviceAssemblyFrom(
			searchpageService,
			serviceenvironment.Wiring{
				searchpageassembly.EnvYaCySearchURL:   upstreamURLs[yacydhtsearchService],
				searchpageassembly.EnvCorpusTextURL:   upstreamURLs[corpustextService],
				searchpageassembly.EnvVisitCrawlURL:   upstreamURLs[visitcrawlService],
				searchpageassembly.EnvVisitLinkSecret: p.visitLinkSecret,
				searchpageassembly.EnvTabBar:          p.webUIURLs.SettingFor(webuitabs.Search),
			},
			searchpageassembly.LoadServiceConfig,
			searchpageassembly.ListenersFor,
			searchpageassembly.RunService,
		),
	}
}
