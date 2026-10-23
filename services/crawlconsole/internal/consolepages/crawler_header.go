package consolepages

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	crawlerCrawling = "crawling"
	crawlerIdle     = "idle"
	crawlerPaused   = "paused"
	crawlerUnknown  = "unknown"
)

type crawlerHeader struct {
	crawler  Crawler
	observer ConsoleObserver
}

type headerView struct {
	Answered     bool
	CrawlerState string
	Status       yacycrawlcontract.CrawlerStatus
}

func (header crawlerHeader) headerFor(r *http.Request) headerView {
	status, err := header.crawler.CrawlerStatus(r.Context())
	if err != nil {
		header.observer.CrawlerCallFailed(r.Context(), err)
		return headerView{CrawlerState: crawlerUnknown}
	}
	return headerView{Answered: true, CrawlerState: crawlerStateOf(status), Status: status}
}

func crawlerStateOf(status yacycrawlcontract.CrawlerStatus) string {
	switch {
	case status.Paused:
		return crawlerPaused
	case status.PendingPageVisits > 0:
		return crawlerCrawling
	default:
		return crawlerIdle
	}
}

func (header crawlerHeader) changePause(w http.ResponseWriter, r *http.Request) {
	change := header.crawler.ResumeCrawler
	if pausedIn(r) {
		change = header.crawler.PauseCrawler
	}
	if err := change(r.Context()); err != nil {
		header.observer.CrawlerCallFailed(r.Context(), err)
	}
	//nolint:gosec // G710: backPathOf keeps local paths only.
	http.Redirect(w, r, backPathOf(r), http.StatusSeeOther)
}

func pausedIn(r *http.Request) bool {
	return r.PostFormValue("Paused") == "true"
}

func backPathOf(r *http.Request) string {
	back, err := url.Parse(r.PostFormValue("Back"))
	if err != nil || back.Scheme != "" || back.Host != "" || !isLocalPath(back.Path) {
		return "/"
	}
	return back.String()
}

func isLocalPath(path string) bool {
	return strings.HasPrefix(path, "/") &&
		!strings.HasPrefix(path, "//") &&
		!strings.HasPrefix(path, `/\`)
}
