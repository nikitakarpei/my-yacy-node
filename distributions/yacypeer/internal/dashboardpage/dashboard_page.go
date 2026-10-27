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
)

const (
	pathDashboards  = "/dashboards"
	pathRangeQuery  = pathDashboards + "/api/v1/query_range"
	pathLabelValues = pathDashboards + "/api/v1/label/{name}/values"
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
) {
	models := make(map[string][]byte, len(dashboards))
	services := make([]string, 0, len(dashboards))
	for _, dashboard := range dashboards {
		models[dashboard.Service] = dashboard.Model
		services = append(services, dashboard.Service)
	}
	mux.Handle("GET "+pathDashboards, dashboardList(services))
	mux.Handle("GET "+pathDashboards+"/{service}", serviceDashboard(models, sampleInterval))
	mux.Handle(
		"GET "+pathDashboards+"/assets/",
		http.StripPrefix(pathDashboards, http.FileServerFS(assets)),
	)
	mux.Handle(pathRangeQuery, newRangeQueryEndpoint(history))
	mux.Handle(pathLabelValues, labelValuesEndpoint{history: history})
}

func dashboardList(services []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		render(w, r, dashboardListPage, dashboardListView{Services: services})
	})
}

type dashboardListView struct {
	Services []string
}

func serviceDashboard(models map[string][]byte, sampleInterval time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		model, known := models[service]
		if !known {
			http.NotFound(w, r)
			return
		}
		embedded := template.JS(model) //nolint:gosec // G203: the binary embeds the model.
		render(w, r, serviceDashboardPage, serviceDashboardView{
			Service:               service,
			Model:                 embedded,
			SampleIntervalSeconds: sampleInterval.Seconds(),
		})
	})
}

type serviceDashboardView struct {
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
