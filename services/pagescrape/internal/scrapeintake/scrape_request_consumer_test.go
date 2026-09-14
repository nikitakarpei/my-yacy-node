package scrapeintake_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	pageURL        = "https://example.org/a"
	landedPageURL  = "https://example.org/b"
	deferralWindow = time.Hour
)

type scrapeIntake struct {
	message   *pullintaketest.Message
	offers    *pageOffers
	schedules *scrapeSchedules
	feed      *scrapeOutcomeFeed
}

type scrapeBroker struct {
	offers    *pageOffers
	schedules *scrapeSchedules
}

func acceptingScrapeBroker() scrapeBroker {
	return scrapeBroker{
		offers:    &pageOffers{},
		schedules: &scrapeSchedules{},
	}
}

func runScrapeIntake(
	t *testing.T,
	request pagescrapecontract.ScrapeRequest,
	fetches pageFetches,
	fetchTime time.Time,
) scrapeIntake {
	t.Helper()
	return runScrapeIntakeAgainst(t, request, fetches, fetchTime, acceptingScrapeBroker())
}

func runScrapeIntakeAgainst(
	t *testing.T,
	request pagescrapecontract.ScrapeRequest,
	fetches pageFetches,
	fetchTime time.Time,
	broker scrapeBroker,
) scrapeIntake {
	t.Helper()
	body, err := pagescrapecontract.MarshalScrapeRequest(request)
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	message := &pullintaketest.Message{Body: body}
	feed := &scrapeOutcomeFeed{}
	consumer := scrapeintake.NewScrapeRequestConsumer(
		pullintaketest.MessageSourceOf(message),
		fetches,
		broker.offers,
		broker.schedules,
		feed,
		scrapeintake.ScrapeIntakeObservers{silentScrapeIntakeObserver{}},
		deferralWindow,
		1,
		fixedClock{now: fetchTime},
	)
	if err := consumer.Run(context.Background()); err != nil {
		t.Fatalf("run intake: %v", err)
	}
	return scrapeIntake{
		message:   message,
		offers:    broker.offers,
		schedules: broker.schedules,
		feed:      feed,
	}
}

func scrapeRequestForThePage(t *testing.T) pagescrapecontract.ScrapeRequest {
	t.Helper()
	pageCanonicalURL := canonicalurltest.CanonicalURLOf(t, pageURL)
	return pagescrapecontract.ScrapeRequest{PageURL: pageCanonicalURL, FetchURL: pageCanonicalURL}
}

func TestPageFetchedFromTheOriginIsOfferedToTheCorpora(t *testing.T) {
	request := scrapeRequestForThePage(t)
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType: "text/html",
			Body:        []byte("<html lang=\"en\"><body>page</body></html>"),
		},
	}}

	intake := runScrapeIntake(t, request, fetch, time.Now())

	if len(intake.offers.offered) != 1 {
		t.Fatalf("offered %d pages, want exactly one", len(intake.offers.offered))
	}
	if got := intake.offers.offered[0].PageURL; got != request.PageURL {
		t.Errorf("offered the page %s, want %s", got, request.PageURL)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestPageThatLandedElsewhereIsOfferedUnderTheURLTheRequestNamed(t *testing.T) {
	request := scrapeRequestForThePage(t)
	landedURL := canonicalurltest.CanonicalURLOf(t, landedPageURL)
	fetch := pageFetches{
		landedURL: landedURL,
		outcome: pagefetch.FetchOutcome{
			Status: pagefetch.FetchSucceeded,
			Page:   pagefetch.FetchedPage{ContentType: "text/html"},
		},
	}

	intake := runScrapeIntake(t, request, fetch, time.Now())

	if len(intake.offers.offered) != 1 {
		t.Fatalf("offered %d pages, want exactly one", len(intake.offers.offered))
	}
	offered := intake.offers.offered[0]
	if offered.PageURL != request.PageURL {
		t.Errorf("offered the page %s, want %s", offered.PageURL, request.PageURL)
	}
	if offered.LandedURL != landedURL {
		t.Errorf("the page landed on %s, want %s", offered.LandedURL, landedURL)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestPageTheOriginDoesNotServeIsReportedAsAScrapeFailureAndSettled(t *testing.T) {
	request := scrapeRequestForThePage(t)
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}}

	intake := runScrapeIntake(t, request, fetch, time.Now())

	if len(intake.offers.failures) != 1 {
		t.Fatalf("reported %d failures, want exactly one", len(intake.offers.failures))
	}
	if got := intake.offers.failures[0].Reason; got != pagescrapecontract.AccessRefused {
		t.Errorf("failure reason = %s, want %s", got, pagescrapecontract.AccessRefused)
	}
	if len(intake.feed.announced) != 1 {
		t.Errorf("announced %d failures on the page feed, want exactly one",
			len(intake.feed.announced))
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestPageStillRedirectingWhenTheFetchStoppedFailsAsRedirectsExhausted(t *testing.T) {
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRedirected}}

	intake := runScrapeIntake(t, scrapeRequestForThePage(t), fetch, time.Now())

	if len(intake.offers.failures) != 1 {
		t.Fatalf("reported %d failures, want exactly one", len(intake.offers.failures))
	}
	if got := intake.offers.failures[0].Reason; got != pagescrapecontract.RedirectsExhausted {
		t.Errorf("failure reason = %s, want %s", got, pagescrapecontract.RedirectsExhausted)
	}
}

func TestPageBehindAnInvalidRedirectTargetFails(t *testing.T) {
	fetch := pageFetches{
		outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRedirectTargetInvalid},
	}

	intake := runScrapeIntake(t, scrapeRequestForThePage(t), fetch, time.Now())

	if len(intake.offers.failures) != 1 {
		t.Fatalf("reported %d failures, want exactly one", len(intake.offers.failures))
	}
	if got := intake.offers.failures[0].Reason; got != pagescrapecontract.RedirectTargetInvalid {
		t.Errorf("failure reason = %s, want %s", got, pagescrapecontract.RedirectTargetInvalid)
	}
}

func TestFetchThatFailsWithoutAnAnswerNamesNoReason(t *testing.T) {
	fetch := pageFetches{err: errBrokerRefused}

	intake := runScrapeIntake(t, scrapeRequestForThePage(t), fetch, time.Now())

	if len(intake.offers.failures) != 1 {
		t.Fatalf("reported %d failures, want exactly one", len(intake.offers.failures))
	}
	if got := intake.offers.failures[0].Reason; got != pagescrapecontract.NoReasonGiven {
		t.Errorf("failure reason = %s, want %s", got, pagescrapecontract.NoReasonGiven)
	}
}

func TestDeferredFetchIsScheduledForALaterFetchAndSettled(t *testing.T) {
	request := scrapeRequestForThePage(t)
	fetchTime := time.Now()
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}

	intake := runScrapeIntake(t, request, fetch, fetchTime)

	if len(intake.schedules.scheduled) != 1 {
		t.Fatalf("scheduled %d scrapes, want exactly one", len(intake.schedules.scheduled))
	}
	scheduled := intake.schedules.scheduled[0]
	if scheduled.after != time.Minute {
		t.Errorf("scheduled in %s, want %s", scheduled.after, time.Minute)
	}
	if !scheduled.request.DeferredSince.Equal(fetchTime) {
		t.Errorf("deferred since %s, want %s", scheduled.request.DeferredSince, fetchTime)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestDeferredFetchOfARequestThatGivesUpOnDeferralFails(t *testing.T) {
	request := scrapeRequestForThePage(t)
	request.GivesUpOnDeferral = true
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}

	intake := runScrapeIntake(t, request, fetch, time.Now())

	if len(intake.schedules.scheduled) != 0 {
		t.Errorf("scheduled %d scrapes, want none", len(intake.schedules.scheduled))
	}
	if len(intake.offers.failures) != 1 {
		t.Fatalf("reported %d failures, want exactly one", len(intake.offers.failures))
	}
	if got := intake.offers.failures[0].Reason; got != pagescrapecontract.Deferred {
		t.Errorf("failure reason = %s, want %s", got, pagescrapecontract.Deferred)
	}
}

func TestFetchDeferredPastTheDeferralWindowFails(t *testing.T) {
	fetchTime := time.Now()
	request := scrapeRequestForThePage(t)
	request.DeferredSince = fetchTime.Add(-deferralWindow - time.Minute)
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}

	intake := runScrapeIntake(t, request, fetch, fetchTime)

	if len(intake.schedules.scheduled) != 0 {
		t.Errorf("scheduled %d scrapes, want none", len(intake.schedules.scheduled))
	}
	if len(intake.offers.failures) != 1 {
		t.Fatalf("reported %d failures, want exactly one", len(intake.offers.failures))
	}
	if got := intake.offers.failures[0].Reason; got != pagescrapecontract.DeferredTooLong {
		t.Errorf("failure reason = %s, want %s", got, pagescrapecontract.DeferredTooLong)
	}
}

func TestRequestComesBackWhenTheBrokerTakesNoOffer(t *testing.T) {
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
	}}
	broker := acceptingScrapeBroker()
	broker.offers = &pageOffers{err: errBrokerRefused}

	intake := runScrapeIntakeAgainst(t, scrapeRequestForThePage(t), fetch, time.Now(), broker)

	if settlement := intake.message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
}

func TestRequestComesBackWhenTheBrokerSchedulesNoLaterFetch(t *testing.T) {
	fetch := pageFetches{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}
	broker := acceptingScrapeBroker()
	broker.schedules = &scrapeSchedules{err: errBrokerRefused}

	intake := runScrapeIntakeAgainst(t, scrapeRequestForThePage(t), fetch, time.Now(), broker)

	if settlement := intake.message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
}

func TestScrapeRequestNoOneCanTakeHaltsIntake(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("not json")}
	consumer := scrapeintake.NewScrapeRequestConsumer(
		pullintaketest.MessageSourceOf(message),
		pageFetches{},
		&pageOffers{},
		&scrapeSchedules{},
		&scrapeOutcomeFeed{},
		scrapeintake.ScrapeIntakeObservers{silentScrapeIntakeObserver{}},
		deferralWindow,
		1,
		wallclock.Clock{},
	)

	if err := consumer.Run(context.Background()); err == nil {
		t.Fatal("want intake to halt on a request it cannot take")
	}
}
