// Package rwipostingimpactorder keeps the postings of every word in order of
// impact, the most relevant posting first. It answers a search with the best
// documents of a word, without a read of every posting of that word. It is a
// projection of the postings this node holds.
package rwipostingimpactorder

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type ImpactOrderQuery interface {
	ScanPostingsInImpactOrder(
		tx *vault.Txn,
		word yacymodel.Hash,
		visit func(document yacymodel.URLHash, impact Impact) (bool, error),
	) error
	LargestImpactOf(tx *vault.Txn, word yacymodel.Hash) (Impact, bool, error)
}

type ImpactOrderProjection interface {
	ImpactOrderQuery
	rwipostings.PostingObserver
}

func Open(v *vault.Vault) (ImpactOrderProjection, error) {
	return openImpactOrder(v)
}
