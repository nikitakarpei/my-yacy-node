package consolepages_test

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestTheCrawlTablesShowWhatTheCrawlerDoes(t *testing.T) {
	cases := []struct {
		name      string
		status    yacycrawlcontract.CrawlerStatus
		fragments []string
	}{
		{
			name: "crawling",
			status: yacycrawlcontract.CrawlerStatus{
				PendingPageVisits:     30,
				AmountOfRunningCrawls: 2,
			},
			fragments: []string{"Crawling", "2 running · 30 pages pending", "Pause all"},
		},
		{
			name:      "idle",
			status:    yacycrawlcontract.CrawlerStatus{},
			fragments: []string{"Idle", "Pause all"},
		},
		{
			name:      "paused",
			status:    yacycrawlcontract.CrawlerStatus{Paused: true, PendingPageVisits: 30},
			fragments: []string{"Paused", "Resume all"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answer := consoleOver(&fakeCrawler{status: tc.status}).get("/")

			assertStatus(t, answer, http.StatusOK)
			assertShows(t, answer, tc.fragments...)
		})
	}
}

func TestTheCrawlTablesShowTheCrawlerDoesNotAnswer(t *testing.T) {
	answer := consoleOver(&fakeCrawler{down: true}).get("/")

	assertShows(t, answer, "Unknown", "The crawler does not answer.")
	assertHides(t, answer, "Pause all", "Resume all")
}

func TestTheOtherPagesShowOnlyThatTheCrawlerIsPaused(t *testing.T) {
	paused := consoleOver(&fakeCrawler{status: yacycrawlcontract.CrawlerStatus{Paused: true}})
	crawling := consoleOver(&fakeCrawler{
		status: yacycrawlcontract.CrawlerStatus{PendingPageVisits: 30},
	})

	pausedForm := paused.get("/crawls/new")
	crawlingForm := crawling.get("/crawls/new")

	assertShows(t, pausedForm, "The crawler is paused.")
	assertHides(t, pausedForm, "Resume all")
	assertHides(t, crawlingForm, "The crawler is paused.", "Pause all", "pages pending")
}

func TestCrawlerPauseReturnsToThePageItWasSentFrom(t *testing.T) {
	crawler := &fakeCrawler{}
	console := consoleOver(crawler)

	paused := console.post(
		"/crawler-pause",
		url.Values{"Paused": {"true"}, "Back": {"/crawls/c1?x=1"}},
	)
	resumed := console.post("/crawler-pause", url.Values{"Paused": {"false"}, "Back": {"/"}})

	assertRedirect(t, paused, "/crawls/c1?x=1")
	assertRedirect(t, resumed, "/")
	if want := []string{"pause crawler", "resume crawler"}; !slices.Equal(crawler.calls, want) {
		t.Fatalf("crawler calls = %v, want %v", crawler.calls, want)
	}
}

func TestCrawlerPauseReturnsHomeInsteadOfToAnotherHost(t *testing.T) {
	for _, back := range []string{"//elsewhere.example/", "https://elsewhere.example/", `/\elsewhere.example`, "crawls", "%zz"} {
		t.Run(back, func(t *testing.T) {
			answer := consoleOver(&fakeCrawler{}).post("/crawler-pause",
				url.Values{"Paused": {"true"}, "Back": {back}})

			assertRedirect(t, answer, "/")
		})
	}
}

func TestCrawlerPauseReportsAFailedCall(t *testing.T) {
	console := consoleOver(&fakeCrawler{down: true})

	answer := console.post("/crawler-pause", url.Values{"Paused": {"true"}, "Back": {"/"}})

	assertRedirect(t, answer, "/")
	if console.observer.failedCalls != 1 {
		t.Fatalf("failed calls = %d, want 1", console.observer.failedCalls)
	}
}
