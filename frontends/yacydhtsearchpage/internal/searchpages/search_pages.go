// Package searchpages serves the search page as HTML that the server renders: a search box
// and, for a query, the results that the search engine finds, ten to a page, with links to
// the pages before and after.
package searchpages

import (
	"context"
	"embed"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchmessages"
)

//go:embed static
var staticFiles embed.FS

type SearchResult struct {
	Title       string
	Address     string
	Description string
	PublishedAt time.Time
}

type SearchEngine interface {
	SearchResultsFor(
		ctx context.Context,
		query string,
		offset int,
		limit int,
	) ([]SearchResult, error)
}

type SearchPageObserver interface {
	SearchEngineCallFailed(ctx context.Context, err error)
	PageRenderFailed(ctx context.Context, err error)
}

func New(
	engine SearchEngine,
	catalog searchmessages.Catalog,
	observer SearchPageObserver,
) http.Handler {
	page := newSearchPage(engine, catalog, observer)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", page.show)
	mux.Handle("GET /static/", http.FileServerFS(staticFiles))
	return mux
}
