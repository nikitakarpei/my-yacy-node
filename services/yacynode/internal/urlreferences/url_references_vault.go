package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const (
	wordsByURLBucket    vault.Name = "urlreferences_words"
	referencedURLBucket vault.Name = "rwi_refs"
)

func registerURLReferences(
	v *vault.Vault,
) (*vault.Set[wordByURL], *vault.Set[yacymodel.URLHash], error) {
	words, err := vault.RegisterSet(v, wordsByURLBucket, wordByURLKeyCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register words by url: %w", err)
	}
	referenced, err := vault.RegisterSet(v, referencedURLBucket, referencedURLKeyCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register referenced urls: %w", err)
	}

	return words, referenced, nil
}

var wordByURLKeyLayout = vaultkey.Pair(hashcodec.URLHash, hashcodec.Hash)

type wordByURLKeyCodec struct{}

func (wordByURLKeyCodec) Encode(reference wordByURL) vaultkey.Key {
	return wordByURLKeyLayout.Key(reference.url, reference.word)
}

func (wordByURLKeyCodec) Decode(storedKey []byte) (wordByURL, error) {
	url, word, err := wordByURLKeyLayout.Parts(storedKey)
	if err != nil {
		return wordByURL{}, fmt.Errorf("word by url key: %w", err)
	}

	return wordByURL{url: url, word: word}, nil
}

func wordByURLKeysOf(url yacymodel.URLHash) vaultkey.KeyRange {
	return wordByURLKeyLayout.KeysWithFirst(url)
}

var referencedURLKeyLayout = vaultkey.Single(hashcodec.URLHash)

type referencedURLKeyCodec struct{}

func (referencedURLKeyCodec) Encode(url yacymodel.URLHash) vaultkey.Key {
	return referencedURLKeyLayout.Key(url)
}

func (referencedURLKeyCodec) Decode(storedKey []byte) (yacymodel.URLHash, error) {
	url, err := referencedURLKeyLayout.Parts(storedKey)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("referenced url key: %w", err)
	}

	return url, nil
}
