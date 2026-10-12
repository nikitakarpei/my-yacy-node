package rwipostingamount

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingAmounts struct {
	amountPerWord *vault.CountIndex[yacymodel.RWIPosting, yacymodel.Hash]
}

func (a *postingAmounts) AmountOfPostingsOf(
	tx *vault.Txn,
	word yacymodel.Hash,
) (int, error) {
	amountOfPostings, err := a.amountPerWord.Get(tx, word)
	if err != nil {
		return 0, fmt.Errorf("read posting amount: %w", err)
	}

	return amountOfPostings, nil
}

var _ PostingAmountQuery = (*postingAmounts)(nil)
