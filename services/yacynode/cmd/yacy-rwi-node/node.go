package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/landing"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
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

func assembleNode(
	ctx context.Context,
	config nodeconfiguration.Settings,
	vault *vault.Vault,
	registry *prometheus.Registry,
) (node, error) {
	identity := nodeIdentity(config.Identity)

	partitions, err := dhtRingPartitionsOf(config.Distribution)
	if err != nil {
		return node{}, err
	}

	egress := newEgressProxyClient(config.Egress.ProxyURL, outboundRequestTimeout)
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
	router := wireRouterOn(mux, config.Serving, status)

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
		config:         config.PeerExchange,
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
		config:      config.Distribution,
		networkName: identity.NetworkName,
		self:        identity.Hash,
		storage:     storage,
		roster:      roster,
		client:      egress,
		observer:    observers.offers,
		dhtRing:     metrics.NewDHTRingMetrics(registry),
		partitions:  partitions,
	}.assemble()

	return node{
		peerHandler:       httpobservation.NewHandler(mux, requestObserversOn(registry)...),
		sweeper:           newStorageSweeper(vault, storage),
		escrow:            storage.escrow,
		evictionObserver:  metrics.NewEvictionMetrics(registry),
		escrowObserver:    observers.escrow,
		announcer:         announcer,
		distributionCycle: cycle,
		crawl:             crawl,
	}, nil
}

func dhtRingPartitionsOf(
	config nodeconfiguration.DistributionConfig,
) (yacymodel.DHTRingPartitions, error) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(config.PartitionExponent)
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
