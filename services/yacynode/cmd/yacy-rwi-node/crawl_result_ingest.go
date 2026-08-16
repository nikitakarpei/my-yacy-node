package main

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawlbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawlresults"
)

type crawlResultIngest struct {
	broker   *crawlbroker.CrawlBroker
	consumer *crawlresults.IngestConsumer
}

func (i *crawlResultIngest) Run(ctx context.Context) {
	i.consumer.Run(ctx)
}

func (i *crawlResultIngest) Close() {
	i.broker.Close()
}
