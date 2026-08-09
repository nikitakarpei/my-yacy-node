package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const postingsBucket vault.Name = "rwi"

var postingKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

func registerPostings(
	v *vault.Vault,
) (*vault.Collection[yacymodel.RWIPosting], error) {
	collection, err := vault.Register[yacymodel.RWIPosting](v, postingsBucket, postingCodec{})
	if err != nil {
		return nil, fmt.Errorf("register rwi posting collection: %w", err)
	}

	return collection, nil
}

func postingKey(wordHash yacymodel.Hash, urlHash yacymodel.URLHash) vault.Key {
	return postingKeyLayout.Key(wordHash.String(), urlHash.String()).Bytes()
}

func postingKeyPrefixOfWord(wordHash yacymodel.Hash) vault.Key {
	return postingKeyLayout.First(wordHash.String()).Bytes()
}
