package rwiescrow

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var heldPostingKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type postingIdentity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

type heldPostingKeyCodec struct{}

func (heldPostingKeyCodec) Encode(posting postingIdentity) vaultkey.Key {
	return heldPostingKeyLayout.Key(posting.URL.String(), posting.Word.String())
}

func (heldPostingKeyCodec) Decode(key vaultkey.Key) (postingIdentity, error) {
	url, word, err := heldPostingKeyLayout.Parts(key)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting key: %w", err)
	}

	return parsedIdentity(word, url)
}

func parsedIdentity(word string, url string) (postingIdentity, error) {
	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting word hash: %w", err)
	}
	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting url hash: %w", err)
	}

	return postingIdentity{Word: parsedWord, URL: parsedURL}, nil
}

func heldPostingKeysOf(url yacymodel.URLHash) vaultkey.KeyRange {
	return vaultkey.KeysUnder(heldPostingKeyLayout.First(url.String()))
}
