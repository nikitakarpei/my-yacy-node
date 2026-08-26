package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	webarchivespywb "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/webarchives/pywb"
)

const indexQueryDeadline = 2 * time.Minute

func RunCommand(
	ctx context.Context,
	cfg CommandConfig,
	published io.Writer,
	report io.Writer,
) error {
	archive := webarchivespywb.New(
		&http.Client{Timeout: indexQueryDeadline},
		cfg.PywbURL,
		cfg.PywbCollection,
	)
	newestReplayURLs, err := archive.NewestReplayURLsFor(
		ctx,
		cfg.PywbCaptureQueries,
		cfg.PageLimit,
	)
	if err != nil {
		return err
	}
	reportSelectedPages(report, newestReplayURLs)

	publisher, err := openScrapeRequestPublisher(cfg)
	if err != nil {
		return err
	}
	defer publisher.Close()

	return publishScrapeRequests(ctx, newestReplayURLs.ReplayURLs, publisher, published)
}

func publishScrapeRequests(
	ctx context.Context,
	replayURLs []canonicalurl.CanonicalURL,
	publisher scrapeRequestPublisher,
	published io.Writer,
) error {
	for _, replayURL := range replayURLs {
		if err := publisher.Publish(ctx, replayURL); err != nil {
			return fmt.Errorf("publish scrape request for %s: %w", replayURL, err)
		}
		if err := writePublishedPage(published, replayURL); err != nil {
			return err
		}
	}
	return nil
}
