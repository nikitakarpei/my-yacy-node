package main

import (
	"context"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawling"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch"
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

//nolint:revive // argument-limit: seven explicit, independently-meaningful collaborators
func assembleNode(
	ctx context.Context,
	config nodeConfig,
	vault *vault.Vault,
	client *http.Client,
	offerObserver *metrics.DistributionMetrics,
	rosterObserver peerroster.RosterObserver,
	escrowObserver rwiescrow.HoldObserver,
) (node, error) {
	guard := httpguard.NewRequestGuard(
		httpguard.DefaultMaxBodyBytes,
		httpguard.DefaultRequestTimeout,
	)
	identity := nodeIdentity(config)

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

	mountNodeEndpoints(router, identity, storage)

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

	cycle, err := distributionCycle{
		config:   config,
		self:     identity.Hash,
		storage:  storage,
		roster:   roster,
		client:   client,
		observer: offerObserver,
	}.assemble()
	if err != nil {
		return node{}, err
	}

	return node{
		peerMux:           mux,
		sweeper:           sweeper,
		escrow:            storage.escrow,
		announcer:         announcer,
		distributionCycle: cycle,
		crawl:             runtime,
	}, nil
}

func mountNodeEndpoints(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	storage nodeStorage,
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
	)

	crawling.MountCrawlReceipt(router)
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
