// Package rwipostingamount knows how many postings this node holds for every
// word. It answers the amount a search reports to the peer that asked, and the
// amount a search weighs how rare a word is with, without any read of the
// postings themselves. The amount is a projection of rwi: it changes only
// inside the transaction that stores or purges a posting, so it never drifts
// from the postings it counts.
package rwipostingamount

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type PostingAmountQuery interface {
	AmountOfPostingsOf(tx *vault.Txn, word yacymodel.Hash) (int, error)
}

type PostingAmountProjection interface {
	PostingAmountQuery
	rwipostings.PostingObserver
}

func Open(v *vault.Vault) (PostingAmountProjection, error) {
	return openPostingAmounts(v)
}
