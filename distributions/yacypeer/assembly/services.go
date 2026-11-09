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
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	crawlconsoleassembly "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/assembly"
	crawlscrapebridgeassembly "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/assembly"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	searchpageassembly "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	visitcrawlassembly "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/assembly"
	yacycrawlerassembly "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/assembly"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

var unreadSettings = []string{
	yacycrawlerassembly.EnvAPIAddr,
	yacydhtsearchassembly.EnvListenAddr,
	corpustextassembly.EnvSearchAddr,
	visitcrawlassembly.EnvListenAddr,
	crawlconsoleassembly.EnvListenAddr,
	searchpageassembly.EnvListenAddr,
	yacynodeassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvServeProfiler,
	corpustextassembly.EnvOpsAddr,
	yacycrawlerassembly.EnvOpsAddr,
	pagescrapeassembly.EnvOpsAddr,
	pagerwiassembly.EnvOpsAddr,
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
	pagerwiService           = "pagerwi"
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
	network            *processnetwork.Network
	visitLinkSecret    string
	peerKey            yacymodel.PeerKey
	webUILinks         webuitabs.Links
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
	downstreamServices, err := servicesFrom(
		ctx,
		p.downstreamServiceAssembliesWith(urlsOf(upstreamServices)),
		getenv,
	)
	if err != nil {
		return nil, nil, errors.Join(
			err,
			closeEvery(upstreamServices),
			closeEvery(streamOnlyServices),
		)
	}
	listeningServices := slices.Concat(upstreamServices, downstreamServices)
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
				yacycrawlerassembly.EnvCrawlNATSURL:  p.natsURL,
				yacycrawlerassembly.EnvCrawlStateDir: p.cfg.directoryOf(yacycrawlerService),
				yacycrawlerassembly.EnvProxyURL:      p.pageCacheProxyURL,
				yacycrawlerassembly.EnvProxyDialMode: pagefetchershttp.ProxyDialAbsoluteURLName,
				yacycrawlerassembly.EnvUserAgent:     pageFetchUserAgent,
			},
			yacycrawlerassembly.LoadServiceConfig,
			yacycrawlerassembly.RunService,
			func(listener net.Listener) yacycrawlerassembly.Listeners {
				return yacycrawlerassembly.Listeners{API: listener}
			},
			p.network,
		),
		listeningServiceAssemblyFrom(
			yacydhtsearchService,
			serviceenvironment.Wiring{
				yacydhtsearchassembly.EnvNetworkName:           p.cfg.NetworkName,
				yacydhtsearchassembly.EnvPartitionExponent:     p.cfg.partitionExponentSetting(),
				yacydhtsearchassembly.EnvSeedlistURLs:          p.cfg.SeedlistURLs,
				yacydhtsearchassembly.EnvEgressProxyURL:        p.peerEgressProxyURL.String(),
				yacydhtsearchassembly.EnvPageReadProxyURL:      p.pageCacheProxyURL,
				yacydhtsearchassembly.EnvPageReadProxyDialMode: pagefetchershttp.ProxyDialAbsoluteURLName,
				yacydhtsearchassembly.EnvPageReadUserAgent:     pageFetchUserAgent,
			},
			yacydhtsearchassembly.LoadServiceConfig,
			yacydhtsearchassembly.RunService,
			func(listener net.Listener) yacydhtsearchassembly.Listeners {
				return yacydhtsearchassembly.Listeners{Search: listener}
			},
			p.network,
		),
		listeningServiceAssemblyFrom(
			corpustextService,
			serviceenvironment.Wiring{
				corpustextassembly.EnvPageOfferNATSURL: p.natsURL,
				corpustextassembly.EnvPageOfferDurable: corpustextService,
				corpustextassembly.EnvBleveDirectory:   p.cfg.directoryOf(corpustextService),
			},
			corpustextassembly.LoadServiceConfig,
			corpustextassembly.RunService,
			func(listener net.Listener) corpustextassembly.Listeners {
				return corpustextassembly.Listeners{Search: listener}
			},
			p.network,
		),
		listeningServiceAssemblyFrom(
			visitcrawlService,
			serviceenvironment.Wiring{
				visitcrawlassembly.EnvCrawlNATSURL: p.natsURL,
				visitcrawlassembly.EnvLinkSecret:   p.visitLinkSecret,
			},
			visitcrawlassembly.LoadServiceConfig,
			visitcrawlassembly.RunService,
			func(listener net.Listener) visitcrawlassembly.Listeners {
				return visitcrawlassembly.Listeners{Intake: listener}
			},
			p.network,
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
				yacynodeassembly.EnvPeerKey:           secretTextOf(p.peerKey),
				yacynodeassembly.EnvPageRWINATSURL:    p.natsURL,
				yacynodeassembly.EnvPageRWIDurable:    yacynodeService,
				yacynodeassembly.EnvAcceptRemoteIndex: strconv.FormatBool(false),
				yacynodeassembly.EnvDataDir: p.cfg.directoryOf(
					yacynodeService,
				),
				yacynodeassembly.EnvNetworkName:                   p.cfg.NetworkName,
				yacynodeassembly.EnvDistributionPartitionExponent: p.cfg.partitionExponentSetting(),
				yacynodeassembly.EnvSeedlistURLs:                  p.cfg.SeedlistURLs,
				yacynodeassembly.EnvEgressProxyURL:                p.peerEgressProxyURL.String(),
			},
			yacynodeConfigLoaderFor(p.cfg),
			yacynodeassembly.ListenersFor,
			yacynodeassembly.RunService,
			p.network,
		),
		streamOnlyServiceAssemblyFrom(
			pagescrapeService,
			serviceenvironment.Wiring{
				pagescrapeassembly.EnvScrapeNATSURL:    p.natsURL,
				pagescrapeassembly.EnvProxyURL:         p.pageCacheProxyURL,
				pagescrapeassembly.EnvProxyDialMode:    pagefetchershttp.ProxyDialAbsoluteURLName,
				pagescrapeassembly.EnvUserAgent:        pageFetchUserAgent,
				pagescrapeassembly.EnvProvisionStreams: strconv.FormatBool(true),
			},
			pagescrapeassembly.LoadServiceConfig,
			pagescrapeassembly.ListenersFor,
			runPagescrape,
			p.network,
		),
		streamOnlyServiceAssemblyFrom(
			pagerwiService,
			serviceenvironment.Wiring{
				pagerwiassembly.EnvPageOfferNATSURL:         p.natsURL,
				pagerwiassembly.EnvPageOfferDurable:         pagerwiService,
				pagerwiassembly.EnvPageRWINATSURL:           p.natsURL,
				pagerwiassembly.EnvPageRWIPartitionExponent: p.cfg.partitionExponentSetting(),
				pagerwiassembly.EnvProvisionStreams:         strconv.FormatBool(true),
			},
			pagerwiassembly.LoadServiceConfig,
			pagerwiassembly.ListenersFor,
			runPagerwi,
			p.network,
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
			p.network,
		),
	}
}

func secretTextOf(key yacymodel.PeerKey) string {
	var secretText strings.Builder
	_ = key.WriteSecret(&secretText)

	return secretText.String()
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
	runService func(context.Context, C, L, dial, *prometheus.Registry) error,
	network *processnetwork.Network,
) serviceAssembly[service] {
	return func(ctx context.Context, getenv func(string) string) (service, error) {
		assembledService, err := assembledServiceFrom(
			ctx,
			name,
			wiring,
			loadServiceConfig,
			listenersFor,
			runService,
			network.DialContext,
			getenv,
		)
		if err != nil {
			return service{}, err
		}
		return assembledService.service(), nil
	}
}

func runPagescrape(
	ctx context.Context,
	cfg pagescrapeassembly.ServiceConfig,
	listeners pagescrapeassembly.Listeners,
	dial dial,
	registry *prometheus.Registry,
) error {
	return pagescrapeassembly.RunService(
		ctx, cfg, listeners, dial,
		jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
		registry, &servicehealth.Health{},
	)
}

func runPagerwi(
	ctx context.Context,
	cfg pagerwiassembly.ServiceConfig,
	listeners pagerwiassembly.Listeners,
	dial dial,
	registry *prometheus.Registry,
) error {
	return pagerwiassembly.RunService(
		ctx,
		cfg,
		listeners,
		dial,
		jetstreamconnect.BrokerAccess{},
		jetstreamconnect.BrokerAccess{},
		pullintake.EveryItemAdmitted{},
		registry,
		&servicehealth.Health{},
	)
}

func (p peer) downstreamServiceAssembliesWith(
	upstreamURLs serviceURLs,
) []serviceAssembly[listeningService] {
	return []serviceAssembly[listeningService]{
		listeningServiceAssemblyFrom(
			crawlconsoleService,
			serviceenvironment.Wiring{
				crawlconsoleassembly.EnvPathPrefix:    crawlConsolePathPrefix,
				crawlconsoleassembly.EnvCrawlerAPIURL: upstreamURLs[yacycrawlerService].String(),
				crawlconsoleassembly.EnvTabBar:        p.webUILinks.SettingFor(webuitabs.Crawls),
			},
			crawlconsoleassembly.LoadServiceConfig,
			crawlconsoleassembly.RunService,
			func(listener net.Listener) crawlconsoleassembly.Listeners {
				return crawlconsoleassembly.Listeners{Console: listener}
			},
			p.network,
		),
		listeningServiceAssemblyFrom(
			searchpageService,
			serviceenvironment.Wiring{
				searchpageassembly.EnvYaCySearchURL:   upstreamURLs[yacydhtsearchService].String(),
				searchpageassembly.EnvCorpusTextURL:   upstreamURLs[corpustextService].String(),
				searchpageassembly.EnvVisitCrawlURL:   upstreamURLs[visitcrawlService].String(),
				searchpageassembly.EnvVisitLinkSecret: p.visitLinkSecret,
				searchpageassembly.EnvTabBar:          p.webUILinks.SettingFor(webuitabs.Search),
			},
			searchpageassembly.LoadServiceConfig,
			searchpageassembly.RunService,
			func(listener net.Listener) searchpageassembly.Listeners {
				return searchpageassembly.Listeners{Page: listener}
			},
			p.network,
		),
	}
}
