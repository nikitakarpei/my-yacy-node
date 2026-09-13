// Package rwipostingamount knows how many postings this node holds for every
// word. It answers the amount a search reports to the peer that asked, and the
// amount a search weighs the rarity of a word with. It is a projection of the
// postings this node holds.
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
