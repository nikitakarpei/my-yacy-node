package classifierproxyassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/safetensorsmodel"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/classificationgate"
	classificationgateobserversapplog "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/classificationgateobservers/applog"
	classificationgateobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/classificationgateobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/egresstransports/absoluteurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/egresstransports/tunnel"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/pageclassifier"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/proxyintake"
	proxyintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/proxyintakeobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/requestrelay"
	requestrelayobserversapplog "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/requestrelayobservers/applog"
	requestrelayobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/requestrelayobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/classificationadding"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/deadlineenforcing"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/failurereporting"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/idlecancelling"
	classificationaddingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippersobservers/classificationadding/applog"
	classificationaddingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippersobservers/classificationadding/prometheus"
	deadlineenforcingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippersobservers/deadlineenforcing/applog"
	deadlineenforcingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippersobservers/deadlineenforcing/prometheus"
	failurereportingobserversapplog "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippersobservers/failurereporting/applog"
	failurereportingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippersobservers/failurereporting/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "classifierproxy started"
	msgServiceStopped = "classifierproxy stopped"
)

type Listeners struct {
	Proxy net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	proxy, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Proxy: proxy}, nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "classifierproxy_info",
		Help: "Classifier proxy application identity.",
	}, func() float64 { return 1 }))
	modelPerCategory, err := modelPerCategoryFrom(cfg.SafetensorsModelPathPerCategory)
	if err != nil {
		return err
	}
	clock := wallclock.Clock{}
	classificationRunner := classificationgate.New(
		pageclassifier.New(modelPerCategory),
		runtime.GOMAXPROCS(0),
		clock,
		classificationgate.Observers{
			classificationgateobserversapplog.GateLog{},
			classificationgateobserversprometheus.New(registry),
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
	classificationAdder := classificationadding.New(
		idleCanceller,
		classificationRunner,
		classificationadding.Limits{
			PageByteCeiling:    cfg.PageByteCeiling,
			MaxPagesReadAtOnce: cfg.MaxPagesReadAtOnce,
		},
		clock,
		classificationadding.Observers{
			classificationaddingobserversapplog.ClassificationLog{},
			classificationaddingobserversprometheus.New(registry),
		},
	)
	deadlineEnforcer := deadlineenforcing.New(
		classificationAdder,
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
		Handler:           proxyintake.New(responder, proxyintakeobserversapplog.IntakeLog{}),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", listeners.Proxy.Addr().String()),
		slog.Group("modelVersions", modelVersionAttributesOf(modelPerCategory)...),
	)
	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "proxy", Listener: listeners.Proxy, Server: proxyServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func modelPerCategoryFrom(
	safetensorsModelPathPerCategory map[pageclassification.Category]string,
) (map[pageclassification.Category]classifiermodel.Model, error) {
	modelPerCategory := make(
		map[pageclassification.Category]classifiermodel.Model,
		len(safetensorsModelPathPerCategory),
	)
	for category, safetensorsModelPath := range safetensorsModelPathPerCategory {
		model, err := modelFrom(safetensorsModelPath)
		if err != nil {
			return nil, fmt.Errorf("model of the category %s: %w", category, err)
		}
		modelPerCategory[category] = model
	}
	return modelPerCategory, nil
}

func modelFrom(safetensorsModelPath string) (classifiermodel.Model, error) {
	//nolint:gosec // the operator names the model file
	modelFile, err := os.ReadFile(safetensorsModelPath)
	if err != nil {
		return classifiermodel.Model{}, fmt.Errorf("read: %w", err)
	}
	return safetensorsmodel.ModelFrom(modelFile)
}

func egressTransportFor(cfg ServiceConfig) failurereporting.Egress {
	if cfg.EgressProxyDialMode == DialModeAbsoluteURL {
		return absoluteurl.New(cfg.EgressProxyURL)
	}
	return tunnel.New(cfg.EgressProxyURL)
}

func modelVersionAttributesOf(
	modelPerCategory map[pageclassification.Category]classifiermodel.Model,
) []any {
	attributes := make([]any, 0, len(modelPerCategory))
	for category, model := range modelPerCategory {
		attributes = append(attributes, slog.String(string(category), model.Version))
	}
	return attributes
}
