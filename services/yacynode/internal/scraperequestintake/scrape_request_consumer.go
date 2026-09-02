// Package scraperequestintake scrapes each page the crawl fleet requests and stores its
// reverse word index: the page's URL metadata, then its postings.
package scraperequestintake

import (
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

var errURLMetadataAdmissionRejected = errors.New("url metadata admission rejected")

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) (pagefetch.FetchOutcome, error)
}

type ScrapeRequestConsumer struct {
	scrapeRequestSource            pullintake.MessageSource
	pageFetcher                    PageFetcher
	formatDerivations              pageformats.FormatDerivationCatalog
	urlReceiver                    urlmeta.URLReceiver
	postingReceiver                rwiadmission.PostingReceiver
	scrapeProgress                 ScrapeProgress
	scrapeRequestIntakeConcurrency int
}

type ScrapeRequestConsumerConfig struct {
	ScrapeRequestSource            pullintake.MessageSource
	PageFetcher                    PageFetcher
	FormatDerivations              pageformats.FormatDerivationCatalog
	URLReceiver                    urlmeta.URLReceiver
	PostingReceiver                rwiadmission.PostingReceiver
	ScrapeProgress                 ScrapeProgress
	ScrapeRequestIntakeConcurrency int
}

func NewScrapeRequestConsumer(config ScrapeRequestConsumerConfig) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		scrapeRequestSource:            config.ScrapeRequestSource,
		pageFetcher:                    config.PageFetcher,
		formatDerivations:              config.FormatDerivations,
		urlReceiver:                    config.URLReceiver,
		postingReceiver:                config.PostingReceiver,
		scrapeProgress:                 config.ScrapeProgress,
		scrapeRequestIntakeConcurrency: config.ScrapeRequestIntakeConcurrency,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(
		ctx,
		c.scrapeRequestSource,
		c.scrapeRequestIntakeConcurrency,
		c.processOne,
	)
}

func (c *ScrapeRequestConsumer) processOne(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	scrapeRequest, err := pagescrapecontract.UnmarshalScrapeRequest(message.Body())
	if err != nil {
		c.scrapeProgress.ScrapeRequestInvalid(ctx)

		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	reachedAt := time.Now()
	scrapedPage, scrapable := c.fetch(ctx, message, scrapeRequest)
	if !scrapable {
		return nil
	}
	document, extracted := c.documentOf(ctx, message, scrapedPage)
	if !extracted {
		message.Acknowledge(ctx)
		return nil
	}
	text, derived := c.fullTextOf(ctx, document, scrapedPage.LandedURL)
	if !derived {
		c.scrapeProgress.NoIndexDerived(ctx, message.Identity(), scrapedPage.PageURL)
		message.Acknowledge(ctx)
		return nil
	}
	c.store(ctx, message, pagerwi.Of(scrapedPage, document, text, reachedAt))
	return nil
}

func (c *ScrapeRequestConsumer) fetch(
	ctx context.Context,
	message pullintake.PendingMessage,
	request pagescrapecontract.ScrapeRequest,
) (pagescrapecontract.OfferedPage, bool) {
	fetchURL := request.FetchURL
	outcome, err := c.pageFetcher.Fetch(ctx, fetchURL, pagefetch.PageVersion{})
	if err != nil {
		c.scrapeProgress.OriginFetchFailed(ctx, message.Identity(), fetchURL, err)
		message.Return(ctx)
		return pagescrapecontract.OfferedPage{}, false
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		return pagescrapecontract.OfferedPageFrom(request, outcome.Page), true
	case pagefetch.FetchFailed:
		c.scrapeProgress.OriginFetchFailed(ctx, message.Identity(), fetchURL, nil)
		message.Return(ctx)
	case pagefetch.FetchDeferred:
		c.scrapeProgress.OriginFetchDeferred(
			ctx,
			message.Identity(),
			fetchURL,
			outcome.DeferFor,
		)
		message.ReturnAfter(ctx, outcome.DeferFor)
	default:
		c.scrapeProgress.NothingToScrape(ctx, message.Identity(), fetchURL)
		message.Acknowledge(ctx)
	}
	return pagescrapecontract.OfferedPage{}, false
}

func (c *ScrapeRequestConsumer) documentOf(
	ctx context.Context,
	message pullintake.PendingMessage,
	scrapedPage pagescrapecontract.OfferedPage,
) (documentextraction.Document, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, scrapedPage.Body, scrapedPage.ContentType, scrapedPage.LandedURL,
	)
	if err != nil {
		c.scrapeProgress.DocumentExtractionFailed(
			ctx,
			message.Identity(),
			scrapedPage.LandedURL,
			err,
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
	urlReceipt, err := c.urlReceiver.Receive(ctx, []yacymodel.URLMetadata{index.Metadata})
	if err != nil {
		c.scrapeProgress.URLMetadataAdmissionFailed(ctx, message.Identity(), index.PageURL, err)
		message.Return(ctx)
		return
	}
	if urlReceipt.Busy {
		c.scrapeProgress.URLMetadataAdmissionBusy(ctx, message.Identity(), index.PageURL)
		message.Return(ctx)
		return
	}
	if len(urlReceipt.ErrorURL) != 0 {
		c.scrapeProgress.URLMetadataAdmissionFailed(
			ctx,
			message.Identity(),
			index.PageURL,
			errURLMetadataAdmissionRejected,
		)
		message.Return(ctx)
		return
	}
	c.scrapeProgress.URLMetadataAdmitted(ctx, message.Identity(), index.PageURL)
	postingReceipt, err := c.postingReceiver.Receive(ctx, index.Postings)
	if err != nil {
		c.scrapeProgress.PostingsAdmissionFailed(
			ctx,
			message.Identity(),
			index.PageURL,
			len(index.Postings),
			err,
		)
		message.Return(ctx)
		return
	}
	if postingReceipt.Busy {
		c.scrapeProgress.PostingsAdmissionBusy(
			ctx,
			message.Identity(),
			index.PageURL,
			len(index.Postings),
		)
		message.Return(ctx)
		return
	}
	c.scrapeProgress.PostingsAdmitted(
		ctx,
		message.Identity(),
		index.PageURL,
		len(index.Postings),
	)
	message.Acknowledge(ctx)
	c.scrapeProgress.ScrapeRequestCompleted(ctx, message.Identity(), index.PageURL)
}
