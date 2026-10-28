// Package consolepages serves the crawl console as HTML pages that the server renders: the
// tables of running and finished crawls with the controls that pause every page visit, one crawl
// with its progress, hosts and controls, and the form that starts a crawl. A form sent from
// another origin is refused. Every page can sit under a path prefix.
package consolepages

import (
	"context"
	"embed"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolemessages"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
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

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func New(
	crawler Crawler,
	catalog consolemessages.Catalog,
	tabBar webuistyle.TabBar,
	pathPrefix string,
	clock Clock,
	observer ConsoleObserver,
) http.Handler {
	controls := crawlerControls{crawler: crawler, observer: observer}
	layout := newPageLayout(catalog, tabBar, pathPrefix, controls, observer)
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
	mux.HandleFunc("POST /crawler-pause", controls.changePause)
	mux.Handle("GET /static/", http.FileServerFS(staticFiles))
	mux.Handle("GET "+webuistyle.StylesheetPath, webuistyle.Stylesheet())
	return http.NewCrossOriginProtection().Handler(mountedAt(pathPrefix, mux))
}
