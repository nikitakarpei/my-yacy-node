// Package scraperequestintake scrapes each page the crawl fleet scrapeRequest and stores its
// reverse word index: the page's URL metadata, then its postings.
package scraperequestintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
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
	source      pullintake.MessageSource
	fetcher     PageFetcher
	formats     pageformats.DerivableFormats
	urls        urlmeta.URLReceiver
	postings    rwiadmission.PostingReceiver
	concurrency int
}

type Config struct {
	Source      pullintake.MessageSource
	Fetcher     PageFetcher
	Formats     pageformats.DerivableFormats
	URLs        urlmeta.URLReceiver
	Postings    rwiadmission.PostingReceiver
	Concurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:      config.Source,
		fetcher:     config.Fetcher,
		formats:     config.Formats,
		urls:        config.URLs,
		postings:    config.Postings,
		concurrency: config.Concurrency,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *ScrapeRequestConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	reachedAt := time.Now()
	fetched, scrapable := c.fetch(ctx, msg, scrapeRequest.CanonicalURL)
	if !scrapable {
		return nil
	}
	document, text, derived := c.fullTextOf(ctx, fetched)
	if !derived {
		slog.DebugContext(ctx, msgNoIndexDerived, slog.String("url", fetched.FinalURL.String()))
		_ = msg.Ack()
		return nil
	}
	c.store(ctx, msg, pagerwi.Of(fetched, document, text, reachedAt))
	return nil
}

func (c *ScrapeRequestConsumer) fetch(
	ctx context.Context,
	msg jetstream.Msg,
	pageURL canonicalurl.CanonicalURL,
) (pagefetch.FetchedPage, bool) {
	outcome, err := c.fetcher.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil {
		slog.WarnContext(ctx, msgFetchFailed,
			slog.String("url", pageURL.String()),
			slog.Any("error", err),
		)
		_ = msg.Nak()
		return pagefetch.FetchedPage{}, false
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		return outcome.Page, true
	case pagefetch.FetchFailed:
		slog.WarnContext(ctx, msgFetchFailed, slog.String("url", pageURL.String()))
		_ = msg.Nak()
	case pagefetch.FetchDeferred:
		slog.DebugContext(ctx, msgFetchDeferred,
			slog.String("url", pageURL.String()),
			slog.Duration("deferFor", outcome.DeferFor),
		)
		_ = msg.NakWithDelay(outcome.DeferFor)
	default:
		slog.DebugContext(ctx, msgNothingToScrape, slog.String("url", pageURL.String()))
		_ = msg.Ack()
	}
	return pagefetch.FetchedPage{}, false
}

func (c *ScrapeRequestConsumer) fullTextOf(
	ctx context.Context,
	fetched pagefetch.FetchedPage,
) (documentextraction.Document, []byte, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, fetched.Body, fetched.ContentType, fetched.FinalURL,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", fetched.FinalURL.String()),
			slog.Any("error", err),
		)
		return documentextraction.Document{}, nil, false
	}
	text, derived, err := c.formats.BodyIn(
		documentextraction.FormatFullText, document, fetched.FinalURL,
	)
	if err != nil {
		slog.WarnContext(ctx, msgNoIndexDerived,
			slog.String("url", fetched.FinalURL.String()),
			slog.Any("error", err),
		)
		return documentextraction.Document{}, nil, false
	}
	return document, text, derived
}

func (c *ScrapeRequestConsumer) store(
	ctx context.Context,
	msg jetstream.Msg,
	index pagerwi.PageRWI,
) {
	urlReceipt, err := c.urls.Receive(ctx, []yacymodel.URLMetadata{index.Metadata})
	if err != nil || urlReceipt.Busy {
		redeliver(ctx, msg, index.CanonicalURL.String(), err)
		return
	}
	postingReceipt, err := c.postings.Receive(ctx, index.Postings)
	if err != nil || postingReceipt.Busy {
		redeliver(ctx, msg, index.CanonicalURL.String(), err)
		return
	}
	_ = msg.Ack()
	slog.DebugContext(ctx, msgPageStored,
		slog.String("url", index.CanonicalURL.String()),
		slog.Int("postings", len(index.Postings)),
	)
}

func redeliver(ctx context.Context, msg jetstream.Msg, canonicalURL string, cause error) {
	slog.WarnContext(ctx, msgStoreDeferred,
		slog.String("url", canonicalURL),
		slog.Any("error", cause),
	)
	_ = msg.Nak()
}
