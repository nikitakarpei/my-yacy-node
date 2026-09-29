package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/safetensorsmodel"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	assessmentgateobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgateobservers/applog"
	assessmentgateobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgateobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/egresstransports/absoluteurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/egresstransports/tunnel"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/pageassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxyintake"
	proxyintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxyintakeobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
	requestrelayobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelayobservers/applog"
	requestrelayobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelayobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "spamproxy started"
	msgServiceStopped = "spamproxy stopped"
)

func RunService(ctx context.Context, cfg ServiceConfig, registry *prometheus.Registry) error {
	modelFile, err := os.ReadFile(cfg.SafetensorsModelPath)
	if err != nil {
		return fmt.Errorf("read spam model: %w", err)
	}
	model, err := safetensorsmodel.ModelFrom(modelFile)
	if err != nil {
		return err
	}
	clock := wallclock.Clock{}
	gatedAssessor := assessmentgate.New(
		pageassessment.NewPageAssessor(model),
		runtime.GOMAXPROCS(0),
		clock,
		assessmentgate.Observers{
			assessmentgateobserversapplog.GateLog{},
			assessmentgateobserversprometheus.New(registry),
		},
	)
	relayer := requestrelay.New(
		egressTransportFor(cfg),
		gatedAssessor,
		requestrelay.Observers{
			requestrelayobserversapplog.RelayLog{},
			requestrelayobserversprometheus.New(registry),
		},
		requestrelay.Limits{
			PageByteCeiling:       cfg.PageByteCeiling,
			MaxPagesReadAtOnce:    cfg.MaxPagesReadAtOnce,
			ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
			RelayIdleTimeout:      cfg.RelayIdleTimeout,
		},
		clock,
	)
	proxyServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           proxyintake.New(relayer, proxyintakeobserversapplog.IntakeLog{}),
		ReadHeaderTimeout: readHeaderLimit,
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", cfg.ListenAddr),
		slog.String("model", model.Version),
	)
	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "proxy", Server: proxyServer},
		{Name: "ops", Server: opsServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func egressTransportFor(cfg ServiceConfig) requestrelay.Egress {
	if cfg.EgressProxyDialMode == DialModeAbsoluteURL {
		return absoluteurl.New(cfg.EgressProxyURL)
	}
	return tunnel.New(cfg.EgressProxyURL)
}
