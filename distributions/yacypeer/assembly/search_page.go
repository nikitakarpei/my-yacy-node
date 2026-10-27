package yacypeerassembly

import (
	"context"
	"errors"

	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	searchpageassembly "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
	visitcrawlassembly "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/assembly"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

func searchPageServicesFor(
	ctx context.Context,
	cfg PeerConfig,
	addresses peerAddresses,
	visitLinkSecret string,
	getenv func(string) string,
) ([]service, error) {
	yacySearch, err := assembledServiceOf(
		ctx,
		yacydhtsearchService,
		serviceenvironment.Wiring{
			yacydhtsearchassembly.EnvNetworkName:    cfg.NetworkName,
			yacydhtsearchassembly.EnvSeedlistURLs:   cfg.SeedlistURLs,
			yacydhtsearchassembly.EnvEgressProxyURL: addresses.egressProxyURL.String(),
		},
		yacydhtsearchassembly.LoadServiceConfig,
		yacydhtsearchassembly.ListenersFor,
		yacydhtsearchassembly.RunService,
		getenv,
	)
	if err != nil {
		return nil, err
	}
	services := []service{yacySearch.service()}
	corpusText, err := assembledServiceOf(
		ctx,
		corpustextService,
		serviceenvironment.Wiring{
			corpustextassembly.EnvPageOfferNATSURL: addresses.natsURL,
			corpustextassembly.EnvPageOfferDurable: corpustextService,
			corpustextassembly.EnvBleveDirectory:   cfg.directoryOf(corpustextService),
		},
		corpustextassembly.LoadServiceConfig,
		corpustextassembly.ListenersFor,
		corpustextassembly.RunService,
		getenv,
	)
	if err != nil {
		return nil, errors.Join(err, closeEvery(services))
	}
	services = append(services, corpusText.service())
	visitCrawl, err := assembledServiceOf(
		ctx,
		visitcrawlService,
		serviceenvironment.Wiring{
			visitcrawlassembly.EnvCrawlNATSURL: addresses.natsURL,
			visitcrawlassembly.EnvLinkSecret:   visitLinkSecret,
		},
		visitcrawlassembly.LoadServiceConfig,
		visitcrawlassembly.ListenersFor,
		visitcrawlassembly.RunService,
		getenv,
	)
	if err != nil {
		return nil, errors.Join(err, closeEvery(services))
	}
	services = append(services, visitCrawl.service())
	searchPage, err := assembledServiceOf(
		ctx,
		searchpageService,
		serviceenvironment.Wiring{
			searchpageassembly.EnvYaCySearchURL:   loopback.URLOf(yacySearch.listeners.Search),
			searchpageassembly.EnvCorpusTextURL:   loopback.URLOf(corpusText.listeners.Search),
			searchpageassembly.EnvVisitCrawlURL:   loopback.URLOf(visitCrawl.listeners.Intake),
			searchpageassembly.EnvVisitLinkSecret: visitLinkSecret,
		},
		searchpageassembly.LoadServiceConfig,
		searchpageassembly.ListenersFor,
		searchpageassembly.RunService,
		getenv,
	)
	if err != nil {
		return nil, errors.Join(err, closeEvery(services))
	}
	return append(services, searchPage.service()), nil
}
