package rwipostingamount

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingAmounts struct {
	amountPerWord *vault.Amounts[yacymodel.Hash]
}

func openPostingAmounts(v *vault.Vault) (*postingAmounts, error) {
	amountPerWord, err := registerPostingAmounts(v)
	if err != nil {
		return nil, err
	}

	return &postingAmounts{amountPerWord: amountPerWord}, nil
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

func (a *postingAmounts) PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if err := a.amountPerWord.Raise(tx, posting.WordHash, 1); err != nil {
		return fmt.Errorf("raise posting amount: %w", err)
	}

	return nil
}

func (a *postingAmounts) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if err := a.amountPerWord.Lower(tx, posting.WordHash, 1); err != nil {
		return fmt.Errorf("lower posting amount: %w", err)
	}

	return nil
}

var _ PostingAmountProjection = (*postingAmounts)(nil)
