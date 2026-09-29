package main

import (
	"context"
	"crypto/tls"
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
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/egresstransports/absoluteurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/egresstransports/tunnel"
	gateobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/gateobservers/applog"
	intakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/intakeobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/pagerelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxyintake"
	relayobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayobservers/applog"
	relayobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "spamproxy started"
	msgServiceStopped = "spamproxy stopped"
)

func RunService(ctx context.Context, cfg ServiceConfig, registry *prometheus.Registry) error {
	modelFile, err := os.ReadFile(cfg.ModelPath)
	if err != nil {
		return fmt.Errorf("read spam model: %w", err)
	}
	model, err := safetensorsmodel.ModelFrom(modelFile)
	if err != nil {
		return err
	}
	clock := wallclock.Clock{}
	gate := assessmentgate.New(
		spamassessment.NewPageAssessor(model),
		runtime.GOMAXPROCS(0),
		clock,
		gateobserversapplog.GateLog{},
	)
	relay := pagerelay.New(
		egressTransportFor(cfg),
		gate,
		pagerelay.Observers{
			relayobserversapplog.RelayLog{},
			relayobserversprometheus.New(registry),
		},
		pagerelay.Limits{
			PageByteCeiling:        cfg.PageByteCeiling,
			MaxPagesAssessedAtOnce: cfg.MaxPagesAssessedAtOnce,
			ReplyTimeouts:          cfg.ReplyTimeouts,
			RelayIdleTimeout:       cfg.RelayIdleTimeout,
		},
		clock,
	)
	proxyServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           proxyintake.New(relay, intakeobserversapplog.IntakeLog{}),
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

func egressTransportFor(cfg ServiceConfig) pagerelay.Egress {
	if cfg.EgressProxyDialMode == DialModeAbsoluteURL {
		return absoluteurl.New(cfg.EgressProxyURL)
	}
	return tunnel.New(cfg.EgressProxyURL, &tls.Config{MinVersion: tls.VersionTLS12})
}
