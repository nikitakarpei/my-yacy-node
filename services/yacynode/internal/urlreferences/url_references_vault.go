package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

var wordByURLKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type wordByURLKeyCodec struct{}

func (wordByURLKeyCodec) Encode(reference wordByURL) vaultkey.Key {
	return wordByURLKeyLayout.Key(reference.url.String(), reference.word.String())
}

func (wordByURLKeyCodec) Decode(key vaultkey.Key) (wordByURL, error) {
	url, word, err := wordByURLKeyLayout.Parts(key)
	if err != nil {
		return wordByURL{}, fmt.Errorf("word by url key: %w", err)
	}

	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return wordByURL{}, fmt.Errorf("word by url url hash: %w", err)
	}
	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return wordByURL{}, fmt.Errorf("word by url word hash: %w", err)
	}

	return wordByURL{url: parsedURL, word: parsedWord}, nil
}

func wordByURLKeysOf(url yacymodel.URLHash) vaultkey.KeyRange {
	return wordByURLKeyLayout.KeysWithFirst(url.String())
}

var referencedURLKeyLayout = vaultkey.Single(vaultkey.Text)

type referencedURLKeyCodec struct{}

func (referencedURLKeyCodec) Encode(url yacymodel.URLHash) vaultkey.Key {
	return referencedURLKeyLayout.Key(url.String())
}

func (referencedURLKeyCodec) Decode(key vaultkey.Key) (yacymodel.URLHash, error) {
	url, err := referencedURLKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("referenced url key: %w", err)
	}

	return yacymodel.ParseURLHash(url)
}
