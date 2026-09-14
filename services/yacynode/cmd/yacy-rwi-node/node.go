package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/bootstrap"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/landing"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodepeerhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peeradmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferinterval"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiringoccupancy"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

type node struct {
	peerHandler           http.Handler
	evictionSweeper       eviction.Sweeper
	postingEscrow         *rwiescrow.PostingEscrow
	evictionObserver      *metrics.EvictionMetrics
	postingEscrowObserver *metrics.RWIEscrowMetrics
	peerAnnouncer         peerannouncement.Announcer
	distributionCycle     *distributioncycle.Cycle
	pageOfferIntake       *pageOfferIntake
}

const advertisedYaCyRelease = 1.83

const egressRequestTimeout = 30 * time.Second

const (
	peerPostingTransferCapacity = 1000
	postingAdmissionBusyPause   = 30 * time.Second
)

const (
	evictionTargetFraction  = 0.9
	evictionURLsPerBatch    = 256
	evictionBatchesPerSweep = 64
)

const servedURLMetadataPerRequest = 1000

const documentsPerIndexAbstract = 1000

func assembleNode(
	ctx context.Context,
	config nodeconfiguration.Settings,
	vault *vault.Vault,
	registry *prometheus.Registry,
) (node, error) {
	now := time.Now

	selfPeerHash, err := nodepeerhash.Open(vault)
	if err != nil {
		return node{}, fmt.Errorf("open peer hash: %w", err)
	}

	settledPeerHash, err := selfPeerHash.Settle(ctx, config.Identity.InitialHash)
	if err != nil {
		return node{}, fmt.Errorf("settle peer hash: %w", err)
	}

	derivedPeerName, err := nodeidentity.PeerNameFromHash(settledPeerHash)
	if err != nil {
		return node{}, err
	}

	identity := nodeidentity.Identity{
		Hash:        settledPeerHash,
		NetworkName: config.Identity.NetworkName,
		Name:        config.Identity.Name.OrElse(derivedPeerName),
		Host:        config.Identity.AdvertiseHost,
		Port:        config.Identity.AdvertisePort,
		Flags:       config.Identity.Flags,
		Version:     yacymodel.SoftwareVersion{Release: advertisedYaCyRelease},
		Start:       now(),
	}

	slog.InfoContext(ctx, "node identity",
		slog.String("peer", identity.Hash.String()),
		slog.String("name", identity.Name.String()),
	)

	dhtRingPartitions, err := yacymodel.DHTRingPartitionsFromExponent(
		config.Distribution.PartitionExponent,
	)
	if err != nil {
		return node{}, fmt.Errorf("dht ring partitions: %w", err)
	}

	egressClient := &http.Client{
		Timeout:   egressRequestTimeout,
		Transport: &http.Transport{Proxy: http.ProxyURL(config.Egress.ProxyURL)},
	}

	metrics.NewVaultCapacityMetrics(registry, vault)
	metrics.NewVaultCollectionMetrics(registry, vault)

	urlMetadataStaleness, err := urlmetastaleness.Open(vault)
	if err != nil {
		return node{}, fmt.Errorf("url metadata staleness: %w", err)
	}

	urlReferences, err := urlreferences.Open(vault)
	if err != nil {
		return node{}, fmt.Errorf("url references: %w", err)
	}

	postingImpactOrder, err := rwipostingimpactorder.Open(vault)
	if err != nil {
		return node{}, fmt.Errorf("rwi impact order: %w", err)
	}

	postingAmounts, err := rwipostingamount.Open(vault)
	if err != nil {
		return node{}, fmt.Errorf("rwi posting amounts: %w", err)
	}

	ringOccupancy, err := rwiringoccupancy.Open(vault, dhtRingPartitions)
	if err != nil {
		return node{}, fmt.Errorf("rwi ring occupancy: %w", err)
	}
	metrics.NewRWIRingOccupancyMetrics(
		registry,
		vault,
		ringOccupancy,
		yacymodel.DHTRingSectorOf(yacymodel.DHTRingPositionOf(identity.Hash)),
	)

	distributionObserver := metrics.NewDistributionMetrics(registry)

	offerSchedule, postingReplicas, postingRecords, err := rwidistribution.Open(
		vault,
		now,
		distributionObserver,
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi distribution: %w", err)
	}

	postings, postingAdmitter, postingPurger, err := rwipostings.Open(
		vault,
		urlReferences,
		postingRecords,
		postingImpactOrder,
		postingAmounts,
		ringOccupancy,
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi storage: %w", err)
	}

	postingEscrowObserver := metrics.NewRWIEscrowMetrics(registry)

	postingEscrow, err := rwiescrow.Open(
		vault,
		postingAdmitter,
		postingEscrowObserver,
		config.Escrow.PostingCapacity,
		now,
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi escrow: %w", err)
	}
	metrics.NewRWIEscrowCapacityMetrics(registry, vault, postingEscrow)

	urlPostingPurge := urlpostingpurge.New(
		urlReferences,
		postingPurger,
		metrics.NewURLPostingPurgeMetrics(registry),
	)

	urlDirectory, urlEvictor, urlMetadataAdmitter, urlReceiver, err := urlmeta.Open(
		vault,
		urlMetadataStaleness,
		postingEscrow,
		urlPostingPurge,
	)
	if err != nil {
		return node{}, fmt.Errorf("urlmeta storage: %w", err)
	}

	admissionRefusals := metrics.NewRWIAdmissionMetrics(registry)
	postingReceiver := rwiadmission.Open(
		vault,
		urlDirectory,
		postingAdmitter,
		postingEscrow,
		rwiadmission.Config{
			Pause:    postingAdmissionBusyPause,
			Refusals: admissionRefusals,
		},
	)

	pageReceiver := pageadmission.Open(
		vault,
		urlEvictor,
		urlMetadataAdmitter,
		postingAdmitter,
		pageadmission.Config{Pause: postingAdmissionBusyPause},
	)

	runtimeStatus := nodestatus.NewRuntimeStatus(identity, now, vault, postings, urlDirectory)

	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(runtimeStatus),
		Address: httpguard.NewClientAddressResolver(config.Serving.TrustedProxyNetworks),
	})

	mux.Handle("/{$}", landing.NewEndpoint())
	urlmeta.MountTransferURL(router, identity, urlReceiver)
	urlmeta.MountURLMetadataLookup(
		router,
		identity,
		vault,
		urlDirectory,
		servedURLMetadataPerRequest,
	)
	rwiingress.Mount(router, identity, postingReceiver, rwiingress.Config{
		PostingCap: peerPostingTransferCapacity,
		Pause:      postingAdmissionBusyPause,
		Refusals:   admissionRefusals,
	})
	nodestatus.MountQuery(router, identity, vault, postings, urlReferences, urlDirectory)
	documentsearch.MountSearch(
		vault,
		router,
		identity,
		postings,
		postingImpactOrder,
		postingAmounts,
		urlDirectory,
		searchmetrics.NewSearchMetrics(registry),
		dhtRingPartitions,
		documentsPerIndexAbstract,
	)

	peerRoster, err := peerroster.Open(
		vault,
		now,
		config.PeerExchange.KnownRosterCapacity,
		config.PeerExchange.ReachableRosterCapacity,
		config.PeerExchange.AnnounceInterval,
		identity.Hash,
		metrics.NewPeerRosterMetrics(registry),
	)
	if err != nil {
		return node{}, fmt.Errorf("open peer roster: %w", err)
	}

	peeradmission.MountHello(router, identity, runtimeStatus, now, peerRoster, egressClient)

	peerAnnouncer := peerannouncement.New(
		peerannouncement.Config{
			Client:             egressClient,
			NetworkName:        identity.NetworkName,
			Interval:           config.PeerExchange.AnnounceInterval,
			ReachableCap:       config.PeerExchange.ReachableRosterCapacity,
			ContactConcurrency: config.PeerExchange.PeerContactConcurrency,
		},
		runtimeStatus,
		bootstrap.New(egressClient, config.PeerExchange.SeedlistURLs),
		peerRoster,
		metrics.NewPeerAnnouncementMetrics(registry),
	)

	var pageOffers *pageOfferIntake

	if config.PageOfferIntake.Enabled() {
		intake, intakeErr := openPageOfferIntake(
			ctx, config.PageOfferIntake, pageReceiver, registry,
		)
		if intakeErr != nil {
			return node{}, intakeErr
		}
		pageOffers = intake
	}

	var distributionCycle *distributioncycle.Cycle

	if config.Distribution.Enabled {
		peerMessageExchange := peerwire.NewMessageExchange(egressClient)
		replicaEligibility := replicaeligibility.New(config.Distribution.RecipientCooldown, now)
		dhtRingObserver := metrics.NewDHTRingMetrics(registry)

		distributionCycle = distributioncycle.New(
			vault,
			postingoffer.New(
				vault,
				offerSchedule,
				postingReplicas,
				postings,
				peerRoster,
				replicaEligibility,
				dhtRingObserver,
				dhtRingPartitions,
				identity.Hash,
				config.Distribution.Redundancy,
			),
			postinghandoff.New(
				postingReplicas,
				postingPurger,
				peerRoster,
				dhtRingPartitions,
				identity.Hash,
				config.Distribution.Redundancy,
			),
			postingtransfer.New(
				vault,
				postingcourier.New(peerMessageExchange, identity.NetworkName, identity.Hash),
				urlmetadatacourier.NewBounded(
					urlmetadatacourier.New(
						peerMessageExchange,
						identity.NetworkName,
						identity.Hash,
					),
					config.Distribution.URLMetadataBatchSize,
				),
				urlDirectory,
				distributionObserver,
			),
			replicaEligibility,
			postingReplicas,
			offerSchedule,
			peerRoster,
			now,
			distributionObserver,
			dhtRingObserver,
			distributioncycle.Config{
				OfferInterval: postingofferinterval.Bounds{
					Shortest: config.Distribution.OfferInterval.Shortest,
					Longest:  config.Distribution.OfferInterval.Longest,
				},
				PostingsPerBatch:  config.Distribution.PostingsPerBatch,
				CycleInterval:     config.Distribution.CycleInterval,
				DrainBudget:       config.Distribution.DrainBudget,
				MinReachablePeers: config.Distribution.MinReachablePeers,
			},
		)
	}

	return node{
		peerHandler: httpobservation.NewHandler(
			mux,
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "yacynode"),
		),
		evictionSweeper: eviction.NewSweeper(
			vault,
			urlEvictor,
			urlMetadataStaleness,
			eviction.Config{
				TargetFraction:  evictionTargetFraction,
				URLsPerBatch:    evictionURLsPerBatch,
				BatchesPerSweep: evictionBatchesPerSweep,
			},
		),
		postingEscrow:         postingEscrow,
		evictionObserver:      metrics.NewEvictionMetrics(registry),
		postingEscrowObserver: postingEscrowObserver,
		peerAnnouncer:         peerAnnouncer,
		distributionCycle:     distributionCycle,
		pageOfferIntake:       pageOffers,
	}, nil
}
