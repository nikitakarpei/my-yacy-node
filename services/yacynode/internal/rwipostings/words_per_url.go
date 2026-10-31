package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordsPerURL struct {
	index *vault.MemberSetIndex[yacymodel.Hash, yacymodel.URLHash, postingIdentity]
}

func (w wordsPerURL) wordsOf(tx *vault.Txn, url yacymodel.URLHash) ([]yacymodel.Hash, error) {
	words, err := w.index.MembersOf(tx, url)
	if err != nil {
		return nil, fmt.Errorf("read words of a url: %w", err)
	}

	return words, nil
}
