package main

import (
	"context"
	"fmt"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scraperequestbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scraperequestintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type scrapeRequestIngest struct {
	broker   *scraperequestbroker.ScrapeRequestBroker
	consumer *scraperequestintake.ScrapeRequestConsumer
}

func openScrapeRequestIngest(
	ctx context.Context,
	config nodeconfiguration.ScrapeRequestIngestConfig,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
) (*scrapeRequestIngest, error) {
	broker, err := scraperequestbroker.Open(ctx, scraperequestbroker.Config{
		ScrapeRequestNATSURL: config.ScrapeRequestNATSURL,
		ScrapeRequestSubject: config.ScrapeRequestSubject,
		ScrapeRequestDurable: config.ScrapeRequestDurable,
		Concurrency:          config.Concurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("open scrape request broker: %w", err)
	}
	derivations, err := pageformats.New()
	if err != nil {
		broker.Close()

		return nil, err
	}

	return &scrapeRequestIngest{
		broker: broker,
		consumer: scraperequestintake.NewScrapeRequestConsumer(scraperequestintake.Config{
			Source: broker.ScrapeRequests,
			Fetcher: pagefetchershttp.New(
				config.ProxyURL,
				config.ProxyDialMode,
				config.UserAgent,
				config.MaxBodyBytes,
				config.FetchDeadline,
			),
			Derivations: derivations,
			URLs:        urls,
			Postings:    postings,
			Concurrency: config.Concurrency,
		}),
	}, nil
}

func (i *scrapeRequestIngest) Run(ctx context.Context) error {
	return i.consumer.Run(ctx)
}

func (i *scrapeRequestIngest) Close() {
	i.broker.Close()
}
