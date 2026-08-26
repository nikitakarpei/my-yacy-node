package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/scraperequests/jetstream"
	scraperequeststext "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/scraperequests/text"
	webarchivespywb "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/webarchives/pywb"
)

const (
	indexQueryDeadline = 2 * time.Minute

	msgCapturesSelected        = "archive selected its newest captures"
	msgScrapeRequestsPublished = "scrape requests published"
)

type ScrapeRequestPublisher interface {
	Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error
	Close()
}

func RunCommand(ctx context.Context, cfg CommandConfig, requests io.Writer) error {
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
	slog.InfoContext(ctx, msgCapturesSelected,
		slog.Int("capturesRead", newestReplayURLs.CapturesRead),
		slog.Int("pagesSelected", len(newestReplayURLs.ReplayURLs)),
		slog.Int("pageLimit", cfg.PageLimit),
		slog.Bool("morePages", newestReplayURLs.HasMorePages),
	)

	publisher, err := publisherFor(cfg, requests)
	if err != nil {
		return err
	}
	defer publisher.Close()

	return publishScrapeRequests(ctx, newestReplayURLs.ReplayURLs, publisher)
}

func publisherFor(cfg CommandConfig, requests io.Writer) (ScrapeRequestPublisher, error) {
	if cfg.DryRun {
		return scraperequeststext.New(requests), nil
	}
	return scraperequestsjetstream.Open(cfg.ScrapeRequestNATSURL)
}

func publishScrapeRequests(
	ctx context.Context,
	replayURLs []canonicalurl.CanonicalURL,
	publisher ScrapeRequestPublisher,
) error {
	publishedRequests := 0
	for _, replayURL := range replayURLs {
		if err := publisher.Publish(ctx, replayURL); err != nil {
			return err
		}
		publishedRequests++
	}
	slog.InfoContext(ctx, msgScrapeRequestsPublished,
		slog.Int("scrapeRequests", publishedRequests),
	)
	return nil
}
