// Package rwiringoccupancy knows how far the postings of this node fill every
// sector of the DHT ring. It answers the metrics of this node with where on
// the ring its postings gather. It is a projection of the postings this node
// holds.
package rwiringoccupancy

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type RingOccupancyQuery interface {
	OccupancyOfDHTRingSectors(
		tx *vault.Txn,
	) (map[yacymodel.DHTRingSector]int, error)
}

type RingOccupancyProjection interface {
	RingOccupancyQuery
	rwipostings.PostingObserver
}

func Open(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (RingOccupancyProjection, error) {
	return openRingOccupancy(v, partitions)
}
