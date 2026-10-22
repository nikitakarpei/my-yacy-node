package consolepages

import (
	"maps"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	crawlTableRows            = 20
	unfinishedCursorParameter = "running"
	finishedCursorParameter   = "finished"
)

var (
	unfinishedStates = []yacycrawlcontract.CrawlState{
		yacycrawlcontract.CrawlStateRunning,
		yacycrawlcontract.CrawlStatePaused,
		yacycrawlcontract.CrawlStateCancelling,
	}
	finishedStates = []yacycrawlcontract.CrawlState{
		yacycrawlcontract.CrawlStateSucceeded,
		yacycrawlcontract.CrawlStateCancelled,
	}
)

type crawlTables struct {
	crawler  Crawler
	clock    Clock
	layout   pageLayout
	observer ConsoleObserver
}

type crawlTablesView struct {
	Unfinished crawlTable
	Finished   crawlTable
}

type crawlTable struct {
	Rows          []crawlRow
	OlderPageURL  string
	NewerPageURL  string
	NewestPageURL string
}

func (tables crawlTables) show(w http.ResponseWriter, r *http.Request) {
	unfinished, unfinishedAnswered := tables.tableOf(
		r, unfinishedStates, unfinishedCursorParameter,
	)
	finished, finishedAnswered := tables.tableOf(r, finishedStates, finishedCursorParameter)
	status := statusOfAnswer(unfinishedAnswered && finishedAnswered)
	tables.layout.render(w, r, crawlTablesScreen, status, crawlTablesView{
		Unfinished: unfinished,
		Finished:   finished,
	})
}

func (tables crawlTables) tableOf(
	r *http.Request,
	states []yacycrawlcontract.CrawlState,
	cursorParameter string,
) (crawlTable, bool) {
	query := r.URL.Query()
	page, err := tables.crawler.CrawlsIn(
		r.Context(),
		states,
		query.Get(cursorParameter),
		crawlTableRows,
	)
	if err != nil {
		tables.observer.CrawlerCallFailed(r.Context(), err)
		return crawlTable{}, false
	}
	now := tables.clock.Now()
	table := crawlTable{Rows: make([]crawlRow, 0, len(page.Crawls))}
	for _, crawl := range page.Crawls {
		table.Rows = append(table.Rows, crawlRowFrom(crawl, now))
	}
	if page.OlderCursor != "" {
		table.OlderPageURL = pageURLAt("/", query, cursorParameter, page.OlderCursor)
	}
	if page.NewerCursor != "" {
		table.NewerPageURL = pageURLAt("/", query, cursorParameter, page.NewerCursor)
		table.NewestPageURL = pageURLAt("/", query, cursorParameter, "")
	}
	return table, true
}

func pageURLAt(path string, query url.Values, cursorParameter string, cursor string) string {
	pageQuery := maps.Clone(query)
	pageQuery.Del(cursorParameter)
	if cursor != "" {
		pageQuery.Set(cursorParameter, cursor)
	}
	return path + "?" + pageQuery.Encode()
}
