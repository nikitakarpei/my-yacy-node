// Package rwiringoccupancy knows how far the postings of this node fill every
// sector of the DHT ring. It answers where on the ring the postings of this
// node gather, without any read of the postings themselves. The occupancy is a
// projection of rwi: it changes only inside the transaction that stores or
// purges a posting, so it never drifts from the postings it reports. The ring
// position of a posting depends on the partition count, so a node that starts
// with a different partition exponent records the occupancy of different
// sectors than the postings it already holds.
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
