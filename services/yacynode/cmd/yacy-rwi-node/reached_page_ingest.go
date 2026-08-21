package main

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawlbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/reachedpageintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type reachedPageIngest struct {
	broker   *crawlbroker.CrawlBroker
	consumer *reachedpageintake.ReachedPageConsumer
}

func openReachedPageIngest(
	ctx context.Context,
	config nodeconfiguration.CrawlConfig,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
) (*reachedPageIngest, error) {
	broker, err := crawlbroker.Open(ctx, crawlbroker.Config{
		NATSURL:            config.NATSURL,
		ReachedPageSubject: config.ReachedPageSubject,
		ReachedPageDurable: config.ReachedPageDurable,
		Concurrency:        config.Concurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("open crawl broker: %w", err)
	}
	scraper, err := pageScraperFor(config)
	if err != nil {
		broker.Close()

		return nil, err
	}

	return &reachedPageIngest{
		broker: broker,
		consumer: reachedpageintake.NewReachedPageConsumer(reachedpageintake.Config{
			Source:          broker.ReachedPages,
			Scraper:         scraper,
			URLs:            urls,
			Postings:        postings,
			PostingBatchCap: postingAdmissionBatchCapacity,
			Concurrency:     config.Concurrency,
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
		contentformatgraph.FormatFullText,
	)
	if err != nil {
		return nil, fmt.Errorf("build page scraper: %w", err)
	}

	return scraper, nil
}

func (i *reachedPageIngest) Run(ctx context.Context) error {
	return i.consumer.Run(ctx)
}

func (i *reachedPageIngest) Close() {
	i.broker.Close()
}
