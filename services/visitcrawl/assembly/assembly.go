package visitcrawlassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	backgroundcrawlorderplacementobserversapplog "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/backgroundcrawlorderplacementobservers/applog"
	backgroundcrawlorderplacementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/backgroundcrawlorderplacementobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderbroker"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderplacers/background"
	crawlorderpublicationobserversapplog "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderpublicationobservers/applog"
	crawlorderpublicationobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderpublicationobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitintake"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "visitcrawl started"
	msgServiceStopped = "visitcrawl stopped"
)

type Listeners struct {
	Intake net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	intake, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Intake: intake}, nil
}

func (l Listeners) Close() error {
	if err := l.Intake.Close(); err != nil {
		return fmt.Errorf("close the intake listener: %w", err)
	}
	return nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "visitcrawl_info",
		Help: "Visit crawl application identity.",
	}, func() float64 { return 1 }))
	broker, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:            cfg.CrawlNATSURL,
		CrawlOrdersSubject: cfg.CrawlOrdersSubject,
	}, crawlorderbroker.CrawlOrderPublicationObservers{
		crawlorderpublicationobserversapplog.CrawlOrderPublicationLog{},
		crawlorderpublicationobserversprometheus.New(registry),
	})
	if err != nil {
		return fmt.Errorf("open crawl order broker: %w", err)
	}
	defer broker.Close()

	placer := background.New(
		broker.OrderPlacer,
		background.BackgroundCrawlOrderPlacementObservers{
			backgroundcrawlorderplacementobserversapplog.BackgroundCrawlOrderPlacementLog{},
			backgroundcrawlorderplacementobserversprometheus.New(registry),
		},
		cfg.OrderTimeout,
		cfg.MaxInFlight,
	)

	mux := http.NewServeMux()
	visitintake.MountVisitIntake(
		mux,
		placer,
		cfg.CrawlProfile,
		cfg.LinkSecret,
	)

	publicServer := &http.Server{
		Handler: http.MaxBytesHandler(
			httpobservation.NewHandler(
				mux,
				httpaccesslog.New(),
				httpmetrics.NewEndpointMetrics(registry, "visitcrawl"),
			),
			cfg.MaxBodyBytes,
		),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("orders", cfg.CrawlOrdersSubject),
		slog.String("listenAddr", listeners.Intake.Addr().String()),
	)

	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "intake", Listener: listeners.Intake, Server: publicServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}
