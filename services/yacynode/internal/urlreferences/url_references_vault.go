package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
)

const (
	wordsByURLBucket    vault.Name = "urlreferences_words"
	referencedURLBucket vault.Name = "rwi_refs"
)

func registerURLReferences(
	v *vault.Vault,
) (*vault.Set[wordByURL], *vault.Set[yacymodel.URLHash], error) {
	words, err := v.RegisterSet(wordsByURLBucket, wordByURLKeyCodec)
	if err != nil {
		return nil, nil, fmt.Errorf("register words by url: %w", err)
	}
	referenced, err := v.RegisterSet(referencedURLBucket, referencedURLKeyCodec)
	if err != nil {
		return nil, nil, fmt.Errorf("register referenced urls: %w", err)
	}

	return words, referenced, nil
}

var wordByURLKeyLayout = vault.PairKey(hashcodec.URLHash, hashcodec.Hash)

var wordByURLKeyCodec = wordByURLKeyLayout.KeyCodecFor(
	func(reference wordByURL) (yacymodel.URLHash, yacymodel.Hash) {
		return reference.url, reference.word
	},
	func(url yacymodel.URLHash, word yacymodel.Hash) wordByURL {
		return wordByURL{url: url, word: word}
	},
)

func everyWordReferencing(url yacymodel.URLHash) vault.KeyRange {
	return wordByURLKeyLayout.KeysWithFirst(url)
}

var referencedURLKeyCodec = vault.SingleKey(hashcodec.URLHash).KeyCodec()
