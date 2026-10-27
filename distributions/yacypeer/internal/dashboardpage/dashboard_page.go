// Package dashboardpage serves the Grafana dashboards of the services as pages that draw
// their panels from the metric history, and the Prometheus range query and label values
// endpoints that the pages ask.
package dashboardpage

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/prometheus/storage"

	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

const (
	Path            = "/dashboards"
	pathRangeQuery  = Path + "/api/v1/query_range"
	pathLabelValues = Path + "/api/v1/label/{name}/values"
)

//go:embed assets
var assets embed.FS

//go:embed dashboard_list.html
var dashboardListSource string

//go:embed service_dashboard.html
var serviceDashboardSource string

var (
	dashboardListPage    = template.Must(template.New("list").Parse(dashboardListSource))
	serviceDashboardPage = template.Must(template.New("dashboard").Parse(serviceDashboardSource))
)

type Dashboard struct {
	Service string
	Model   []byte
}

func Register(
	mux *http.ServeMux,
	history storage.Queryable,
	sampleInterval time.Duration,
	dashboards []Dashboard,
	tabBar webuistyle.TabBar,
) {
	models := make(map[string][]byte, len(dashboards))
	services := make([]string, 0, len(dashboards))
	for _, dashboard := range dashboards {
		models[dashboard.Service] = dashboard.Model
		services = append(services, dashboard.Service)
	}
	frame := pageFrame{WebUIStylesheet: webuistyle.StylesheetPath, TabBar: tabBar.HTML()}
	mux.Handle("GET "+Path, dashboardList(frame, services))
	mux.Handle(
		"GET "+Path+"/{service}",
		serviceDashboard(frame, models, sampleInterval),
	)
	mux.Handle(
		"GET "+Path+"/assets/",
		http.StripPrefix(Path, http.FileServerFS(assets)),
	)
	mux.Handle("GET "+webuistyle.StylesheetPath, webuistyle.Stylesheet())
	mux.Handle(pathRangeQuery, newRangeQueryEndpoint(history))
	mux.Handle(pathLabelValues, labelValuesEndpoint{history: history})
}

type pageFrame struct {
	WebUIStylesheet string
	TabBar          template.HTML
}

func dashboardList(frame pageFrame, services []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		render(w, r, dashboardListPage, dashboardListView{Frame: frame, Services: services})
	})
}

type dashboardListView struct {
	Frame    pageFrame
	Services []string
}

func serviceDashboard(
	frame pageFrame,
	models map[string][]byte,
	sampleInterval time.Duration,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		model, known := models[service]
		if !known {
			http.NotFound(w, r)
			return
		}
		embedded := template.JS(model) //nolint:gosec // G203: the binary embeds the model.
		render(w, r, serviceDashboardPage, serviceDashboardView{
			Frame:                 frame,
			Service:               service,
			Model:                 embedded,
			SampleIntervalSeconds: sampleInterval.Seconds(),
		})
	})
}

type serviceDashboardView struct {
	Frame                 pageFrame
	Service               string
	Model                 template.JS
	SampleIntervalSeconds float64
}

func render(w http.ResponseWriter, r *http.Request, page *template.Template, view any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Execute(w, view); err != nil {
		slog.DebugContext(r.Context(), "dashboard reader left", slog.Any("error", err))
	}
}
