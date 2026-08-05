package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type node struct {
	peerMux           *http.ServeMux
	sweeper           eviction.Sweeper
	escrow            *rwiescrow.HeldPostings
	announcer         peerannouncement.Announcer
	distributionCycle *distributioncycle.Cycle
	crawl             *crawlRuntime
}

//nolint:revive // argument-limit: explicit, independently-meaningful collaborators
func assembleNode(
	ctx context.Context,
	config nodeConfig,
	vault *vault.Vault,
	client *http.Client,
	offerObserver *metrics.DistributionMetrics,
	dhtRingObserver *metrics.DHTRingMetrics,
	rosterObserver peerroster.RosterObserver,
	escrowObserver rwiescrow.HoldObserver,
	searchObserver *searchmetrics.SearchMetrics,
) (node, error) {
	guard := httpguard.NewRequestGuard(
		httpguard.DefaultMaxBodyBytes,
		httpguard.DefaultRequestTimeout,
	)
	identity := nodeIdentity(config)

	partitions, err := dhtRingPartitionsOf(config)
	if err != nil {
		return node{}, err
	}

	storage, err := openNodeStorage(vault, time.Now, escrowObserver, offerObserver)
	if err != nil {
		return node{}, err
	}

	report := nodestatus.NewReport(identity, storage.postings, storage.urlDirectory)

	gate := httpguard.WireGate{
		Guard:   guard,
		Respond: httpguard.NewWireResponder(report),
		Address: httpguard.NewClientAddressResolver(config.TrustedProxies),
	}

	mux := http.NewServeMux()
	mux.Handle("/{$}", landing.NewEndpoint())
	router := httpguard.NewWireRouter(mux, gate)

	mountNodeEndpoints(router, identity, storage, searchObserver, partitions)

	announcer, roster, err := peerExchange{
		router:         router,
		identity:       identity,
		report:         report,
		config:         config,
		vault:          vault,
		client:         client,
		rosterObserver: rosterObserver,
	}.assemble()
	if err != nil {
		return node{}, err
	}

	sweeper := newStorageSweeper(vault, storage)

	runtime, err := buildCrawlRuntime(ctx, config.Crawl, storage)
	if err != nil {
		return node{}, err
	}

	cycle := distributionCycle{
		config:     config,
		self:       identity.Hash,
		storage:    storage,
		roster:     roster,
		client:     client,
		observer:   offerObserver,
		dhtRing:    dhtRingObserver,
		partitions: partitions,
	}.assemble()

	return node{
		peerMux:           mux,
		sweeper:           sweeper,
		escrow:            storage.escrow,
		announcer:         announcer,
		distributionCycle: cycle,
		crawl:             runtime,
	}, nil
}

func dhtRingPartitionsOf(config nodeConfig) (yacymodel.DHTRingPartitions, error) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(
		config.Distribution.PartitionExponent,
	)
	if err != nil {
		return 0, fmt.Errorf("dht ring partitions: %w", err)
	}

	return partitions, nil
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
