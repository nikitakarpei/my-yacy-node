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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionhandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiringoccupancy"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/stopwordhashes"
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
	peerPostingTransferCap    = 1000
	postingAdmissionBusyPause = 30 * time.Second
)

const (
	evictionTargetFraction  = 0.9
	evictionURLsPerBatch    = 16
	evictionBatchesPerSweep = 64
)

const servedURLMetadataPerRequest = 1000

// (200000-byte YaCy answer limit − about 40000 bytes for 10 resources and other fields) / 14 bytes per document on a distinct host
const indexAbstractDocumentsPerAnswer = 10000

func assembleNode(
	ctx context.Context,
	config nodeconfiguration.Settings,
	storage *vault.Vault,
	registry *prometheus.Registry,
) (node, error) {
	now := time.Now

	selfPeerHash, err := nodepeerhash.Open(storage)
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
		Hash:         settledPeerHash,
		NetworkName:  config.Identity.NetworkName,
		Name:         config.Identity.Name.OrElse(derivedPeerName),
		Host:         config.Identity.AdvertiseHost,
		Port:         config.Identity.AdvertisePort,
		Capabilities: config.Identity.Capabilities,
		Version:      yacymodel.SoftwareVersion{Release: advertisedYaCyRelease},
		Start:        now(),
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

	metrics.NewVaultCapacityMetrics(registry, storage)
	metrics.NewVaultBucketRecordMetrics(registry, storage)

	urlMetadataStaleness, stalenessProjection, err := urlmetastaleness.Open(storage)
	if err != nil {
		return node{}, fmt.Errorf("url metadata staleness: %w", err)
	}

	urlReferences, urlReferenceProjection, err := urlreferences.Open(storage)
	if err != nil {
		return node{}, fmt.Errorf("url references: %w", err)
	}

	postingImpactOrder, postingImpactProjection, err := rwipostingimpactorder.Open(storage)
	if err != nil {
		return node{}, fmt.Errorf("rwi impact order: %w", err)
	}

	postingAmounts, postingAmountProjection, err := rwipostingamount.Open(storage)
	if err != nil {
		return node{}, fmt.Errorf("rwi posting amounts: %w", err)
	}

	ringOccupancy, ringOccupancyProjection, err := rwiringoccupancy.Open(
		storage,
		dhtRingPartitions,
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi ring occupancy: %w", err)
	}
	metrics.NewRWIRingOccupancyMetrics(
		registry,
		storage,
		ringOccupancy,
		yacymodel.DHTRingSectorOf(yacymodel.DHTRingPositionOf(identity.Hash)),
	)

	undeliveredPostings, undeliveredPostingProjection, wordPartitionHolders, err := openDistributionRecords(
		storage,
		dhtRingPartitions,
		config.Distribution.RefreshInterval,
		config.Distribution.RecipientCooldown,
		now,
	)
	if err != nil {
		return node{}, err
	}

	postings, postingAdmitter, postingPurger, err := rwipostings.Open(
		ctx,
		storage,
		dhtRingPartitions,
		[]vault.Projection[yacymodel.RWIPosting]{
			urlReferenceProjection,
			postingImpactProjection,
			postingAmountProjection,
			ringOccupancyProjection,
			undeliveredPostingProjection,
		},
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi storage: %w", err)
	}

	postingEscrowObserver := metrics.NewRWIEscrowMetrics(registry)

	postingEscrow, err := rwiescrow.Open(
		ctx,
		storage,
		postingAdmitter,
		postingEscrowObserver,
		config.Escrow.PostingCapacity,
		now,
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi escrow: %w", err)
	}
	metrics.NewRWIEscrowCapacityMetrics(registry, storage, postingEscrow)

	urlPostingPurge := urlpostingpurge.New(
		urlReferences,
		postingPurger,
		metrics.NewURLPostingPurgeMetrics(registry),
	)

	urlDirectory, urlEvictor, urlMetadataAdmitter, urlReceiver, err := urlmeta.Open(
		ctx,
		storage,
		[]vault.Projection[yacymodel.URLMetadata]{stalenessProjection},
		postingEscrow,
		urlPostingPurge,
	)
	if err != nil {
		return node{}, fmt.Errorf("urlmeta storage: %w", err)
	}

	stopwordHashes := stopwordhashes.New()
	admissionRefusals := metrics.NewRWIAdmissionMetrics(registry)
	postingReceiver := rwiadmission.Open(
		storage,
		urlDirectory,
		postingAdmitter,
		postingEscrow,
		stopwordHashes,
		rwiadmission.Config{
			PostingCap: peerPostingTransferCap,
			Pause:      postingAdmissionBusyPause,
			Observer:   admissionRefusals,
		},
	)

	pageReceiver := pageadmission.Open(
		storage,
		urlEvictor,
		urlMetadataAdmitter,
		postingAdmitter,
		stopwordHashes,
		pageadmission.Config{Pause: postingAdmissionBusyPause},
	)

	runtimeStatus := nodestatus.NewRuntimeStatus(identity, now, storage, postings, urlDirectory)

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
		storage,
		urlDirectory,
		servedURLMetadataPerRequest,
	)
	rwiingress.Mount(router, identity, postingReceiver)
	nodestatus.MountQuery(router, identity, now, storage, postings, postingAmounts, urlDirectory)
	documentsearch.MountSearch(
		storage,
		router,
		identity,
		postings,
		postingImpactOrder,
		postingAmounts,
		urlDirectory,
		searchmetrics.NewSearchMetrics(registry),
		dhtRingPartitions,
		indexAbstractDocumentsPerAnswer,
	)

	peerRoster, err := peerroster.Open(
		storage,
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
		var handoff distributioncycle.WordPartitionHandoff = wordpartitionhandoff.KeepEveryWordPartition{}
		if config.Distribution.HandoffEnabled {
			handoff = wordpartitionhandoff.New(
				postings,
				postingPurger,
				undeliveredPostings,
				wordPartitionHolders,
				wordpartitionhandoff.Config{
					Partitions: dhtRingPartitions,
					Self:       identity.Hash,
					Redundancy: config.Distribution.Redundancy,
				},
			)
		}
		peerMessageExchange := peerwire.NewMessageExchange(egressClient)
		recipientCooldowns := replicaeligibility.New(config.Distribution.RecipientCooldown, now)
		distributionObserver := metrics.NewDistributionMetrics(registry)
		dhtRingObserver := metrics.NewDHTRingMetrics(registry)

		distributionCycle = distributioncycle.New(
			storage,
			wordpartitionoffer.New(
				storage,
				undeliveredPostings,
				wordPartitionHolders,
				postings,
				peerRoster,
				recipientCooldowns,
				dhtRingObserver,
				wordpartitionoffer.Config{
					Partitions:       dhtRingPartitions,
					Self:             identity.Hash,
					Redundancy:       config.Distribution.Redundancy,
					PostingsPerBatch: config.Distribution.PostingsPerBatch,
				},
			),
			undeliveredPostings,
			wordPartitionHolders,
			handoff,
			postingtransfer.New(
				storage,
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
			recipientCooldowns,
			peerRoster,
			now,
			distributionObserver,
			dhtRingObserver,
			distributioncycle.Config{
				PostingsPerBatch:  config.Distribution.PostingsPerBatch,
				BatchConcurrency:  config.Distribution.BatchConcurrency,
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
			storage,
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

func openDistributionRecords(
	storage *vault.Vault,
	dhtRingPartitions yacymodel.DHTRingPartitions,
	refreshInterval time.Duration,
	retryDelay time.Duration,
	now func() time.Time,
) (
	*undeliveredpostings.UndeliveredPostings,
	vault.Projection[yacymodel.RWIPosting],
	*wordpartitionholders.Holders,
	error,
) {
	undeliveredPostings, undeliveredPostingProjection, err := undeliveredpostings.Open(
		storage,
		dhtRingPartitions,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("undelivered postings: %w", err)
	}

	wordPartitionHolders, err := wordpartitionholders.Open(
		storage,
		dhtRingPartitions,
		refreshInterval,
		retryDelay,
		now,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("word partition holders: %w", err)
	}

	return undeliveredPostings, undeliveredPostingProjection, wordPartitionHolders, nil
}
