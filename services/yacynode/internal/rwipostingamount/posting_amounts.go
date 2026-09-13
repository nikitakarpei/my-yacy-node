package rwipostingamount

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingAmounts struct {
	amountPerWord *vault.Collection[yacymodel.Hash, int]
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
	amountOfPostings, _, err := a.amountPerWord.Get(tx, word)
	if err != nil {
		return 0, fmt.Errorf("read posting amount: %w", err)
	}

	return amountOfPostings, nil
}

func (a *postingAmounts) PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	return a.raiseAmountOfPostingsOf(tx, posting.WordHash)
}

func (a *postingAmounts) raiseAmountOfPostingsOf(tx *vault.Txn, word yacymodel.Hash) error {
	amountOfPostings, err := a.AmountOfPostingsOf(tx, word)
	if err != nil {
		return err
	}
	if _, err := a.amountPerWord.Put(tx, word, amountOfPostings+1); err != nil {
		return fmt.Errorf("record posting amount: %w", err)
	}

	return nil
}

func (a *postingAmounts) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	return a.lowerAmountOfPostingsOf(tx, posting.WordHash)
}

func (a *postingAmounts) lowerAmountOfPostingsOf(tx *vault.Txn, word yacymodel.Hash) error {
	amountOfPostings, err := a.AmountOfPostingsOf(tx, word)
	if err != nil {
		return err
	}
	if amountOfPostings <= 1 {
		if _, err := a.amountPerWord.Delete(tx, word); err != nil {
			return fmt.Errorf("drop posting amount: %w", err)
		}

		return nil
	}
	if _, err := a.amountPerWord.Put(tx, word, amountOfPostings-1); err != nil {
		return fmt.Errorf("record posting amount: %w", err)
	}

	return nil
}

var _ PostingAmountProjection = (*postingAmounts)(nil)
