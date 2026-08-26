package main

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/scraperequests/jetstream"
	scraperequestsnoop "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/scraperequests/noop"
)

type scrapeRequestPublisher interface {
	Publish(ctx context.Context, replayURL canonicalurl.CanonicalURL) error
	Close()
}

func openScrapeRequestPublisher(cfg CommandConfig) (scrapeRequestPublisher, error) {
	if cfg.DryRun {
		return scraperequestsnoop.Open(), nil
	}
	return scraperequestsjetstream.Open(cfg.ScrapeRequestNATSURL)
}
