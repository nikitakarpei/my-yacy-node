package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const wordsByURLBucket vault.Name = "urlreferences_words"

const wordsByURLVersion = "1"

func registerWordsByURL(
	v *vault.Vault,
) (*vault.ValueIndex[yacymodel.RWIPosting, wordByURL], error) {
	words, err := v.RegisterValueIndex(
		wordsByURLBucket,
		wordsByURLVersion,
		wordByURLKeyLayout,
		wordByURLOf,
	)
	if err != nil {
		return nil, fmt.Errorf("register words by url: %w", err)
	}

	return words, nil
}

var wordByURLKeyParts = vault.PairKey(hashkeypart.URLHash, hashkeypart.Hash)

var wordByURLKeyLayout = wordByURLKeyParts.KeyLayoutFor(
	func(reference wordByURL) (yacymodel.URLHash, yacymodel.Hash) {
		return reference.url, reference.word
	},
	func(url yacymodel.URLHash, word yacymodel.Hash) wordByURL {
		return wordByURL{url: url, word: word}
	},
)

func everyWordReferencing(url yacymodel.URLHash) vault.KeyRange {
	return wordByURLKeyParts.KeysWithFirst(url)
}
