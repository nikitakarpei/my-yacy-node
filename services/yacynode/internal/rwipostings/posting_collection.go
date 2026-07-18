package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const postingsBucket vault.Name = "rwi"

const postingKeyLength = yacymodel.HashLength + yacymodel.HashLength

func registerPostings(
	v *vault.Vault,
) (*vault.Collection[yacymodel.RWIPosting], error) {
	collection, err := vault.Register[yacymodel.RWIPosting](v, postingsBucket, postingCodec{})
	if err != nil {
		return nil, fmt.Errorf("register rwi posting collection: %w", err)
	}

	return collection, nil
}

func postingKey(wordHash, urlHash yacymodel.Hash) vault.Key {
	key := make(vault.Key, 0, postingKeyLength)
	key = append(key, wordHash.String()...)
	key = append(key, urlHash.String()...)

	return key
}
