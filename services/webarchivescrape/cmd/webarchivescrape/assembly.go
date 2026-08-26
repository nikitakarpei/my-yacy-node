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
	msgCaptureSkipped          = "capture has no readable replay url, no scrape request published"
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
	newestCaptures, err := archive.NewestCapturesFor(ctx, cfg.PywbQuery, cfg.PageLimit)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, msgCapturesSelected,
		slog.Int("capturesRead", newestCaptures.CapturesRead),
		slog.Int("pagesSelected", len(newestCaptures.Captures)),
		slog.Int("pageLimit", cfg.PageLimit),
		slog.Bool("morePages", newestCaptures.HasMorePages),
	)

	publisher, err := publisherFor(cfg, requests)
	if err != nil {
		return err
	}
	defer publisher.Close()

	return publishScrapeRequests(ctx, archive, newestCaptures.Captures, publisher)
}

func publisherFor(cfg CommandConfig, requests io.Writer) (ScrapeRequestPublisher, error) {
	if cfg.DryRun {
		return scraperequeststext.New(requests), nil
	}
	return scraperequestsjetstream.Open(cfg.ScrapeRequestNATSURL)
}

func publishScrapeRequests(
	ctx context.Context,
	archive *webarchivespywb.Archive,
	captures []webarchivespywb.Capture,
	publisher ScrapeRequestPublisher,
) error {
	publishedRequests := 0
	for _, capture := range captures {
		replayURL, err := archive.ReplayURLOf(capture)
		if err != nil {
			slog.WarnContext(ctx, msgCaptureSkipped,
				slog.String("capturedUrl", capture.OriginalURL),
				slog.Any("error", err),
			)
			continue
		}
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
