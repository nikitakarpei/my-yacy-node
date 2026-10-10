package rwipostingamount

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const postingAmountBucket vault.Name = "rwi_amount"

func registerPostingAmounts(v *vault.Vault) (*vault.Amounts[yacymodel.Hash], error) {
	amountPerWord, err := v.RegisterAmounts(
		postingAmountBucket,
		vault.SingleKey(hashkeypart.Hash).KeyLayout(),
	)
	if err != nil {
		return nil, fmt.Errorf("register posting amounts: %w", err)
	}

	return amountPerWord, nil
}
