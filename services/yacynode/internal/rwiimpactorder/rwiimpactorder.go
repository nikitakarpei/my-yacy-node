// Package rwiimpactorder keeps, for every word, the postings of that word in
// order of impact, the most relevant posting first. The impact of a posting
// comes from its hits, saturated so that many hits do not dominate, and rises
// when the word appears in the title of the document. It is fixed when the
// posting is stored. A search reads a word in this order, so it finds the best
// documents of that word without reading every posting of it. The order is a
// projection of rwi: it changes only inside the transaction that stores or
// purges a posting, so it never drifts from the postings it mirrors.
package rwiimpactorder

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
