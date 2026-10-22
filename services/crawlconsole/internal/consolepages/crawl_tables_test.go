package consolepages_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCrawlTablesListRunningAndFinishedCrawls(t *testing.T) {
	running := runningCrawlOf(t, "c-running")
	running.CompletedPageVisitsByResult = map[yacycrawlcontract.PageVisitResult]int64{
		yacycrawlcontract.ResultPublished: 8,
		yacycrawlcontract.ResultNotFound:  3,
	}
	crawler := &fakeCrawler{pages: map[string]yacycrawlcontract.CrawlPage{
		"running":   {Crawls: []yacycrawlcontract.Crawl{running}},
		"succeeded": {Crawls: []yacycrawlcontract.Crawl{finishedCrawlOf(t, "c-finished")}},
	}}

	answer := consoleOver(crawler).get("/")

	assertStatus(t, answer, http.StatusOK)
	assertShows(t, answer,
		`<td class="crawl"><a href="/crawls/c-running">docs site</a><div><span class="url muted">`+
			`https://example.org/docs/</span> <span class="muted">and 2 more</span></div></td>`,
		"Running",
		`<td class="number">5</td>
    <td class="number">8</td>
    <td class="muted moment">2026-10-21 09:30:00 UTC</td>
    <td class="number">2h 5m</td>`,
		`<td class="crawl"><a class="url" href="/crawls/c-finished">https://example.org/docs/</a>`,
		"Succeeded",
		`<td class="muted moment">2026-10-21 10:30:00 UTC</td>
    <td class="number">1h 0m</td>`,
		`href="/crawls/new?from=c-finished"`,
	)
	wantStates := [][]yacycrawlcontract.CrawlState{
		{
			yacycrawlcontract.CrawlStateRunning,
			yacycrawlcontract.CrawlStatePaused,
			yacycrawlcontract.CrawlStateCancelling,
		},
		{yacycrawlcontract.CrawlStateSucceeded, yacycrawlcontract.CrawlStateCancelled},
	}
	for i, query := range crawler.crawlsQueries {
		if !slices.Equal(query.states, wantStates[i]) {
			t.Errorf("table %d asked for %v, want %v", i, query.states, wantStates[i])
		}
	}
}

func TestCrawlTablesSayWhenNoCrawlIsListed(t *testing.T) {
	answer := consoleOver(&fakeCrawler{}).get("/")

	assertShows(t, answer, "No running crawls.", "No finished crawls.")
}

func TestCrawlTablesTurnEachTablePageOnItsOwn(t *testing.T) {
	crawler := &fakeCrawler{pages: map[string]yacycrawlcontract.CrawlPage{
		"running": {
			Crawls:      []yacycrawlcontract.Crawl{runningCrawlOf(t, "c-running")},
			OlderCursor: "r-older",
			NewerCursor: "r-newer",
		},
	}}

	answer := consoleOver(crawler).get("/?finished=f-at&running=r-at")

	assertShows(t, answer,
		`href="/?finished=f-at&amp;running=r-older"`,
		`href="/?finished=f-at&amp;running=r-newer"`,
		`href="/?finished=f-at"`,
	)
	wantCursors := []string{"r-at", "f-at"}
	for i, query := range crawler.crawlsQueries {
		if query.cursor != wantCursors[i] {
			t.Errorf("table %d read from cursor %q, want %q", i, query.cursor, wantCursors[i])
		}
	}
}

func TestCrawlTablesShowTheCrawlerDoesNotAnswer(t *testing.T) {
	console := consoleOver(&fakeCrawler{down: true})

	answer := console.get("/")

	assertStatus(t, answer, http.StatusBadGateway)
	assertShows(t, answer, "The crawler does not answer.")
	if console.observer.failedCalls == 0 {
		t.Fatal("no failed crawler call was reported")
	}
}
