package yacydhtsearchassembly

import (
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
)

func OpsMuxFor(cfg ServiceConfig, registry *prometheus.Registry) *http.ServeMux {
	mux := opsmetrics.MuxFor(registry)
	if cfg.ServeProfiler {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	return mux
}
