package scrapeintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		knownVersion pagefetch.PageVersion,
	) (redirectfollowingfetch.LandedFetch, error)
}

type PageOffers interface {
	OfferPage(ctx context.Context, page pagescrapecontract.OfferedPage) error
	ReportScrapeFailure(ctx context.Context, failure pagescrapecontract.ScrapeFailure) error
}

type ScrapeSchedules interface {
	ScheduleScrape(
		ctx context.Context,
		request pagescrapecontract.ScrapeRequest,
		after time.Duration,
	) error
}

type ScrapeOutcomeFeed interface {
	AnnounceScrapeFailure(ctx context.Context, failure pagescrapecontract.ScrapeFailure)
}

type ScrapeRequestConsumer struct {
	scrapeRequests       pullintake.MessageSource
	pageFetcher          PageFetcher
	pageOffers           PageOffers
	scrapeSchedules      ScrapeSchedules
	scrapeOutcomeFeed    ScrapeOutcomeFeed
	scrapeIntakeObserver ScrapeIntakeObserver
	deferralWindow       time.Duration
	intakeConcurrency    int
	clock                Clock
}

//nolint:revive // a consumer names every collaborator it scrapes a page with
func NewScrapeRequestConsumer(
	scrapeRequests pullintake.MessageSource,
	pageFetcher PageFetcher,
	pageOffers PageOffers,
	scrapeSchedules ScrapeSchedules,
	scrapeOutcomeFeed ScrapeOutcomeFeed,
	scrapeIntakeObserver ScrapeIntakeObserver,
	deferralWindow time.Duration,
	intakeConcurrency int,
	clock Clock,
) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		scrapeRequests:       scrapeRequests,
		pageFetcher:          pageFetcher,
		pageOffers:           pageOffers,
		scrapeSchedules:      scrapeSchedules,
		scrapeOutcomeFeed:    scrapeOutcomeFeed,
		scrapeIntakeObserver: scrapeIntakeObserver,
		deferralWindow:       deferralWindow,
		intakeConcurrency:    intakeConcurrency,
		clock:                clock,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.scrapeRequests, c.intakeConcurrency, c.processOne)
}

func (c *ScrapeRequestConsumer) processOne(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	request, err := pagescrapecontract.UnmarshalScrapeRequest(message.Body())
	if err != nil {
		c.scrapeIntakeObserver.ScrapeRequestInvalid(ctx, message.Identity(), err)
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	c.scrapeIntakeObserver.ScrapeRequestReceived(ctx, request.PageURL)
	landed, err := c.pageFetcher.Fetch(ctx, request.FetchURL, pagefetch.PageVersion{})
	if err != nil {
		c.scrapeIntakeObserver.OriginFetchFailed(ctx, request.FetchURL, err)
		c.reportFailure(ctx, message, request, pagescrapecontract.NoReasonGiven)
		return nil
	}
	switch landed.Outcome.Status {
	case pagefetch.FetchSucceeded:
		c.offerPage(ctx, message, pagescrapecontract.OfferedPageFrom(
			request, landed.Outcome.Page, landed.URL,
		))
	case pagefetch.FetchDeferred:
		c.deferScrape(ctx, message, request, landed.Outcome.DeferFor)
	default:
		c.reportFailure(ctx, message, request, scrapeFailureReasonOf(landed.Outcome.Status))
	}
	return nil
}

func (c *ScrapeRequestConsumer) offerPage(
	ctx context.Context,
	message pullintake.PendingMessage,
	page pagescrapecontract.OfferedPage,
) {
	if err := c.pageOffers.OfferPage(ctx, page); err != nil {
		c.scrapeIntakeObserver.PageNotOffered(ctx, page.PageURL, err)
		message.Return(ctx)
		return
	}
	c.scrapeIntakeObserver.PageOffered(ctx, page.PageURL, page.LandedURL)
	message.Acknowledge(ctx)
}

func (c *ScrapeRequestConsumer) deferScrape(
	ctx context.Context,
	message pullintake.PendingMessage,
	request pagescrapecontract.ScrapeRequest,
	deferFor time.Duration,
) {
	if request.GivesUpOnDeferral {
		c.reportFailure(ctx, message, request, pagescrapecontract.Deferred)
		return
	}
	deferred := request
	if deferred.DeferredSince.IsZero() {
		deferred.DeferredSince = c.clock.Now()
	}
	if c.clock.Now().Sub(deferred.DeferredSince) > c.deferralWindow {
		c.reportFailure(ctx, message, request, pagescrapecontract.DeferredTooLong)
		return
	}
	if err := c.scrapeSchedules.ScheduleScrape(ctx, deferred, deferFor); err != nil {
		c.scrapeIntakeObserver.ScrapeScheduleFailed(ctx, request.PageURL, err)
		message.Return(ctx)
		return
	}
	c.scrapeIntakeObserver.ScrapeDeferred(ctx, request.PageURL, deferFor)
	message.Acknowledge(ctx)
}

func (c *ScrapeRequestConsumer) reportFailure(
	ctx context.Context,
	message pullintake.PendingMessage,
	request pagescrapecontract.ScrapeRequest,
	reason pagescrapecontract.ScrapeFailureReason,
) {
	failure := pagescrapecontract.ScrapeFailure{
		PageURL:  request.PageURL,
		FetchURL: request.FetchURL,
		Reason:   reason,
	}
	if err := c.pageOffers.ReportScrapeFailure(ctx, failure); err != nil {
		c.scrapeIntakeObserver.PageNotOffered(ctx, request.PageURL, err)
		message.Return(ctx)
		return
	}
	c.scrapeOutcomeFeed.AnnounceScrapeFailure(ctx, failure)
	c.scrapeIntakeObserver.ScrapeFailed(ctx, request.PageURL, reason)
	message.Acknowledge(ctx)
}

func scrapeFailureReasonOf(status pagefetch.FetchStatus) pagescrapecontract.ScrapeFailureReason {
	switch status {
	case pagefetch.FetchNotModified:
		return pagescrapecontract.NotModified
	case pagefetch.FetchAccessRefused, pagefetch.FetchRejected:
		return pagescrapecontract.AccessRefused
	case pagefetch.FetchRedirected:
		return pagescrapecontract.RedirectsExhausted
	case pagefetch.FetchRedirectTargetInvalid:
		return pagescrapecontract.RedirectTargetInvalid
	case pagefetch.FetchOversized:
		return pagescrapecontract.Oversized
	default:
		return pagescrapecontract.NoReasonGiven
	}
}
