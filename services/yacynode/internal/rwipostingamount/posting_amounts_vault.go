package rwipostingamount

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const postingAmountBucket vault.Name = "rwi_amount"

const postingAmountVersion = "1"

var wordKeys = vault.SingleKey(hashkeypart.Hash)

func registerPostingAmounts(
	v *vault.Vault,
) (*vault.CountIndex[yacymodel.RWIPosting, yacymodel.Hash], error) {
	amountPerWord, err := v.RegisterCountIndex(
		postingAmountBucket,
		postingAmountVersion,
		wordKeys.KeyLayout(),
		wordOf,
	)
	if err != nil {
		return nil, fmt.Errorf("register posting amounts: %w", err)
	}

	return amountPerWord, nil
}

func wordOf(posting yacymodel.RWIPosting) yacymodel.Hash {
	return posting.WordHash
}
