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
)

const (
	pageURL        = "https://example.org/a"
	landedPageURL  = "https://example.org/b"
	deferralWindow = time.Hour
)

type scrapeIntake struct {
	message      *pullintaketest.Message
	offers       *pageOffers
	redirections *pageRedirections
	schedules    *scrapeSchedules
	feed         *scrapeOutcomeFeed
}

type scrapeBroker struct {
	offers       *pageOffers
	redirections *pageRedirections
	schedules    *scrapeSchedules
}

func acceptingScrapeBroker() scrapeBroker {
	return scrapeBroker{
		offers:       &pageOffers{},
		redirections: &pageRedirections{},
		schedules:    &scrapeSchedules{},
	}
}

func runScrapeIntake(
	t *testing.T,
	request pagescrapecontract.ScrapeRequest,
	reads pageReads,
	readingTime time.Time,
) scrapeIntake {
	t.Helper()
	return runScrapeIntakeAgainst(t, request, reads, readingTime, acceptingScrapeBroker())
}

func runScrapeIntakeAgainst(
	t *testing.T,
	request pagescrapecontract.ScrapeRequest,
	reads pageReads,
	readingTime time.Time,
	broker scrapeBroker,
) scrapeIntake {
	t.Helper()
	body, err := pagescrapecontract.MarshalScrapeRequest(request)
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	message := &pullintaketest.Message{Body: body}
	feed := &scrapeOutcomeFeed{}
	consumer := scrapeintake.NewScrapeRequestConsumer(scrapeintake.Config{
		ScrapeRequests:    pullintaketest.MessageSourceOf(message),
		PageFetcher:       reads,
		PageOffers:        broker.offers,
		PageRedirections:  broker.redirections,
		ScrapeSchedules:   broker.schedules,
		ScrapeOutcomeFeed: feed,
		ScrapeProgress:    scrapeintake.ScrapeProgressObservers{silentScrapeProgress{}},
		DeferralWindow:    deferralWindow,
		IntakeConcurrency: 1,
		ReadingTime:       func() time.Time { return readingTime },
	})
	if err := consumer.Run(context.Background()); err != nil {
		t.Fatalf("run intake: %v", err)
	}
	return scrapeIntake{
		message:      message,
		offers:       broker.offers,
		redirections: broker.redirections,
		schedules:    broker.schedules,
		feed:         feed,
	}
}

func scrapeRequestForThePage(t *testing.T) pagescrapecontract.ScrapeRequest {
	t.Helper()
	pageCanonicalURL := canonicalurltest.CanonicalURLOf(t, pageURL)
	return pagescrapecontract.ScrapeRequest{PageURL: pageCanonicalURL, FetchURL: pageCanonicalURL}
}

func TestPageReadFromTheOriginIsOfferedToTheCorpora(t *testing.T) {
	request := scrapeRequestForThePage(t)
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			LandedURL:   canonicalurltest.CanonicalURLOf(t, pageURL),
			ContentType: "text/html",
			Body:        []byte("<html lang=\"en\"><body>page</body></html>"),
		},
	}}

	intake := runScrapeIntake(t, request, read, time.Now())

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

func TestPageThatRedirectsIsRememberedUnderTheRequestedURL(t *testing.T) {
	request := scrapeRequestForThePage(t)
	landedURL := canonicalurltest.CanonicalURLOf(t, landedPageURL)
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page:   pagefetch.FetchedPage{LandedURL: landedURL, ContentType: "text/html"},
	}}

	intake := runScrapeIntake(t, request, read, time.Now())

	if got := intake.redirections.recorded[request.PageURL]; got != landedURL {
		t.Errorf("%s redirects to %s, want %s", request.PageURL, got, landedURL)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestPageThatLandsWhereItWasAskedForRemembersNoRedirection(t *testing.T) {
	request := scrapeRequestForThePage(t)
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			LandedURL:   canonicalurltest.CanonicalURLOf(t, pageURL),
			ContentType: "text/html",
		},
	}}

	intake := runScrapeIntake(t, request, read, time.Now())

	if len(intake.redirections.recorded) != 0 {
		t.Errorf("remembered %d redirections, want none", len(intake.redirections.recorded))
	}
}

func TestRequestComesBackWhenTheRedirectionIsNotRemembered(t *testing.T) {
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			LandedURL:   canonicalurltest.CanonicalURLOf(t, landedPageURL),
			ContentType: "text/html",
		},
	}}
	broker := acceptingScrapeBroker()
	broker.redirections = &pageRedirections{err: errBrokerRefused}

	intake := runScrapeIntakeAgainst(t, scrapeRequestForThePage(t), read, time.Now(), broker)

	if len(intake.offers.offered) != 0 {
		t.Errorf("offered %d pages, want none", len(intake.offers.offered))
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
}

func TestPageTheOriginDoesNotServeIsReportedAsAScrapeFailureAndSettled(t *testing.T) {
	request := scrapeRequestForThePage(t)
	read := pageReads{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}}

	intake := runScrapeIntake(t, request, read, time.Now())

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

func TestReadThatFailsWithoutAnAnswerNamesNoReason(t *testing.T) {
	read := pageReads{err: errBrokerRefused}

	intake := runScrapeIntake(t, scrapeRequestForThePage(t), read, time.Now())

	if len(intake.offers.failures) != 1 {
		t.Fatalf("reported %d failures, want exactly one", len(intake.offers.failures))
	}
	if got := intake.offers.failures[0].Reason; got != pagescrapecontract.NoReasonGiven {
		t.Errorf("failure reason = %s, want %s", got, pagescrapecontract.NoReasonGiven)
	}
}

func TestDeferredReadIsScheduledForALaterReadAndSettled(t *testing.T) {
	request := scrapeRequestForThePage(t)
	readingTime := time.Now()
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}

	intake := runScrapeIntake(t, request, read, readingTime)

	if len(intake.schedules.scheduled) != 1 {
		t.Fatalf("scheduled %d scrapes, want exactly one", len(intake.schedules.scheduled))
	}
	scheduled := intake.schedules.scheduled[0]
	if scheduled.after != time.Minute {
		t.Errorf("scheduled in %s, want %s", scheduled.after, time.Minute)
	}
	if !scheduled.request.DeferredSince.Equal(readingTime) {
		t.Errorf("deferred since %s, want %s", scheduled.request.DeferredSince, readingTime)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestDeferredReadOfARequestThatGivesUpOnDeferralFails(t *testing.T) {
	request := scrapeRequestForThePage(t)
	request.GivesUpOnDeferral = true
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}

	intake := runScrapeIntake(t, request, read, time.Now())

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

func TestReadDeferredPastTheDeferralWindowFails(t *testing.T) {
	readingTime := time.Now()
	request := scrapeRequestForThePage(t)
	request.DeferredSince = readingTime.Add(-deferralWindow - time.Minute)
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}

	intake := runScrapeIntake(t, request, read, readingTime)

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
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page:   pagefetch.FetchedPage{LandedURL: canonicalurltest.CanonicalURLOf(t, pageURL)},
	}}
	broker := acceptingScrapeBroker()
	broker.offers = &pageOffers{err: errBrokerRefused}

	intake := runScrapeIntakeAgainst(t, scrapeRequestForThePage(t), read, time.Now(), broker)

	if settlement := intake.message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
}

func TestRequestComesBackWhenTheBrokerSchedulesNoLaterRead(t *testing.T) {
	read := pageReads{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: time.Minute,
	}}
	broker := acceptingScrapeBroker()
	broker.schedules = &scrapeSchedules{err: errBrokerRefused}

	intake := runScrapeIntakeAgainst(t, scrapeRequestForThePage(t), read, time.Now(), broker)

	if settlement := intake.message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("the request was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
}

func TestUnreadableScrapeRequestHaltsIntake(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("not json")}
	consumer := scrapeintake.NewScrapeRequestConsumer(scrapeintake.Config{
		ScrapeRequests:    pullintaketest.MessageSourceOf(message),
		PageFetcher:       pageReads{},
		PageOffers:        &pageOffers{},
		PageRedirections:  &pageRedirections{},
		ScrapeSchedules:   &scrapeSchedules{},
		ScrapeOutcomeFeed: &scrapeOutcomeFeed{},
		ScrapeProgress:    scrapeintake.ScrapeProgressObservers{silentScrapeProgress{}},
		DeferralWindow:    deferralWindow,
		IntakeConcurrency: 1,
	})

	if err := consumer.Run(context.Background()); err == nil {
		t.Fatal("want intake to halt on a request it cannot read")
	}
}
