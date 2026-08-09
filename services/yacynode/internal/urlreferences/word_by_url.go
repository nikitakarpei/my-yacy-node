package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var wordByURLKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type wordByURL struct {
	url  yacymodel.URLHash
	word yacymodel.Hash
}

func (w wordByURL) key() vault.Key {
	return wordByURLKeyLayout.Key(w.url.String(), w.word.String()).Bytes()
}

func wordKeyPrefixOfURL(url yacymodel.URLHash) vault.Key {
	return wordByURLKeyLayout.First(url.String()).Bytes()
}

func wordFromKey(key vault.Key) (yacymodel.Hash, error) {
	_, word, err := wordByURLKeyLayout.Parts(vaultkey.KeyFrom(key))
	if err != nil {
		return yacymodel.Hash{}, fmt.Errorf("word by url key: %w", err)
	}

	return yacymodel.ParseHash(word)
}
