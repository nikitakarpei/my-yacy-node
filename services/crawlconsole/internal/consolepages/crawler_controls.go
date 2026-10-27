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

type crawlerControls struct {
	crawler    Crawler
	pathPrefix string
	observer   ConsoleObserver
}

type crawlerView struct {
	Answered     bool
	CrawlerState string
	Status       yacycrawlcontract.CrawlerStatus
}

func (controls crawlerControls) viewFor(r *http.Request) crawlerView {
	status, err := controls.crawler.CrawlerStatus(r.Context())
	if err != nil {
		controls.observer.CrawlerCallFailed(r.Context(), err)
		return crawlerView{CrawlerState: crawlerUnknown}
	}
	return crawlerView{Answered: true, CrawlerState: crawlerStateOf(status), Status: status}
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

func (controls crawlerControls) changePause(w http.ResponseWriter, r *http.Request) {
	change := controls.crawler.ResumeCrawler
	if pausedIn(r) {
		change = controls.crawler.PauseCrawler
	}
	if err := change(r.Context()); err != nil {
		controls.observer.CrawlerCallFailed(r.Context(), err)
	}
	//nolint:gosec // G710: backPathOf keeps local paths only.
	http.Redirect(w, r, controls.pathPrefix+backPathOf(r), http.StatusSeeOther)
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
