package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/landing"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	version = "1.83"

	receiveBatchCap       = 1000
	receiveBusyPause      = 30 * time.Second
	searchPostingsPerWord = 1000

	evictionTargetFraction = 0.9
	evictionBatch          = 256
	evictionInterval       = time.Minute

	escrowHoldFor        = 5 * time.Minute
	escrowExpiryBatch    = 256
	escrowExpiryInterval = time.Minute

	serverReadHeaderTimeout = 10 * time.Second
	shutdownTimeout         = 15 * time.Second
)

type node struct {
	peerHandler       http.Handler
	sweeper           eviction.Sweeper
	escrow            *rwiescrow.PostingEscrow
	evictionObserver  *metrics.EvictionMetrics
	escrowObserver    *metrics.RWIEscrowMetrics
	announcer         peerannouncement.Announcer
	distributionCycle *distributioncycle.Cycle
	crawl             *crawlRuntime
}

func RunNode(
	ctx context.Context,
	config NodeConfig,
	vault *vault.Vault,
	registry *prometheus.Registry,
) error {
	metrics.NewVaultCapacityMetrics(registry, vault)
	metrics.NewVaultCollectionMetrics(registry, vault)

	assembled, err := assembleNode(ctx, config, vault, registry)
	if err != nil {
		return fmt.Errorf("assemble node: %w", err)
	}
	metrics.NewRWIEscrowCapacityMetrics(registry, assembled.escrow)
	if assembled.crawl != nil {
		defer assembled.crawl.Close()
	}

	servers := []servergroup.NamedServer{
		{Name: "peer protocol", Server: serverOn(config.PeerAddr, assembled.peerHandler)},
		{Name: "ops", Server: serverOn(config.OpsAddr, opsmetrics.NewMux(
			promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		))},
	}
	for _, server := range servers {
		slog.InfoContext(ctx, "serving",
			slog.String("service", server.Name),
			slog.String("addr", server.Server.Addr),
		)
	}

	return servergroup.Run(ctx, shutdownTimeout, servers, backgroundLoopsOf(assembled)...)
}

func assembleNode(
	ctx context.Context,
	config NodeConfig,
	vault *vault.Vault,
	registry *prometheus.Registry,
) (node, error) {
	identity := nodeIdentity(config)

	partitions, err := dhtRingPartitionsOf(config)
	if err != nil {
		return node{}, err
	}

	egress := newEgressProxyClient(config.ProxyURL, outboundRequestTimeout)
	observers := nodeObserversOn(registry)

	storage, err := openNodeStorage(vault, time.Now, config.Escrow.PostingCapacity, observers)
	if err != nil {
		return node{}, err
	}
	status := nodestatus.NewRuntimeStatus(
		identity,
		time.Now,
		storage.postings,
		storage.urlDirectory,
	)

	mux := http.NewServeMux()
	mux.Handle("/{$}", landing.NewEndpoint())
	router := wireRouterOn(mux, config, status)

	mountNodeEndpoints(
		router,
		identity,
		storage,
		searchmetrics.NewSearchMetrics(registry),
		partitions,
	)

	announcer, roster, err := peerExchange{
		router:         router,
		identity:       identity,
		status:         status,
		config:         config,
		vault:          vault,
		client:         egress,
		rosterObserver: metrics.NewPeerRosterMetrics(registry),
	}.assemble()
	if err != nil {
		return node{}, err
	}

	crawl, err := buildCrawlRuntime(ctx, config.Crawl, storage)
	if err != nil {
		return node{}, err
	}

	cycle := distributionCycle{
		config:     config,
		self:       identity.Hash,
		storage:    storage,
		roster:     roster,
		client:     egress,
		observer:   observers.offers,
		dhtRing:    metrics.NewDHTRingMetrics(registry),
		partitions: partitions,
	}.assemble()

	return node{
		peerHandler: logHTTPRequests(
			instrumentHTTP(metrics.NewHTTPEndpointMetrics(registry), mux),
		),
		sweeper:           newStorageSweeper(vault, storage),
		escrow:            storage.escrow,
		evictionObserver:  metrics.NewEvictionMetrics(registry),
		escrowObserver:    observers.escrow,
		announcer:         announcer,
		distributionCycle: cycle,
		crawl:             crawl,
	}, nil
}

func dhtRingPartitionsOf(config NodeConfig) (yacymodel.DHTRingPartitions, error) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(
		config.Distribution.PartitionExponent,
	)
	if err != nil {
		return 0, fmt.Errorf("dht ring partitions: %w", err)
	}

	return partitions, nil
}

type nodeObservers struct {
	escrow   *metrics.RWIEscrowMetrics
	refusals *metrics.RWIAdmissionMetrics
	offers   *metrics.DistributionMetrics
}

func nodeObserversOn(registry *prometheus.Registry) nodeObservers {
	return nodeObservers{
		escrow:   metrics.NewRWIEscrowMetrics(registry),
		refusals: metrics.NewRWIAdmissionMetrics(registry),
		offers:   metrics.NewDistributionMetrics(registry),
	}
}

func wireRouterOn(
	mux *http.ServeMux,
	config NodeConfig,
	status nodestatus.RuntimeStatus,
) httpguard.WireRouter {
	return httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(status),
		Address: httpguard.NewClientAddressResolver(config.TrustedProxyNetworks),
	})
}

func mountNodeEndpoints(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	storage nodeStorage,
	searchObserver *searchmetrics.SearchMetrics,
	partitions yacymodel.DHTRingPartitions,
) {
	urlmeta.MountTransferURL(router, identity, storage.urlReceiver)
	rwiingress.Mount(router, identity, storage.postingReceiver)
	nodestatus.MountQuery(
		router,
		identity,
		storage.postings,
		storage.references,
		storage.urlDirectory,
	)

	documentsearch.MountSearch(
		router,
		identity,
		storage.postings,
		storage.urlDirectory,
		searchPostingsPerWord,
		searchObserver,
		partitions,
	)
}

func newStorageSweeper(vault *vault.Vault, storage nodeStorage) eviction.Sweeper {
	return eviction.NewSweeper(
		vault,
		storage.postingPurger,
		storage.references,
		storage.urlEvictor,
		storage.staleness,
		eviction.Config{TargetFraction: evictionTargetFraction, BatchSize: evictionBatch},
	)
}

func serverOn(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}
}

func backgroundLoopsOf(assembled node) []func(context.Context) error {
	loops := []func(context.Context) error{
		func(ctx context.Context) error {
			assembled.announcer.Run(ctx)

			return nil
		},
		func(ctx context.Context) error {
			eviction.RunSweepLoop(
				ctx,
				assembled.sweeper,
				assembled.evictionObserver,
				evictionInterval,
			)

			return nil
		},
		func(ctx context.Context) error {
			rwiescrow.RunExpiryLoop(
				ctx,
				assembled.escrow,
				assembled.escrowObserver,
				rwiescrow.ExpiryConfig{
					HoldFor:  escrowHoldFor,
					Interval: escrowExpiryInterval,
					Batch:    escrowExpiryBatch,
				},
			)

			return nil
		},
	}
	if assembled.distributionCycle != nil {
		loops = append(loops, func(ctx context.Context) error {
			assembled.distributionCycle.Run(ctx)

			return nil
		})
	}
	if assembled.crawl != nil {
		loops = append(loops, func(ctx context.Context) error {
			assembled.crawl.Run(ctx)

			return nil
		})
	}

	return loops
}
