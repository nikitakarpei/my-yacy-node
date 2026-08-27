// Package scraperequestintake scrapes each page the crawl fleet scrapeRequest and stores its
// reverse word index: the page's URL metadata, then its postings.
package scraperequestintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scrapedpage"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	msgFetchFailed      = "scrape request fetch failed"
	msgFetchDeferred    = "scrape request fetch deferred by the origin"
	msgNothingToScrape  = "scrape request fetch holds no content to scrape"
	msgExtractionFailed = "scrape request document extraction failed, nothing stored"
	msgNoIndexDerived   = "scrape request derives no index, nothing stored"
	msgPageStored       = "scrape request stored"
	msgStoreDeferred    = "scrape request store deferred"
)

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) (pagefetch.FetchOutcome, error)
}

type ScrapeRequestConsumer struct {
	source                         pullintake.MessageSource
	fetcher                        PageFetcher
	formatDerivations              pageformats.FormatDerivationCatalog
	urls                           urlmeta.URLReceiver
	postings                       rwiadmission.PostingReceiver
	scrapeRequestIntakeConcurrency int
}

type Config struct {
	Source                         pullintake.MessageSource
	Fetcher                        PageFetcher
	FormatDerivations              pageformats.FormatDerivationCatalog
	URLs                           urlmeta.URLReceiver
	Postings                       rwiadmission.PostingReceiver
	ScrapeRequestIntakeConcurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:                         config.Source,
		fetcher:                        config.Fetcher,
		formatDerivations:              config.FormatDerivations,
		urls:                           config.URLs,
		postings:                       config.Postings,
		scrapeRequestIntakeConcurrency: config.ScrapeRequestIntakeConcurrency,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.scrapeRequestIntakeConcurrency, c.processOne)
}

func (c *ScrapeRequestConsumer) processOne(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	reachedAt := time.Now()
	scrapedPage, scrapable := c.fetch(ctx, message, scrapeRequest)
	if !scrapable {
		return nil
	}
	document, extracted := c.documentOf(ctx, scrapedPage)
	if !extracted {
		message.Acknowledge(ctx)
		return nil
	}
	text, derived := c.fullTextOf(ctx, document, scrapedPage.LandedURL)
	if !derived {
		slog.DebugContext(ctx, msgNoIndexDerived, slog.String("url", scrapedPage.PageURL.String()))
		message.Acknowledge(ctx)
		return nil
	}
	c.store(ctx, message, pagerwi.Of(scrapedPage, document, text, reachedAt))
	return nil
}

func (c *ScrapeRequestConsumer) fetch(
	ctx context.Context,
	message pullintake.PendingMessage,
	request scraperequestcontract.ScrapeRequest,
) (scrapedpage.ScrapedPage, bool) {
	fetchURL := request.FetchURL
	outcome, err := c.fetcher.Fetch(ctx, fetchURL, pagefetch.PageVersion{})
	if err != nil {
		slog.WarnContext(ctx, msgFetchFailed,
			slog.String("url", fetchURL.String()),
			slog.Any("error", err),
		)
		message.Return(ctx)
		return scrapedpage.ScrapedPage{}, false
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		return scrapedpage.Of(request, outcome.Page), true
	case pagefetch.FetchFailed:
		slog.WarnContext(ctx, msgFetchFailed, slog.String("url", fetchURL.String()))
		message.Return(ctx)
	case pagefetch.FetchDeferred:
		slog.DebugContext(ctx, msgFetchDeferred,
			slog.String("url", fetchURL.String()),
			slog.Duration("deferFor", outcome.DeferFor),
		)
		message.ReturnAfter(ctx, outcome.DeferFor)
	default:
		slog.DebugContext(ctx, msgNothingToScrape, slog.String("url", fetchURL.String()))
		message.Acknowledge(ctx)
	}
	return scrapedpage.ScrapedPage{}, false
}

func (c *ScrapeRequestConsumer) documentOf(
	ctx context.Context,
	scrapedPage scrapedpage.ScrapedPage,
) (documentextraction.Document, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, scrapedPage.Body, scrapedPage.ContentType, scrapedPage.LandedURL,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", scrapedPage.LandedURL.String()),
			slog.Any("error", err),
		)
		return documentextraction.Document{}, false
	}
	return document, true
}

func (c *ScrapeRequestConsumer) fullTextOf(
	ctx context.Context,
	document documentextraction.Document,
	landedURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	return c.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, document, landedURL,
	)
}

func (c *ScrapeRequestConsumer) store(
	ctx context.Context,
	message pullintake.PendingMessage,
	index pagerwi.PageRWI,
) {
	urlReceipt, err := c.urls.Receive(ctx, []yacymodel.URLMetadata{index.Metadata})
	if err != nil || urlReceipt.Busy {
		redeliver(ctx, message, index.PageURL.String(), err)
		return
	}
	postingReceipt, err := c.postings.Receive(ctx, index.Postings)
	if err != nil || postingReceipt.Busy {
		redeliver(ctx, message, index.PageURL.String(), err)
		return
	}
	message.Acknowledge(ctx)
	slog.DebugContext(ctx, msgPageStored,
		slog.String("url", index.PageURL.String()),
		slog.Int("postings", len(index.Postings)),
	)
}

func redeliver(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL string,
	cause error,
) {
	slog.WarnContext(ctx, msgStoreDeferred,
		slog.String("url", pageURL),
		slog.Any("error", cause),
	)
	message.Return(ctx)
}
