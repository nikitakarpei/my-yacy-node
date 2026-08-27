package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

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
	newestArchivedPages, err := archive.NewestArchivedPagesFor(
		ctx,
		cfg.PywbCaptureQueries,
		cfg.PageLimit,
	)
	if err != nil {
		return err
	}
	reportSelectedPages(report, newestArchivedPages)

	publisher, err := openScrapeRequestPublisher(cfg)
	if err != nil {
		return err
	}
	defer publisher.Close()

	return publishScrapeRequests(ctx, newestArchivedPages.ArchivedPages, publisher, published)
}

func publishScrapeRequests(
	ctx context.Context,
	archivedPages []webarchivespywb.ArchivedPage,
	publisher scrapeRequestPublisher,
	published io.Writer,
) error {
	for _, archivedPage := range archivedPages {
		if err := publisher.Publish(ctx, archivedPage.PageURL, archivedPage.ReplayURL); err != nil {
			return fmt.Errorf("publish scrape request for %s: %w", archivedPage.PageURL, err)
		}
		if err := writePublishedPage(published, archivedPage); err != nil {
			return err
		}
	}
	return nil
}
