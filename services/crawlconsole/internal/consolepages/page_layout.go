package consolepages

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolemessages"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

//go:embed templates
var templateFiles embed.FS

const momentLayout = "2006-01-02 15:04:05 UTC"

const (
	crawlTablesScreen  = "crawl_tables"
	crawlDetailScreen  = "crawl_detail"
	newCrawlFormScreen = "new_crawl_form"
)

type pageLayout struct {
	screens  map[string]*template.Template
	tabBar   template.HTML
	controls crawlerControls
	observer ConsoleObserver
}

type pageView struct {
	Path              string
	WebUIStylesheet   string
	TabBar            template.HTML
	Crawler           crawlerView
	CrawlerUnanswered bool
	Content           any
}

func newPageLayout(
	catalog consolemessages.Catalog,
	tabBar webuistyle.TabBar,
	controls crawlerControls,
	observer ConsoleObserver,
) pageLayout {
	functions := template.FuncMap{"text": catalog.TextOf, "moment": momentOf}
	screens := map[string]*template.Template{}
	for _, screen := range []string{crawlTablesScreen, crawlDetailScreen, newCrawlFormScreen} {
		screens[screen] = template.Must(template.New("layout.html").Funcs(functions).ParseFS(
			templateFiles, "templates/layout.html", "templates/"+screen+".html",
		))
	}
	return pageLayout{
		screens:  screens,
		tabBar:   tabBar.HTML(),
		controls: controls,
		observer: observer,
	}
}

func momentOf(moment time.Time) string {
	return moment.UTC().Format(momentLayout)
}

func (layout pageLayout) render(
	w http.ResponseWriter,
	r *http.Request,
	screen string,
	status int,
	content any,
) {
	crawler := layout.controls.viewFor(r)
	view := pageView{
		Path:              r.URL.RequestURI(),
		WebUIStylesheet:   webuistyle.StylesheetPath,
		TabBar:            layout.tabBar,
		Crawler:           crawler,
		CrawlerUnanswered: !crawler.Answered || status == http.StatusBadGateway,
		Content:           content,
	}
	var page bytes.Buffer
	if err := layout.screens[screen].Execute(&page, view); err != nil {
		layout.observer.PageRenderFailed(r.Context(), screen, err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = page.WriteTo(w)
}

func statusOfAnswer(answered bool) int {
	if answered {
		return http.StatusOK
	}
	return http.StatusBadGateway
}
