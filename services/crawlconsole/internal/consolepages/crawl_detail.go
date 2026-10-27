package consolepages

import (
	"net/http"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type crawlDetail struct {
	crawler    Crawler
	clock      Clock
	layout     pageLayout
	pathPrefix string
	observer   ConsoleObserver
}

const (
	hostTableRows        = 20
	hostsCursorParameter = "hosts"
)

type crawlDetailView struct {
	CrawlID  string
	Answered bool
	Found    bool
	Crawl    crawlRow
	Order    *yacycrawlcontract.CrawlOrder
	Hosts    hostTable
}

type hostTable struct {
	Rows            []hostRow
	NextPageURL     string
	PreviousPageURL string
}

func (detail crawlDetail) show(w http.ResponseWriter, r *http.Request) {
	crawlID := r.PathValue("id")
	crawl, found, err := detail.crawler.CrawlOf(r.Context(), crawlID)
	switch {
	case err != nil:
		detail.observer.CrawlerCallFailed(r.Context(), err)
		detail.layout.render(w, r, crawlDetailScreen, http.StatusBadGateway,
			crawlDetailView{CrawlID: crawlID})
	case !found:
		detail.layout.render(w, r, crawlDetailScreen, http.StatusNotFound,
			crawlDetailView{CrawlID: crawlID, Answered: true})
	default:
		view := crawlDetailViewOf(crawl, detail.clock.Now())
		hosts, answered := detail.hostTableOf(r, crawlID)
		view.Hosts = hosts
		detail.layout.render(w, r, crawlDetailScreen, statusOfAnswer(answered), view)
	}
}

func crawlDetailViewOf(crawl yacycrawlcontract.Crawl, now time.Time) crawlDetailView {
	return crawlDetailView{
		CrawlID:  crawl.CrawlID,
		Answered: true,
		Found:    true,
		Crawl:    crawlRowFrom(crawl, now),
		Order:    crawl.Order,
	}
}

func (detail crawlDetail) hostTableOf(r *http.Request, crawlID string) (hostTable, bool) {
	query := r.URL.Query()
	page, err := detail.crawler.HostsOf(
		r.Context(),
		crawlID,
		query.Get(hostsCursorParameter),
		hostTableRows,
	)
	if err != nil {
		detail.observer.CrawlerCallFailed(r.Context(), err)
		return hostTable{}, false
	}
	table := hostTable{Rows: make([]hostRow, 0, len(page.Hosts))}
	for _, crawlHost := range page.Hosts {
		table.Rows = append(table.Rows, hostRowFrom(crawlHost))
	}
	if page.NextCursor != "" {
		table.NextPageURL = pageURLFrom(query, hostsCursorParameter, page.NextCursor)
	}
	if page.PreviousCursor != "" {
		table.PreviousPageURL = pageURLFrom(query, hostsCursorParameter, page.PreviousCursor)
	}
	return table, true
}

func (detail crawlDetail) cancel(w http.ResponseWriter, r *http.Request) {
	crawlID := r.PathValue("id")
	if err := detail.crawler.Cancel(r.Context(), crawlID); err != nil {
		detail.observer.CrawlerCallFailed(r.Context(), err)
	}
	returnToCrawl(w, r, detail.pathPrefix, crawlID)
}

func (detail crawlDetail) changePause(w http.ResponseWriter, r *http.Request) {
	crawlID := r.PathValue("id")
	change := detail.crawler.Resume
	if pausedIn(r) {
		change = detail.crawler.Pause
	}
	if err := change(r.Context(), crawlID); err != nil {
		detail.observer.CrawlerCallFailed(r.Context(), err)
	}
	returnToCrawl(w, r, detail.pathPrefix, crawlID)
}

func returnToCrawl(w http.ResponseWriter, r *http.Request, pathPrefix string, crawlID string) {
	http.Redirect(w, r, pathPrefix+"/crawls/"+url.PathEscape(crawlID), http.StatusSeeOther)
}
