package scrapeintake_test

import (
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

var errBrokerRefused = errors.New("the broker refused the message")

type pageReads struct {
	outcome pagefetch.FetchOutcome
	err     error
}

func (r pageReads) Fetch(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return r.outcome, r.err
}

type pageOffers struct {
	offered  []pagescrapecontract.OfferedPage
	failures []pagescrapecontract.ScrapeFailure
	err      error
}

func (o *pageOffers) OfferPage(_ context.Context, page pagescrapecontract.OfferedPage) error {
	if o.err != nil {
		return o.err
	}
	o.offered = append(o.offered, page)
	return nil
}

func (o *pageOffers) ReportScrapeFailure(
	_ context.Context,
	failure pagescrapecontract.ScrapeFailure,
) error {
	if o.err != nil {
		return o.err
	}
	o.failures = append(o.failures, failure)
	return nil
}

type scrapeSchedule struct {
	request pagescrapecontract.ScrapeRequest
	after   time.Duration
}

type scrapeSchedules struct {
	scheduled []scrapeSchedule
	err       error
}

func (s *scrapeSchedules) ScheduleScrape(
	_ context.Context,
	request pagescrapecontract.ScrapeRequest,
	after time.Duration,
) error {
	if s.err != nil {
		return s.err
	}
	s.scheduled = append(s.scheduled, scrapeSchedule{request: request, after: after})
	return nil
}

type scrapeOutcomeFeed struct {
	announced []pagescrapecontract.ScrapeFailure
}

func (f *scrapeOutcomeFeed) AnnounceScrapeFailure(
	_ context.Context,
	failure pagescrapecontract.ScrapeFailure,
) {
	f.announced = append(f.announced, failure)
}

type silentScrapeIntakeObserver struct{}

func (silentScrapeIntakeObserver) ScrapeRequestInvalid(_ context.Context, _ string, _ error) {}

func (silentScrapeIntakeObserver) ScrapeRequestReceived(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) {
}

func (silentScrapeIntakeObserver) OriginReadFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (silentScrapeIntakeObserver) PageOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
) {
}

func (silentScrapeIntakeObserver) PageNotOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (silentScrapeIntakeObserver) ScrapeDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
}

func (silentScrapeIntakeObserver) ScrapeScheduleFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (silentScrapeIntakeObserver) ScrapeFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagescrapecontract.ScrapeFailureReason,
) {
}
