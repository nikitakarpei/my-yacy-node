package main

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawlbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawlresults"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
)

type crawlRuntime struct {
	broker   *crawlbroker.CrawlBroker
	consumer *crawlresults.IngestConsumer
}

func buildCrawlRuntime(
	ctx context.Context,
	config nodeconfiguration.CrawlConfig,
	storage nodeStorage,
) (*crawlRuntime, error) {
	if !config.Enabled() {
		return nil, nil
	}

	broker, err := crawlbroker.Open(ctx, crawlbroker.Config{
		NATSURL:       config.NATSURL,
		IngestSubject: config.IngestSubject,
		IngestDurable: config.IngestDurable,
	})
	if err != nil {
		return nil, fmt.Errorf("open crawl broker: %w", err)
	}

	consumer := crawlresults.NewIngestConsumer(
		broker.Ingest,
		storage.urlReceiver,
		storage.postingReceiver,
	)

	return &crawlRuntime{
		broker:   broker,
		consumer: consumer,
	}, nil
}

func (r *crawlRuntime) Run(ctx context.Context) {
	r.consumer.Run(ctx)
}

func (r *crawlRuntime) Close() {
	r.broker.Close()
}
