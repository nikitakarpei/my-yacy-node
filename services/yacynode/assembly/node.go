package yacynodeassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/bootstrap"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/landing"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/messageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/messageinboxes/webhook"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodepeerhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peeradmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peertls"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/publicaddress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributionloop"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/peereligibility"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionhandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiringoccupancy"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/stopwordhashes"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/wordranking"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type node struct {
	peerHandler           http.Handler
	evictionSweeper       eviction.Sweeper
	postingEscrow         *rwiescrow.PostingEscrow
	evictionObserver      *metrics.EvictionMetrics
	postingEscrowObserver *metrics.RWIEscrowMetrics
	peerAnnouncer         peerannouncement.Announcer
	distributionLoop      *distributionloop.Loop
	pageOfferIntake       *pageOfferIntake
}

const advertisedYaCyRelease = 1.83

const egressRequestTimeout = 30 * time.Second

const (
	peerPostingTransferCap      = 1000
	postingAdmissionBusyPause   = 30 * time.Second
	pageVersionSwapFloorPerHour = 100
	pageVersionSwapSharePerHour = 0.01
)

const wordSampleLifetime = 5 * time.Minute

const (
	evictionTargetFraction  = 0.9
	evictionURLsPerBatch    = 16
	evictionBatchesPerSweep = 64
)

const servedURLMetadataPerRequest = 1000

// (200000-byte YaCy answer limit − about 40000 bytes for 10 resources and other fields) / 14 bytes per document on a distinct host
const indexAbstractDocumentsPerAnswer = 10000

//nolint:gocognit // composition root: one error check per opened unit, no branching logic
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

	keyHash := config.Identity.PeerKey.Hash()
	if err := selfPeerHash.Settle(ctx, keyHash); err != nil {
		return node{}, fmt.Errorf("settle peer hash: %w", err)
	}

	derivedPeerName, err := nodeidentity.PeerNameFromHash(keyHash)
	if err != nil {
		return node{}, err
	}

	identity := nodeidentity.Identity{
		Hash:            keyHash,
		NetworkName:     config.Identity.NetworkName,
		Name:            config.Identity.Name.OrElse(derivedPeerName),
		AdditionalHosts: config.Identity.AdvertiseAdditionalHosts,
		Port:            config.Identity.AdvertisePort,
		SecurePort:      config.Identity.AdvertiseSecurePort,
		Capabilities:    config.Identity.Capabilities,
		Version:         yacymodel.SoftwareVersion{Release: advertisedYaCyRelease},
		Start:           now(),
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

	vaultmetrics.NewCapacityMetrics(registry, metricNamespace, storage)
	vaultmetrics.NewBucketRecordMetrics(registry, metricNamespace, storage)

	urlMetadataStaleness, stalenessProjection, err := urlmetastaleness.Open(storage)
	if err != nil {
		return node{}, fmt.Errorf("url metadata staleness: %w", err)
	}

	postingImpactOrder, postingImpactProjection, err := rwipostingimpactorder.Open(storage)
	if err != nil {
		return node{}, fmt.Errorf("rwi impact order: %w", err)
	}

	postingAmounts, wordAmounts, postingAmountProjection, err := rwipostingamount.Open(
		storage,
		now,
		wordSampleLifetime,
	)
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

	undeliveredPostings, undeliveredPostingProjection, wordPartitionSchedule, err := openDistributionRecords(
		ctx,
		storage,
		dhtRingPartitions,
		config.Distribution.RefreshInterval,
		config.Distribution.PeerPause,
		now,
	)
	if err != nil {
		return node{}, err
	}

	rankedWords, err := wordranking.WordsFrom(config.Storage.WordRankingFile)
	if err != nil {
		return node{}, err
	}
	postings, postingWriter, postingPurger, err := rwipostings.Open(
		ctx,
		storage,
		dhtRingPartitions,
		rankedWords,
		vaultmetrics.NewMemberSetMetrics(registry, metricNamespace),
		[]vault.Projection[yacymodel.RWIPosting]{
			postingImpactProjection,
			postingAmountProjection,
			ringOccupancyProjection,
			undeliveredPostingProjection,
		},
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi storage: %w", err)
	}
	loopRelay := &distributionLoopRelay{}
	postingWriter = wordPartitionSchedule.WriterFrom(postingWriter, loopRelay)

	postingEscrowObserver := metrics.NewRWIEscrowMetrics(registry)

	postingEscrow, err := rwiescrow.Open(
		ctx,
		storage,
		config.Escrow.PostingCapacity,
		now,
	)
	if err != nil {
		return node{}, fmt.Errorf("rwi escrow: %w", err)
	}
	metrics.NewRWIEscrowCapacityMetrics(registry, storage, postingEscrow)

	urlPostingPurge := urlpostingpurge.New(
		postingPurger,
		metrics.NewURLPostingPurgeMetrics(registry),
	)

	urlDirectory, urlEvictor, urlMetadataWriter, err := urlmeta.Open(
		ctx,
		storage,
		[]vault.Projection[yacymodel.URLMetadata]{stalenessProjection},
		urlPostingPurge,
	)
	if err != nil {
		return node{}, fmt.Errorf("urlmeta storage: %w", err)
	}

	stopwordHashes := stopwordhashes.New()
	postingAdmission, urlMetadataIntake, err := remoteindexadmission.Open(
		storage,
		urlDirectory,
		urlEvictor,
		urlMetadataWriter,
		postingEscrow,
		postings,
		postingWriter,
		stopwordHashes,
		metrics.NewRemoteIndexAdmissionMetrics(registry),
		now,
		remoteindexadmission.Config{
			PostingCap: peerPostingTransferCap,
			Pause:      postingAdmissionBusyPause,
			SwapLimit: remoteindexadmission.SwapLimit{
				FloorPerHour: pageVersionSwapFloorPerHour,
				SharePerHour: pageVersionSwapSharePerHour,
			},
		},
	)
	if err != nil {
		return node{}, fmt.Errorf("remote index admission: %w", err)
	}

	pageReceiver := pageadmission.Open(
		storage,
		urlEvictor,
		urlMetadataWriter,
		postingWriter,
		stopwordHashes,
		pageadmission.Config{Pause: postingAdmissionBusyPause},
	)

	publicAddressSource := publicaddress.For(config.Identity.AdvertiseHost)
	runtimeStatus := nodestatus.NewRuntimeStatus(
		identity,
		config.Identity.PeerKey,
		publicAddressSource,
		now,
		storage,
		wordAmounts,
		urlDirectory,
	)

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
	urlmeta.MountURLMetadataLookup(
		router,
		identity,
		storage,
		urlDirectory,
		servedURLMetadataPerRequest,
	)
	remoteindexingress.Mount(router, identity, postingAdmission, urlMetadataIntake)
	nodestatus.MountQuery(router, identity, now, storage, wordAmounts, postingAmounts, urlDirectory)
	mountMessageAdmission(router, identity, now, config.MessageInbox, config.Egress.ProxyURL)
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
		identity.NetworkName,
		rosterObserverFanOut{
			metrics: metrics.NewPeerRosterMetrics(registry),
			relay:   loopRelay,
		},
	)
	if err != nil {
		return node{}, fmt.Errorf("open peer roster: %w", err)
	}

	peerWalker := peerwire.NewWalker(
		egressClient,
		peertls.NewTransports(http.ProxyURL(config.Egress.ProxyURL), identity.NetworkName),
		wallclock.Clock{},
	)
	peeradmission.MountHello(
		router,
		identity,
		runtimeStatus,
		wallclock.Clock{},
		peerRoster,
		peerWalker,
	)

	peerAnnouncer := peerannouncement.New(
		peerannouncement.Config{
			NetworkName:        identity.NetworkName,
			Interval:           config.PeerExchange.AnnounceInterval,
			ReachableCap:       config.PeerExchange.ReachableRosterCapacity,
			ContactConcurrency: config.PeerExchange.PeerContactConcurrency,
		},
		peerWalker,
		runtimeStatus,
		publicAddressSource,
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

	var distributionLoop *distributionloop.Loop

	if config.Distribution.Enabled {
		var handoff distributionloop.WordPartitionHandoff = wordpartitionhandoff.KeepEveryWordPartition{}
		if config.Distribution.HandoffEnabled {
			handoff = wordpartitionhandoff.New(
				postingPurger,
				undeliveredPostings,
				wordPartitionSchedule,
				wordpartitionhandoff.Config{
					Partitions: dhtRingPartitions,
					Self:       identity.Hash,
					Redundancy: config.Distribution.Redundancy,
				},
			)
		}
		distributionObserver := metrics.NewDistributionMetrics(registry)
		peerEligibility := peereligibility.New(
			peereligibility.Config{
				LeastPause:                      config.Distribution.PeerPause,
				UnacceptedOffersBeforeExclusion: config.Distribution.UnacceptedOffersBeforeExclusion,
				ExclusionDuration:               config.Distribution.PeerExclusionDuration,
				WorkInFlight:                    config.Distribution.PeerWorkInFlight,
			},
			now,
			distributionObserver,
		)
		dhtRingObserver := metrics.NewDHTRingMetrics(registry)

		distributionLoop = distributionloop.New(
			storage,
			wordpartitionoffer.New(
				storage,
				undeliveredPostings,
				wordPartitionSchedule,
				postings,
				peerRoster,
				peerEligibility,
				wordpartitionoffer.Config{
					Partitions: dhtRingPartitions,
					Self:       identity.Hash,
					Redundancy: config.Distribution.Redundancy,
				},
			),
			undeliveredPostings,
			wordPartitionSchedule,
			handoff,
			postingtransfer.New(
				storage,
				postingcourier.New(
					peerWalker,
					peerRoster,
					identity.NetworkName,
					identity.Hash,
					distributionObserver,
				),
				urlmetadatacourier.NewBounded(
					urlmetadatacourier.New(
						peerWalker,
						peerRoster,
						identity.NetworkName,
						identity.Hash,
						distributionObserver,
					),
					config.Distribution.URLMetadataBatchSize,
				),
				urlDirectory,
				distributionObserver,
			),
			peerEligibility,
			peerRoster,
			wallclock.Clock{},
			distributionObserver,
			distributionObserver,
			dhtRingObserver,
			distributionloop.Config{
				PostingsPerBatch:    config.Distribution.PostingsPerBatch,
				PostingsPerTransfer: yacyproto.MaxTransferRWIEntryCount,
				AmountOfWorkers:     config.Distribution.AmountOfWorkers,
				MinReachablePeers:   config.Distribution.MinReachablePeers,
			},
		)
		loopRelay.forwardTo(distributionLoop)
	}

	return node{
		peerHandler: httpobservation.NewHandler(
			mux,
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, metricNamespace),
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
		distributionLoop:      distributionLoop,
		pageOfferIntake:       pageOffers,
	}, nil
}

func openDistributionRecords(
	ctx context.Context,
	storage *vault.Vault,
	dhtRingPartitions yacymodel.DHTRingPartitions,
	refreshInterval time.Duration,
	firstRetryDelay time.Duration,
	now func() time.Time,
) (
	*undeliveredpostings.Postings,
	vault.Projection[yacymodel.RWIPosting],
	*wordpartitionschedule.Schedule,
	error,
) {
	undeliveredPostings, undeliveredPostingProjection, err := undeliveredpostings.Open(
		storage,
		dhtRingPartitions,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("undelivered postings: %w", err)
	}

	wordPartitionSchedule, err := wordpartitionschedule.Open(
		ctx,
		storage,
		dhtRingPartitions,
		refreshInterval,
		firstRetryDelay,
		now,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("word partition schedule: %w", err)
	}

	return undeliveredPostings, undeliveredPostingProjection, wordPartitionSchedule, nil
}

func mountMessageAdmission(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	now func() time.Time,
	inbox nodeconfiguration.MessageInboxConfig,
	egressProxy *url.URL,
) {
	if !inbox.Enabled() {
		messageadmission.MountRefusing(router)

		return
	}

	messageadmission.MountAccepting(
		router,
		identity,
		now,
		webhook.New(inbox.WebhookURL, egressProxy, net.DefaultResolver),
	)
}
