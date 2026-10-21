// Package consolepages serves the crawl console as HTML pages that the server renders: the
// tables of running and finished crawls, one crawl with its progress, hosts and controls, the form
// that starts a crawl, and the header that pauses every page visit. A form sent from another
// origin is refused.
package consolepages

import (
	"context"
	"embed"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolemessages"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

//go:embed static
var staticFiles embed.FS

type Crawler interface {
	Place(
		ctx context.Context,
		order yacycrawlcontract.CrawlOrder,
	) (crawlID string, refusal string, err error)
	CrawlsIn(
		ctx context.Context,
		states []yacycrawlcontract.CrawlState,
		cursor string,
		limit int,
	) (yacycrawlcontract.CrawlPage, error)
	CrawlOf(ctx context.Context, crawlID string) (yacycrawlcontract.Crawl, bool, error)
	HostsOf(
		ctx context.Context,
		crawlID string,
		cursor string,
		limit int,
	) (yacycrawlcontract.CrawlHostPage, error)
	Cancel(ctx context.Context, crawlID string) error
	Pause(ctx context.Context, crawlID string) error
	Resume(ctx context.Context, crawlID string) error
	CrawlerStatus(ctx context.Context) (yacycrawlcontract.CrawlerStatus, error)
	PauseCrawler(ctx context.Context) error
	ResumeCrawler(ctx context.Context) error
}

type Clock interface {
	Now() time.Time
}

type ConsoleObserver interface {
	CrawlerCallFailed(ctx context.Context, err error)
	PageRenderFailed(ctx context.Context, screen string, err error)
}

func New(
	crawler Crawler,
	catalog consolemessages.Catalog,
	clock Clock,
	observer ConsoleObserver,
) http.Handler {
	header := crawlerHeader{crawler: crawler, observer: observer}
	layout := newPageLayout(catalog, header, observer)
	tables := crawlTables{crawler: crawler, clock: clock, layout: layout, observer: observer}
	detail := crawlDetail{crawler: crawler, clock: clock, layout: layout, observer: observer}
	form := newCrawlForm{crawler: crawler, layout: layout, observer: observer}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", tables.show)
	mux.HandleFunc("GET /crawls/new", form.show)
	mux.HandleFunc("POST /crawls", form.placeOrder)
	mux.HandleFunc("GET /crawls/{id}", detail.show)
	mux.HandleFunc("POST /crawls/{id}/cancel", detail.cancel)
	mux.HandleFunc("POST /crawls/{id}/pause", detail.changePause)
	mux.HandleFunc("POST /crawler-pause", header.changePause)
	mux.Handle("GET /static/", http.FileServerFS(staticFiles))
	return http.NewCrossOriginProtection().Handler(mux)
}
