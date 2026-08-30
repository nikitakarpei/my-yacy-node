package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const (
	wordsByURLBucket    vault.Name = "urlreferences_words"
	referencedURLBucket vault.Name = "rwi_refs"
)

func registerURLReferences(
	v *vault.Vault,
) (*vault.Set[wordByURL], *vault.Set[yacymodel.URLHash], error) {
	words, err := v.RegisterSet(wordsByURLBucket, wordByURLKeyLayout)
	if err != nil {
		return nil, nil, fmt.Errorf("register words by url: %w", err)
	}
	referenced, err := v.RegisterSet(referencedURLBucket, referencedURLKeyLayout)
	if err != nil {
		return nil, nil, fmt.Errorf("register referenced urls: %w", err)
	}

	return words, referenced, nil
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

var referencedURLKeyLayout = vault.SingleKey(hashkeypart.URLHash).KeyLayout()
