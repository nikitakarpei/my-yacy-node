package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawling"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/landing"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type node struct {
	peerMux      *http.ServeMux
	sweeper      eviction.Sweeper
	escrow       *rwiescrow.HeldPostings
	announcer    peerannouncement.Announcer
	distribution rwidistribution.Runner
	crawl        *crawlRuntime
}

//nolint:revive // argument-limit: seven explicit, independently-meaningful collaborators
func assembleNode(
	ctx context.Context,
	config nodeConfig,
	vault *vault.Vault,
	client *http.Client,
	offerObserver rwidistribution.PostingOfferCycleObserver,
	rosterObserver peerroster.RosterObserver,
	escrowObserver rwiescrow.HoldObserver,
) (node, error) {
	guard := httpguard.NewRequestGuard(
		httpguard.DefaultMaxBodyBytes,
		httpguard.DefaultRequestTimeout,
	)
	identity := nodeIdentity(config)

	storage, err := openNodeStorage(vault, escrowObserver)
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

	distribution, err := assembleDistribution(
		config,
		identity.Hash,
		storage,
		roster,
		client,
		offerObserver,
	)
	if err != nil {
		return node{}, err
	}

	return node{
		peerMux:      mux,
		sweeper:      sweeper,
		escrow:       storage.escrow,
		announcer:    announcer,
		distribution: distribution,
		crawl:        runtime,
	}, nil
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func assembleDistribution(
	config nodeConfig,
	self yacymodel.Hash,
	storage nodeStorage,
	roster peerroster.Roster,
	client *http.Client,
	offerObserver rwidistribution.PostingOfferCycleObserver,
) (rwidistribution.Runner, error) {
	if !config.Distribution.Enabled {
		return nil, nil
	}

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(
		config.Distribution.PartitionExponent,
	)
	if err != nil {
		return nil, fmt.Errorf("rwi distribution partitions: %w", err)
	}

	return storage.distribution.Cycle(
		client,
		storage.postings,
		storage.postingPurger,
		roster,
		storage.urlDirectory,
		offerObserver,
		rwidistribution.Config{
			NetworkName:          config.NetworkName,
			Self:                 self,
			Redundancy:           config.Distribution.Redundancy,
			Partitions:           partitions,
			PostingsPerCycle:     config.Distribution.PostingsPerCycle,
			CycleInterval:        config.Distribution.CycleInterval,
			RefreshInterval:      config.Distribution.RefreshInterval,
			RetryInterval:        config.Distribution.RetryInterval,
			RecipientCooldown:    config.Distribution.RecipientCooldown,
			MinReachablePeers:    config.Distribution.MinReachablePeers,
			URLMetadataBatchSize: config.Distribution.URLMetadataBatchSize,
		},
	), nil
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
