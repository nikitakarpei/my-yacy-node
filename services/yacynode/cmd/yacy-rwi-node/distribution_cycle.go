package main

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
)

type distributionCycle struct {
	config     nodeConfig
	self       yacymodel.Hash
	storage    nodeStorage
	roster     peerroster.Roster
	client     *http.Client
	observer   *metrics.DistributionMetrics
	dhtRing    *metrics.DHTRingMetrics
	partitions yacymodel.DHTRingPartitions
}

func (d distributionCycle) assemble() *distributioncycle.Cycle {
	if !d.config.Distribution.Enabled {
		return nil
	}

	exchange := peerwire.NewMessageExchange(d.client)
	eligibility := replicaeligibility.New(
		d.config.Distribution.RecipientCooldown,
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
		d.config.Distribution.Redundancy,
	)
	handoff := postinghandoff.New(
		d.storage.replicas,
		d.storage.postingPurger,
		d.roster,
		d.partitions,
		d.self,
		d.config.Distribution.Redundancy,
	)
	transfers := postingtransfer.New(
		postingcourier.NewHTTP(exchange, d.config.NetworkName, d.self),
		urlmetadatacourier.NewBounded(
			urlmetadatacourier.NewHTTP(exchange, d.config.NetworkName, d.self),
			d.config.Distribution.URLMetadataBatchSize,
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
		d.observer,
		d.dhtRing,
		distributioncycle.Config{
			OfferInterval: postingofferschedule.OfferInterval{
				Shortest: d.config.Distribution.ShortestOfferInterval,
				Longest:  d.config.Distribution.LongestOfferInterval,
			},
			PostingsPerCycle:  d.config.Distribution.PostingsPerCycle,
			CycleInterval:     d.config.Distribution.CycleInterval,
			MinReachablePeers: d.config.Distribution.MinReachablePeers,
		},
	)
}
