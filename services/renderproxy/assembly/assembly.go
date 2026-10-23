package renderproxyassembly

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/cdprender"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/originpreflight"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/proxyintake"
	rendercapacityobserversapplog "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendercapacityobservers/applog"
	rendercapacityobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendercapacityobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
	renderobserversapplog "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderobservers/applog"
	renderobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "renderproxy started"
	msgServiceStopped = "renderproxy stopped"
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
		Name: "renderproxy_info",
		Help: "Render proxy application identity.",
	}, func() float64 { return 1 }))
	renderObservers := rendergate.RenderObservers{
		renderobserversapplog.RenderLog{},
		renderobserversprometheus.New(registry, cfg.RequestDeadline),
	}
	renderCapacityObservers := rendergate.RenderCapacityObservers{
		rendercapacityobserversapplog.RenderCapacityLog{},
		rendercapacityobserversprometheus.New(registry),
	}
	browser := cdprender.New(ctx, cfg.CDPURL, cfg.EgressProxyURL, cfg.MaxResponseBytes)
	defer browser.Close()

	deadlineRenderer := rendergate.NewDeadlineRenderer(
		originpreflight.New(browser, cfg.EgressProxyURL, cfg.MaxResponseBytes),
		cfg.RequestDeadline,
		renderObservers,
	)
	capacityLimitedRenderer := rendergate.NewCapacityLimitedRenderer(
		deadlineRenderer,
		cfg.RenderConcurrency,
		renderCapacityObservers,
	)
	proxyServer := &http.Server{
		Handler: httpobservation.NewHandler(
			proxyintake.New(capacityLimitedRenderer),
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "renderproxy"),
		),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", listeners.Proxy.Addr().String()),
		slog.Int("renderConcurrency", cfg.RenderConcurrency),
	)

	err := servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "proxy", Listener: listeners.Proxy, Server: proxyServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}
