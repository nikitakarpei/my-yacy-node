package rwipostingamount

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const postingAmountBucket vault.Name = "rwi_amount"

func registerPostingAmounts(
	v *vault.Vault,
) (*vault.Collection[yacymodel.Hash, int], error) {
	amountPerWord, err := v.RegisterCollection(
		postingAmountBucket,
		vault.SingleKey(hashkeypart.Hash).KeyLayout(),
		postingAmountValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register posting amounts: %w", err)
	}

	return amountPerWord, nil
}

var errBadPostingAmount = errors.New("bad posting amount")

type postingAmountValueCodec struct{}

func (postingAmountValueCodec) Encode(amountOfPostings int) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(amountOfPostings)

	return stored.Record(), nil
}

func (postingAmountValueCodec) Decode(raw []byte) (int, error) {
	stored := storedfields.ReaderOf(raw, errBadPostingAmount)
	amountOfPostings := stored.Count("posting amount")

	return amountOfPostings, stored.Err()
}
