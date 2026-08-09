package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var wordByURLKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type wordByURL struct {
	url  yacymodel.URLHash
	word yacymodel.Hash
}

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

func wordKeysOf(url yacymodel.URLHash) vaultkey.KeyRange {
	return vaultkey.KeysUnder(wordByURLKeyLayout.First(url.String()))
}
