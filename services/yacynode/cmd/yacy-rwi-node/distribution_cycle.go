package main

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferinterval"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
)

type distributionCycle struct {
	config      nodeconfiguration.DistributionConfig
	networkName string
	self        yacymodel.Hash
	storage     nodeStorage
	roster      peerroster.Roster
	client      *http.Client
	observer    *metrics.DistributionMetrics
	dhtRing     *metrics.DHTRingMetrics
	partitions  yacymodel.DHTRingPartitions
}

func (d distributionCycle) assemble() *distributioncycle.Cycle {
	if !d.config.Enabled {
		return nil
	}

	exchange := peerwire.NewMessageExchange(d.client)
	eligibility := replicaeligibility.New(
		d.config.RecipientCooldown,
		d.storage.now,
	)

	offers := postingoffer.New(
		d.storage.vault,
		d.storage.offerSchedule,
		d.storage.replicas,
		d.storage.postings,
		d.roster,
		eligibility,
		d.dhtRing,
		d.partitions,
		d.self,
		d.config.Redundancy,
	)
	handoff := postinghandoff.New(
		d.storage.replicas,
		d.storage.postingPurger,
		d.roster,
		d.partitions,
		d.self,
		d.config.Redundancy,
	)
	transfers := postingtransfer.New(
		postingcourier.New(exchange, d.networkName, d.self),
		urlmetadatacourier.NewBounded(
			urlmetadatacourier.New(exchange, d.networkName, d.self),
			d.config.URLMetadataBatchSize,
		),
		d.storage.urlDirectory,
		d.observer,
	)

	return distributioncycle.New(
		d.storage.vault,
		offers,
		handoff,
		transfers,
		eligibility,
		d.storage.replicas,
		d.storage.offerSchedule,
		d.roster,
		d.storage.now,
		d.observer,
		d.dhtRing,
		distributioncycle.Config{
			OfferInterval: postingofferinterval.Bounds{
				Shortest: d.config.OfferInterval.Shortest,
				Longest:  d.config.OfferInterval.Longest,
			},
			PostingsPerBatch:  d.config.PostingsPerBatch,
			CycleInterval:     d.config.CycleInterval,
			DrainBudget:       d.config.DrainBudget,
			MinReachablePeers: d.config.MinReachablePeers,
		},
	)
}
