package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordByURL struct {
	url  yacymodel.URLHash
	word yacymodel.Hash
}

func wordByURLOf(posting yacymodel.RWIPosting) wordByURL {
	return wordByURL{url: posting.URLHash, word: posting.WordHash}
}

type urlReferences struct {
	words *vault.ValueIndex[yacymodel.RWIPosting, wordByURL]
}

func (r *urlReferences) WordsReferencing(
	tx *vault.Txn,
	url yacymodel.URLHash,
) ([]yacymodel.Hash, error) {
	var words []yacymodel.Hash
	err := r.words.Scan(
		tx,
		everyWordReferencing(url),
		func(reference wordByURL) (bool, error) {
			words = append(words, reference.word)

			return true, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("scan words by url: %w", err)
	}

	return words, nil
}

var _ ReferenceQuery = (*urlReferences)(nil)
