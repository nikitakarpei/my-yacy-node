package consolepages_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func everyWordOnce() yacycrawlcontract.CrawlHost {
	crawlHost := yacycrawlcontract.CrawlHost{
		Host:                        "example.org",
		PendingPageVisits:           4,
		CompletedPageVisitsByResult: map[yacycrawlcontract.PageVisitResult]int64{},
		DroppedPageVisitsByReason:   map[yacycrawlcontract.PageVisitDropReason]int64{},
	}
	for _, result := range yacycrawlcontract.PageVisitResults {
		crawlHost.CompletedPageVisitsByResult[result] = 1
	}
	for _, reason := range yacycrawlcontract.PageVisitDropReasons {
		crawlHost.DroppedPageVisitsByReason[reason] = 1
	}
	return crawlHost
}

func TestHostTableSortsEveryWordIntoItsColumn(t *testing.T) {
	crawler := &fakeCrawler{
		crawls: map[string]yacycrawlcontract.Crawl{"c1": runningCrawlOf(t, "c1")},
		hostPage: yacycrawlcontract.CrawlHostPage{
			Hosts: []yacycrawlcontract.CrawlHost{everyWordOnce()},
		},
	}

	answer := consoleOver(crawler).get("/crawls/c1")

	assertStatus(t, answer, http.StatusOK)
	assertShows(t, answer, `<td class="host">example.org</td>
    <td class="number">4</td>
    <td class="number">1</td>
    <td class="number">2</td>
    <td class="number">2</td>
    <td class="number">1</td>
    <td class="number">3</td>
    <td class="number">2</td>
    <td class="number">1</td>`)
}

func TestHostTableTurnsItsPagesOnTheCrawlPage(t *testing.T) {
	crawler := &fakeCrawler{
		crawls: map[string]yacycrawlcontract.Crawl{"c1": runningCrawlOf(t, "c1")},
		hostPage: yacycrawlcontract.CrawlHostPage{
			NextCursor:     "h-next",
			PreviousCursor: "h-previous",
		},
	}

	answer := consoleOver(crawler).get("/crawls/c1?hosts=h-at")

	assertShows(t, answer, `href="/crawls/c1?hosts=h-next"`, `href="/crawls/c1?hosts=h-previous"`,
		"No hosts yet")
	if !slices.Equal(crawler.hostCursors, []string{"h-at"}) {
		t.Fatalf("host cursors = %v, want h-at", crawler.hostCursors)
	}
}

func TestHostTableShowsTheCrawlerDoesNotAnswer(t *testing.T) {
	crawler := &fakeCrawler{
		crawls:    map[string]yacycrawlcontract.Crawl{"c1": runningCrawlOf(t, "c1")},
		hostsDown: true,
	}

	answer := consoleOver(crawler).get("/crawls/c1")

	assertStatus(t, answer, http.StatusBadGateway)
	assertShows(t, answer, "docs site", "The crawler does not answer.")
}

func TestHostTableLeavesOutPendingVisitsOnceTheCrawlFinished(t *testing.T) {
	crawler := &fakeCrawler{
		crawls: map[string]yacycrawlcontract.Crawl{"c1": finishedCrawlOf(t, "c1")},
		hostPage: yacycrawlcontract.CrawlHostPage{
			Hosts: []yacycrawlcontract.CrawlHost{everyWordOnce()},
		},
	}

	answer := consoleOver(crawler).get("/crawls/c1")

	assertShows(t, answer, `<td class="host">example.org</td>`)
	assertHides(t, answer, ">Pending<", `<td class="number">4</td>`)
}

func TestHostTableTotalsEveryHostOfTheCrawl(t *testing.T) {
	crawl := runningCrawlOf(t, "c1")
	crawl.CompletedPageVisitsByResult = everyWordOnce().CompletedPageVisitsByResult
	crawl.DroppedPageVisitsByReason = everyWordOnce().DroppedPageVisitsByReason

	answer := consoleOver(
		&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}},
	).get("/crawls/c1")

	assertShows(t, answer, `<td>All hosts</td>
    <td class="number">5</td>
    <td class="number">1</td>
    <td class="number">2</td>
    <td class="number">2</td>
    <td class="number">1</td>
    <td class="number">3</td>
    <td class="number">2</td>
    <td class="number">1</td>`, `<div class="amount">9</div><div class="muted">Not fetched</div>`)
}
