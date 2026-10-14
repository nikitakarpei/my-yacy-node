// Package rwipostingamount knows how many postings this node holds for every
// word. It answers the amount a search reports to the peer that asked, and the
// amount a search weighs the rarity of a word with. It is a projection of the
// postings this node holds. It also estimates how many words the node holds
// postings of: it counts a small index in full, and a sample of a large one.
package rwipostingamount

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PostingAmountQuery interface {
	AmountOfPostingsOf(tx *vault.Txn, word yacymodel.Hash) (int, error)
}

type WordAmountEstimate interface {
	EstimatedAmountOfWords(tx *vault.Txn) (int, error)
}

func Open(
	v *vault.Vault,
) (PostingAmountQuery, WordAmountEstimate, vault.Projection[yacymodel.RWIPosting], error) {
	amountPerWord, err := registerPostingAmounts(v)
	if err != nil {
		return nil, nil, nil, err
	}
	return &postingAmounts{amountPerWord: amountPerWord}, wordCensus{amountPerWord: amountPerWord},
		amountPerWord, nil
}
