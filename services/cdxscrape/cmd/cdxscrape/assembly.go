package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/capturereplay"
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/captureselection"
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/cdxindex"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/scraperequests/jetstream"
	scraperequeststext "github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/scraperequests/text"
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
	index := cdxindex.New(
		&http.Client{Timeout: indexQueryDeadline},
		cfg.CDXURL,
		cfg.Collection,
	)
	captures, err := index.CapturesFor(ctx, cfg.Query)
	if err != nil {
		return err
	}
	newestCaptures := captureselection.NewestCapturesOf(captures)
	slog.InfoContext(ctx, msgCapturesRead,
		slog.Int("captures", len(captures)),
		slog.Int("newestCaptures", len(newestCaptures)),
	)

	publisher, err := publisherFor(cfg, requests)
	if err != nil {
		return err
	}
	defer publisher.Close()

	archive := capturereplay.New(cfg.ReplayURL, cfg.Collection)
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
	archive *capturereplay.Archive,
	captures []cdxindex.Capture,
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
