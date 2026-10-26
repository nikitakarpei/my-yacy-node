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
	pathRangeQuery  = "/api/v1/query_range"
	pathLabelValues = "/api/v1/label/{name}/values"
)

//go:embed assets
var assets embed.FS

//go:embed page.html
var pageSource string

var page = template.Must(template.New("page").Parse(pageSource))

type Dashboard struct {
	Service   string
	JSONModel []byte
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
		models[dashboard.Service] = dashboard.JSONModel
		services = append(services, dashboard.Service)
	}
	mux.Handle("GET "+pathDashboards, dashboardList(services))
	mux.Handle("GET "+pathDashboards+"/{service}", dashboardShell(models, sampleInterval))
	mux.Handle(
		"GET "+pathDashboards+"/assets/",
		http.StripPrefix(pathDashboards, http.FileServerFS(assets)),
	)
	mux.Handle(pathRangeQuery, newRangeQueryEndpoint(history))
	mux.Handle(pathLabelValues, labelValuesEndpoint{history: history})
}

func dashboardList(services []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		render(w, r, pageView{Services: services})
	})
}

func dashboardShell(models map[string][]byte, sampleInterval time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		model, known := models[service]
		if !known {
			http.NotFound(w, r)
			return
		}
		embedded := template.JS(model) //nolint:gosec // G203: the binary embeds the model.
		render(w, r, pageView{
			Service:       service,
			Model:         embedded,
			SampleSeconds: sampleInterval.Seconds(),
		})
	})
}

type pageView struct {
	Service       string
	Model         template.JS
	Services      []string
	SampleSeconds float64
}

func render(w http.ResponseWriter, r *http.Request, view pageView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Execute(w, view); err != nil {
		slog.DebugContext(r.Context(), "dashboard reader left", slog.Any("error", err))
	}
}
