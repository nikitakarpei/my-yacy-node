// Package markdownintake derives the markdown of each page a scrape request names, stores
// it under the URL the origin settled on, remembers the redirection that led there, and
// reports what became of every request it takes on.
package markdownintake

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scrapedpage"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

var errOriginReportedFetchFailure = errors.New("the origin reported the fetch failed")

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) (pagefetch.FetchOutcome, error)
}

type PageMarkdownCorpus interface {
	Put(ctx context.Context, canonicalURL canonicalurl.CanonicalURL, markdown []byte) error
}

type PageRedirections interface {
	Record(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
		markdownURL canonicalurl.CanonicalURL,
	) error
}

type ScrapeRequestConsumer struct {
	source                         pullintake.MessageSource
	fetcher                        PageFetcher
	formatDerivations              pageformats.FormatDerivationCatalog
	corpus                         PageMarkdownCorpus
	redirections                   PageRedirections
	progress                       ScrapeProgress
	scrapeRequestIntakeConcurrency int
}

type Config struct {
	Source                         pullintake.MessageSource
	Fetcher                        PageFetcher
	FormatDerivations              pageformats.FormatDerivationCatalog
	Corpus                         PageMarkdownCorpus
	Redirections                   PageRedirections
	Progress                       ScrapeProgress
	ScrapeRequestIntakeConcurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:                         config.Source,
		fetcher:                        config.Fetcher,
		formatDerivations:              config.FormatDerivations,
		corpus:                         config.Corpus,
		redirections:                   config.Redirections,
		progress:                       config.Progress,
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
	c.progress.ScrapeRequestReceived(ctx)
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	requestedURL := scrapeRequest.PageURL
	scrapedPage, scrapable := c.fetch(ctx, message, scrapeRequest)
	if !scrapable {
		return nil
	}
	document, extracted := c.documentOf(ctx, scrapedPage, requestedURL)
	if !extracted {
		message.Acknowledge(ctx)
		return nil
	}
	markdown, derived := c.markdownOf(ctx, document, scrapedPage.LandedURL)
	if !derived {
		c.progress.NoMarkdownDerived(ctx, requestedURL, scrapedPage.LandedURL)
		message.Acknowledge(ctx)
		return nil
	}
	return c.store(ctx, message, requestedURL, scrapedPage.PageURL, markdown)
}

func (c *ScrapeRequestConsumer) fetch(
	ctx context.Context,
	message pullintake.PendingMessage,
	request scraperequestcontract.ScrapeRequest,
) (scrapedpage.ScrapedPage, bool) {
	fetchURL := request.FetchURL
	outcome, err := c.fetcher.Fetch(ctx, fetchURL, pagefetch.PageVersion{})
	if err != nil {
		c.progress.OriginFetchFailed(ctx, fetchURL, err)
		message.Return(ctx)
		return scrapedpage.ScrapedPage{}, false
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		return scrapedpage.Of(request, outcome.Page), true
	case pagefetch.FetchFailed:
		c.progress.OriginFetchFailed(ctx, fetchURL, errOriginReportedFetchFailure)
		message.Return(ctx)
	case pagefetch.FetchDeferred:
		c.progress.OriginFetchDeferred(ctx, fetchURL, outcome.DeferFor)
		message.ReturnAfter(ctx, outcome.DeferFor)
	default:
		c.progress.NothingToScrape(ctx, fetchURL)
		message.Acknowledge(ctx)
	}
	return scrapedpage.ScrapedPage{}, false
}

func (c *ScrapeRequestConsumer) documentOf(
	ctx context.Context,
	scrapedPage scrapedpage.ScrapedPage,
	requestedURL canonicalurl.CanonicalURL,
) (documentextraction.Document, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, scrapedPage.Body, scrapedPage.ContentType, scrapedPage.LandedURL,
	)
	if err != nil {
		c.progress.DocumentExtractionFailed(ctx, requestedURL, scrapedPage.LandedURL, err)
		return documentextraction.Document{}, false
	}
	return document, true
}

func (c *ScrapeRequestConsumer) markdownOf(
	ctx context.Context,
	document documentextraction.Document,
	landedURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	return c.formatDerivations.BodyIn(
		ctx, documentextraction.FormatMarkdown, document, landedURL,
	)
}

func (c *ScrapeRequestConsumer) store(
	ctx context.Context,
	message pullintake.PendingMessage,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if err := c.corpus.Put(ctx, markdownURL, markdown); err != nil {
		c.progress.MarkdownCorpusWriteFailed(ctx, markdownURL, err)
		message.Return(ctx)
		return nil
	}
	if requestedURL != markdownURL {
		if err := c.redirections.Record(ctx, requestedURL, markdownURL); err != nil {
			c.progress.RedirectionRecordWriteFailed(ctx, requestedURL, markdownURL, err)
			message.Return(ctx)
			return nil
		}
	}
	c.progress.MarkdownStored(ctx, requestedURL, markdownURL)
	message.Acknowledge(ctx)
	return nil
}
