package main

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawlbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scraperequestintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type scrapeRequestIngest struct {
	broker   *crawlbroker.CrawlBroker
	consumer *scraperequestintake.ScrapeRequestConsumer
}

func openScrapeRequestIngest(
	ctx context.Context,
	config nodeconfiguration.CrawlConfig,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
) (*scrapeRequestIngest, error) {
	broker, err := crawlbroker.Open(ctx, crawlbroker.Config{
		ScrapeRequestNATSURL: config.ScrapeRequestNATSURL,
		ScrapeRequestSubject: config.ScrapeRequestSubject,
		ScrapeRequestDurable: config.ScrapeRequestDurable,
		Concurrency:          config.Concurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("open crawl broker: %w", err)
	}
	scraper, err := pageScraperFor(config)
	if err != nil {
		broker.Close()

		return nil, err
	}

	return &scrapeRequestIngest{
		broker: broker,
		consumer: scraperequestintake.NewScrapeRequestConsumer(scraperequestintake.Config{
			Source:      broker.ScrapeRequests,
			Scraper:     scraper,
			URLs:        urls,
			Postings:    postings,
			Concurrency: config.Concurrency,
		}),
	}, nil
}

func pageScraperFor(config nodeconfiguration.CrawlConfig) (*pagescrape.Scraper, error) {
	scraper, err := pagescrape.New(
		pagefetchershttp.New(
			config.ProxyURL,
			config.ProxyDialMode,
			config.UserAgent,
			config.MaxBodyBytes,
			config.FetchDeadline,
		),
		documentextraction.FormatFullText,
	)
	if err != nil {
		return nil, fmt.Errorf("build page scraper: %w", err)
	}

	return scraper, nil
}

func (i *scrapeRequestIngest) Run(ctx context.Context) error {
	return i.consumer.Run(ctx)
}

func (i *scrapeRequestIngest) Close() {
	i.broker.Close()
}
