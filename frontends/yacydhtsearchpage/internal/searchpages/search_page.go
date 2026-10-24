package searchpages

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchmessages"
)

//go:embed templates
var templateFiles embed.FS

const (
	fieldQuery       = "query"
	fieldStartRecord = "startRecord"
	resultsPerPage   = 10
	momentLayout     = "2006-01-02"
)

type searchPage struct {
	engine   SearchEngine
	screen   *template.Template
	observer SearchPageObserver
}

type searchView struct {
	Query            string
	Results          []SearchResult
	EngineUnanswered bool
	PreviousPath     string
	NextPath         string
}

func newSearchPage(
	engine SearchEngine,
	catalog searchmessages.Catalog,
	observer SearchPageObserver,
) searchPage {
	functions := template.FuncMap{"text": catalog.TextOf, "moment": momentOf}
	screen := template.Must(template.New("search.html").Funcs(functions).ParseFS(
		templateFiles, "templates/search.html",
	))
	return searchPage{engine: engine, screen: screen, observer: observer}
}

func momentOf(moment time.Time) string {
	return moment.UTC().Format(momentLayout)
}

func (page searchPage) show(w http.ResponseWriter, r *http.Request) {
	form := r.URL.Query()
	query := strings.TrimSpace(form.Get(fieldQuery))
	if query == "" {
		page.render(w, r, http.StatusOK, searchView{})
		return
	}
	offset := offsetIn(form)
	results, err := page.engine.SearchResultsFor(r.Context(), query, offset, resultsPerPage)
	if err != nil {
		page.observer.SearchEngineCallFailed(r.Context(), err)
		page.render(w, r, http.StatusBadGateway, searchView{Query: query, EngineUnanswered: true})
		return
	}
	page.render(w, r, http.StatusOK, resultsViewOf(query, offset, results))
}

func offsetIn(form url.Values) int {
	offset, err := strconv.Atoi(form.Get(fieldStartRecord))
	if err != nil || offset < 0 {
		return 0
	}
	return offset
}

func resultsViewOf(query string, offset int, results []SearchResult) searchView {
	view := searchView{Query: query, Results: results}
	if offset > 0 {
		view.PreviousPath = pathOf(query, max(offset-resultsPerPage, 0))
	}
	if len(results) == resultsPerPage {
		view.NextPath = pathOf(query, offset+resultsPerPage)
	}
	return view
}

func pathOf(query string, offset int) string {
	form := url.Values{fieldQuery: {query}}
	if offset > 0 {
		form.Set(fieldStartRecord, strconv.Itoa(offset))
	}
	return "/?" + form.Encode()
}

func (page searchPage) render(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	view searchView,
) {
	var body bytes.Buffer
	if err := page.screen.Execute(&body, view); err != nil {
		page.observer.PageRenderFailed(r.Context(), err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = body.WriteTo(w)
}
