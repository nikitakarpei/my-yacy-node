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

	msgCapturesRead            = "archive listed its captures"
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
	captures, err := archive.CapturesFor(ctx, cfg.PywbQuery)
	if err != nil {
		return err
	}
	newestCaptures := webarchivespywb.NewestCapturesOf(captures)
	slog.InfoContext(ctx, msgCapturesRead,
		slog.Int("captures", len(captures)),
		slog.Int("newestCaptures", len(newestCaptures)),
	)

	publisher, err := publisherFor(cfg, requests)
	if err != nil {
		return err
	}
	defer publisher.Close()

	return publishScrapeRequests(ctx, archive, newestCaptures, publisher)
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
