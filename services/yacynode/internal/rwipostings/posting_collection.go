package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const postingsBucket vault.Name = "rwi"

func registerPostings(
	v *vault.Vault,
) (*vault.Collection[postingIdentity, yacymodel.RWIPosting], error) {
	collection, err := vault.Register(
		v,
		postingsBucket,
		postingKeyCodec{},
		postingValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register rwi posting collection: %w", err)
	}

	return collection, nil
}
