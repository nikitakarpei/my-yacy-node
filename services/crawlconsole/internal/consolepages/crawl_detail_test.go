package consolepages_test

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCrawlDetailShowsTheProgressAndTheOrder(t *testing.T) {
	crawl := runningCrawlOf(t, "c1")
	crawl.CompletedPageVisitsByResult = map[yacycrawlcontract.PageVisitResult]int64{
		yacycrawlcontract.ResultRedirected: 2,
		yacycrawlcontract.ResultPublished:  6,
	}
	crawl.DroppedPageVisitsByReason = map[yacycrawlcontract.PageVisitDropReason]int64{
		yacycrawlcontract.DropReasonNotDue: 2,
	}

	answer := consoleOver(
		&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}},
	).get("/crawls/c1")

	assertStatus(t, answer, http.StatusOK)
	assertShows(
		t,
		answer,
		"docs site",
		"Running",
		`<div class="amount">5</div><div class="muted">Pending</div>`,
		`<div class="amount">6</div><div class="muted">Published</div>`,
		`<div class="amount">2</div><div class="muted">Unchanged</div>`,
		`<div class="amount">0</div><div class="muted">Not fetched</div>`,
		`<div class="amount">2h 5m</div><div class="muted">Duration</div>`,
		`<dt>Links not followed</dt><dd>11<div class="hint">`,
		"https://example.org/blog/",
		"Pages on Same Domain",
		"all URLs",
		"Skipped",
		"<dd>50</dd>",
		"Not yet",
		`<meta http-equiv="refresh"`,
		`action="/crawls/c1/pause"`,
		`name="Paused" value="true"`,
		`action="/crawls/c1/cancel"`,
		`href="/crawls/new?from=c1"`,
	)
}

func TestCrawlDetailShowsTheRecrawlGraceOfTheOrder(t *testing.T) {
	for recrawlGrace, shown := range map[time.Duration]string{
		yacycrawlcontract.NoRecrawlGrace: "<dd>any age</dd>",
		48 * time.Hour:                   "<dd>2d</dd>",
		36 * time.Hour:                   "<dd>36h</dd>",
		90 * time.Second:                 "<dd>2m</dd>",
	} {
		crawl := runningCrawlOf(t, "c1")
		crawl.Order.Profile.RecrawlGrace = recrawlGrace

		answer := consoleOver(
			&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}},
		).get("/crawls/c1")

		assertShows(t, answer, "<dt>Re-crawl Pages Older Than</dt>"+shown)
	}
}

func TestCrawlDetailOffersResumeAndCancelForAPausedCrawl(t *testing.T) {
	crawl := runningCrawlOf(t, "c1")
	crawl.State = yacycrawlcontract.CrawlStatePaused

	answer := consoleOver(
		&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}},
	).get("/crawls/c1")

	assertShows(t, answer, `name="Paused" value="false"`, "Resume", `action="/crawls/c1/cancel"`)
}

func TestCrawlDetailShowsACancellingCrawlWithItsPendingVisitsAndNoControls(t *testing.T) {
	crawl := runningCrawlOf(t, "c1")
	crawl.State = yacycrawlcontract.CrawlStateCancelling

	answer := consoleOver(
		&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}},
	).get("/crawls/c1")

	assertShows(t, answer, "Cancelling", ">Pending<", `http-equiv="refresh"`)
	assertHides(t, answer, "/crawls/c1/pause", "/crawls/c1/cancel")
}

func TestCrawlDetailOffersNoControlsOnceTheCrawlFinished(t *testing.T) {
	crawl := finishedCrawlOf(t, "c1")
	crawl.Order.Profile.MaxPagesPerHost = yacycrawlcontract.UnlimitedPagesPerHost

	answer := consoleOver(
		&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}},
	).get("/crawls/c1")

	assertShows(t, answer, "Succeeded", "2026-10-21 10:30:00 UTC", "no limit",
		`<div class="amount">1h 0m</div>`)
	assertHides(t, answer,
		"/crawls/c1/pause", "/crawls/c1/cancel", `http-equiv="refresh"`, ">Pending<")
}

func TestCrawlDetailAnswersNotFoundForAnUnknownCrawl(t *testing.T) {
	answer := consoleOver(&fakeCrawler{}).get("/crawls/c-gone")

	assertStatus(t, answer, http.StatusNotFound)
	assertShows(t, answer, "The crawler knows no crawl with this ID.")
}

func TestCrawlDetailShowsTheCrawlerDoesNotAnswer(t *testing.T) {
	answer := consoleOver(&fakeCrawler{down: true}).get("/crawls/c1")

	assertStatus(t, answer, http.StatusBadGateway)
	assertHides(t, answer, "The crawler knows no crawl with this ID.")
}

func TestCrawlDetailReportsAWordItHasNoTextFor(t *testing.T) {
	crawl := runningCrawlOf(t, "c1")
	crawl.State = "archived"
	console := consoleOver(&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}})

	answer := console.get("/crawls/c1")

	assertStatus(t, answer, http.StatusInternalServerError)
	if !slices.Equal(console.observer.unrenderedPages, []string{"crawl_detail"}) {
		t.Fatalf("unrendered pages = %v", console.observer.unrenderedPages)
	}
}

func TestCrawlControlsAskTheCrawlerAndReturnToTheCrawl(t *testing.T) {
	crawler := &fakeCrawler{}
	console := consoleOver(crawler)

	answers := []*http.Response{
		console.post("/crawls/c%2F1/pause", url.Values{"Paused": {"true"}}).Result(),
		console.post("/crawls/c%2F1/pause", url.Values{"Paused": {"false"}}).Result(),
		console.post("/crawls/c%2F1/cancel", nil).Result(),
	}

	for _, answer := range answers {
		_ = answer.Body.Close()
		if location := answer.Header.Get("Location"); answer.StatusCode != http.StatusSeeOther ||
			location != "/crawls/c%2F1" {
			t.Errorf("answer %d to %q, want 303 to /crawls/c%%2F1", answer.StatusCode, location)
		}
	}
	if want := []string{
		"pause c/1",
		"resume c/1",
		"cancel c/1",
	}; !slices.Equal(
		crawler.calls,
		want,
	) {
		t.Fatalf("crawler calls = %v, want %v", crawler.calls, want)
	}
}

func TestCrawlControlsReportAFailedCall(t *testing.T) {
	console := consoleOver(&fakeCrawler{down: true})

	console.post("/crawls/c1/cancel", nil)
	console.post("/crawls/c1/pause", url.Values{"Paused": {"true"}})

	if console.observer.failedCalls != 2 {
		t.Fatalf("failed calls = %d, want 2", console.observer.failedCalls)
	}
}
