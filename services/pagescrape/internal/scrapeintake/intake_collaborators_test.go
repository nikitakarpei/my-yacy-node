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

type pageRedirections struct {
	recorded map[canonicalurl.CanonicalURL]canonicalurl.CanonicalURL
	err      error
}

func (r *pageRedirections) Record(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
	pageURL canonicalurl.CanonicalURL,
) error {
	if r.err != nil {
		return r.err
	}
	if r.recorded == nil {
		r.recorded = map[canonicalurl.CanonicalURL]canonicalurl.CanonicalURL{}
	}
	r.recorded[requestedURL] = pageURL
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
) error {
	f.announced = append(f.announced, failure)
	return nil
}

type silentScrapeProgress struct{}

func (silentScrapeProgress) ScrapeRequestInvalid(_ context.Context, _ string, _ error) {}

func (silentScrapeProgress) ScrapeRequestReceived(_ context.Context, _ canonicalurl.CanonicalURL) {}

func (silentScrapeProgress) OriginReadFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (silentScrapeProgress) PageOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
) {
}

func (silentScrapeProgress) PageNotOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (silentScrapeProgress) RedirectionNotRecorded(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (silentScrapeProgress) ScrapeDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
}

func (silentScrapeProgress) ScrapeScheduleFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (silentScrapeProgress) ScrapeFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagescrapecontract.ScrapeFailureReason,
) {
}

func (silentScrapeProgress) ScrapeOutcomeAnnouncementFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}
