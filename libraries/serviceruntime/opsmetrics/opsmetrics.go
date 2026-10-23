// Package opsmetrics serves the operator-facing endpoints of a service, among them its
// Prometheus registry, while the service runs.
package opsmetrics

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	pathMetrics = "/metrics"

	serverReadHeaderLimit = 10 * time.Second
	serverShutdownLimit   = 15 * time.Second
)

func ProcessRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return registry
}

func MuxFor(registry *prometheus.Registry) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(pathMetrics, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return mux
}

func ServeAlongside(
	ctx context.Context,
	listener net.Listener,
	opsHandler http.Handler,
	service func(context.Context) error,
) error {
	opsServer := &http.Server{Handler: opsHandler, ReadHeaderTimeout: serverReadHeaderLimit}
	return servergroup.Run(
		ctx,
		serverShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Listener: listener, Server: opsServer}},
		service,
	)
}
