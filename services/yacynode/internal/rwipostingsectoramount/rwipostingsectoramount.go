// Package rwipostingsectoramount knows how many postings this node holds in
// every sector of the DHT ring. It answers where on the ring the postings of
// this node gather, without any read of the postings themselves. The amount is
// a projection of rwi: it changes only inside the transaction that stores or
// purges a posting, so it never drifts from the postings it counts. The ring
// position of a posting depends on the partition count, so a node that starts
// with a different partition exponent counts into different sectors than the
// postings it already holds.
package rwipostingsectoramount

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type PostingSectorAmountQuery interface {
	AmountOfPostingsPerDHTRingSector(
		tx *vault.Txn,
	) (map[yacymodel.DHTRingSector]int, error)
}

type PostingSectorAmountProjection interface {
	PostingSectorAmountQuery
	rwipostings.PostingObserver
}

func Open(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (PostingSectorAmountProjection, error) {
	return openPostingSectorAmounts(v, partitions)
}
