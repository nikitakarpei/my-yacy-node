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
	crawlConsolePathPrefix   = "/" + crawlconsoleService
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
}

type serviceAssembly[S io.Closer] func(ctx context.Context, getenv func(string) string) (S, error)

func (p peer) services(
	ctx context.Context,
	getenv func(string) string,
) ([]service, serviceURLs, error) {
	upstreamServices, err := servicesFrom(ctx, p.upstreamServiceAssemblies(), getenv)
	if err != nil {
		return nil, nil, err
	}
	streamOnlyServices, err := servicesFrom(ctx, p.streamOnlyServiceAssemblies(), getenv)
	if err != nil {
		return nil, nil, errors.Join(err, closeEvery(upstreamServices))
	}
	webUIServices, err := servicesFrom(
		ctx,
		p.webUIServiceAssembliesWith(urlsOf(upstreamServices)),
		getenv,
	)
	if err != nil {
		return nil, nil, errors.Join(
			err,
			closeEvery(upstreamServices),
			closeEvery(streamOnlyServices),
		)
	}
	listeningServices := slices.Concat(upstreamServices, webUIServices)
	return slices.Concat(
		servicesOf(listeningServices),
		streamOnlyServices,
	), urlsOf(listeningServices), nil
}

func (p peer) upstreamServiceAssemblies() []serviceAssembly[listeningService] {
	return []serviceAssembly[listeningService]{
		listeningServiceAssemblyFrom(
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
		listeningServiceAssemblyFrom(
			yacydhtsearchService,
			serviceenvironment.Wiring{
				yacydhtsearchassembly.EnvListenAddr:            loopback.AnyFreeAddress,
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
		listeningServiceAssemblyFrom(
			corpustextService,
			serviceenvironment.Wiring{
				corpustextassembly.EnvSearchAddr:       loopback.AnyFreeAddress,
				corpustextassembly.EnvPageOfferNATSURL: p.natsURL,
				corpustextassembly.EnvPageOfferDurable: corpustextService,
				corpustextassembly.EnvBleveDirectory:   p.cfg.directoryOf(corpustextService),
			},
			corpustextassembly.LoadServiceConfig,
			corpustextassembly.ListenersFor,
			corpustextassembly.RunService,
			func(listeners corpustextassembly.Listeners) net.Listener { return listeners.Search },
		),
		listeningServiceAssemblyFrom(
			visitcrawlService,
			serviceenvironment.Wiring{
				visitcrawlassembly.EnvListenAddr:   loopback.AnyFreeAddress,
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
		streamOnlyServiceAssemblyFrom(
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
		streamOnlyServiceAssemblyFrom(
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
		streamOnlyServiceAssemblyFrom(
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

func streamOnlyServiceAssemblyFrom[C any, L io.Closer](
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

func (p peer) webUIServiceAssembliesWith(
	upstreamURLs serviceURLs,
) []serviceAssembly[listeningService] {
	return []serviceAssembly[listeningService]{
		listeningServiceAssemblyFrom(
			crawlconsoleService,
			serviceenvironment.Wiring{
				crawlconsoleassembly.EnvListenAddr:    loopback.AnyFreeAddress,
				crawlconsoleassembly.EnvPathPrefix:    crawlConsolePathPrefix,
				crawlconsoleassembly.EnvCrawlerAPIURL: upstreamURLs[yacycrawlerService].String(),
				crawlconsoleassembly.EnvTabBar:        webUIURLs.SettingFor(webuitabs.Crawls),
			},
			crawlconsoleassembly.LoadServiceConfig,
			crawlconsoleassembly.ListenersFor,
			crawlconsoleassembly.RunService,
			func(listeners crawlconsoleassembly.Listeners) net.Listener { return listeners.Console },
		),
		listeningServiceAssemblyFrom(
			searchpageService,
			serviceenvironment.Wiring{
				searchpageassembly.EnvListenAddr:      loopback.AnyFreeAddress,
				searchpageassembly.EnvYaCySearchURL:   upstreamURLs[yacydhtsearchService].String(),
				searchpageassembly.EnvCorpusTextURL:   upstreamURLs[corpustextService].String(),
				searchpageassembly.EnvVisitCrawlURL:   upstreamURLs[visitcrawlService].String(),
				searchpageassembly.EnvVisitLinkSecret: p.visitLinkSecret,
				searchpageassembly.EnvTabBar:          webUIURLs.SettingFor(webuitabs.Search),
			},
			searchpageassembly.LoadServiceConfig,
			searchpageassembly.ListenersFor,
			searchpageassembly.RunService,
			func(listeners searchpageassembly.Listeners) net.Listener { return listeners.Page },
		),
	}
}
