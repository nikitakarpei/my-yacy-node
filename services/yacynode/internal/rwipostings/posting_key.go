package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var postingKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type postingIdentity struct {
	word yacymodel.Hash
	url  yacymodel.URLHash
}

type postingKeyCodec struct{}

func (postingKeyCodec) Encode(posting postingIdentity) vaultkey.Key {
	return postingKeyLayout.Key(posting.word.String(), posting.url.String())
}

func (postingKeyCodec) Decode(key vaultkey.Key) (postingIdentity, error) {
	word, url, err := postingKeyLayout.Parts(key)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("rwi posting key: %w", err)
	}

	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("rwi posting word hash: %w", err)
	}
	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("rwi posting url hash: %w", err)
	}

	return postingIdentity{word: parsedWord, url: parsedURL}, nil
}

func postingKeysOf(word yacymodel.Hash) vaultkey.KeyRange {
	return vaultkey.KeysUnder(postingKeyLayout.First(word.String()))
}
