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
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/deadlineenforcing"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/failurereporting"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/idlecancelling"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/verdictadding"
	deadlineenforcingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/deadlineenforcing/applog"
	deadlineenforcingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/deadlineenforcing/prometheus"
	failurereportingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/failurereporting/applog"
	failurereportingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/failurereporting/prometheus"
	verdictaddingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/verdictadding/applog"
	verdictaddingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/verdictadding/prometheus"
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
	assessmentRunner := assessmentgate.New(
		pageassessment.New(model),
		runtime.GOMAXPROCS(0),
		clock,
		assessmentgate.Observers{
			assessmentgateobserversapplog.GateLog{},
			assessmentgateobserversprometheus.New(registry),
		},
	)
	failureReporter := failurereporting.New(
		egressTransportFor(cfg),
		failurereporting.Observers{
			failurereportingobserversapplog.FailureLog{},
			failurereportingobserversprometheus.New(registry),
		},
	)
	idleCanceller := idlecancelling.New(failureReporter, cfg.RelayIdleTimeout, clock)
	verdictAdder := verdictadding.New(
		idleCanceller,
		assessmentRunner,
		verdictadding.Limits{
			PageByteCeiling:    cfg.PageByteCeiling,
			MaxPagesReadAtOnce: cfg.MaxPagesReadAtOnce,
		},
		clock,
		verdictadding.Observers{
			verdictaddingobserversapplog.VerdictLog{},
			verdictaddingobserversprometheus.New(registry),
		},
	)
	deadlineEnforcer := deadlineenforcing.New(
		verdictAdder,
		clock,
		deadlineenforcing.Observers{
			deadlineenforcingobserversapplog.DeadlineLog{},
			deadlineenforcingobserversprometheus.New(registry),
		},
	)
	responder := requestrelay.New(
		deadlineEnforcer,
		cfg.ResponseHeaderTimeout,
		clock,
		requestrelay.Observers{
			requestrelayobserversapplog.RelayLog{},
			requestrelayobserversprometheus.New(registry),
		},
	)
	proxyServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           proxyintake.New(responder, proxyintakeobserversapplog.IntakeLog{}),
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

func egressTransportFor(cfg ServiceConfig) failurereporting.Egress {
	if cfg.EgressProxyDialMode == DialModeAbsoluteURL {
		return absoluteurl.New(cfg.EgressProxyURL)
	}
	return tunnel.New(cfg.EgressProxyURL)
}
