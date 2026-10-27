// Package searchpages serves the search page as HTML that the server renders: the search box, the
// operators of the offered sources, and one page of results that merges every source the search
// asks, with a notice for each source that failed. It also serves the OpenSearch description that lets a browser add the
// page as a search engine.
package searchpages

import (
	"context"
	"embed"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/pageshell"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

//go:embed static
var staticFiles embed.FS

const (
	searchPath                = "/search"
	openSearchDescriptionPath = "/opensearch.xml"
)

type ResultSource interface {
	PageFor(ctx context.Context, search searchqueries.Search) (searchresults.SourcePage, bool)
}

type ResultLinks interface {
	LinkFor(page string) string
}

type PageObserver interface {
	PageRenderFailed(ctx context.Context, err error)
}

func New(
	sources map[searchqueries.Source]ResultSource,
	resultLinks ResultLinks,
	tabBar pageshell.TabBar,
	observer PageObserver,
) http.Handler {
	page := newResultsPage(sources, resultLinks, tabBar, observer)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", page.show)
	mux.HandleFunc("GET "+searchPath, page.show)
	mux.HandleFunc("GET "+openSearchDescriptionPath, showOpenSearchDescription)
	mux.Handle("GET /static/", http.FileServerFS(staticFiles))
	mux.Handle("GET "+pageshell.StylesheetPath, pageshell.Stylesheet())
	return mux
}
