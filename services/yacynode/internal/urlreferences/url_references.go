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

type urlReferences struct {
	words *vault.Set[wordByURL]
}

func openURLReferences(v *vault.Vault) (*urlReferences, error) {
	words, err := registerWordsByURL(v)
	if err != nil {
		return nil, err
	}

	return &urlReferences{words: words}, nil
}

func (r *urlReferences) PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	reference := wordByURL{url: posting.URLHash, word: posting.WordHash}
	if _, err := r.words.Add(tx, reference); err != nil {
		return fmt.Errorf("record word by url: %w", err)
	}

	return nil
}

func (r *urlReferences) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	purged := wordByURL{url: posting.URLHash, word: posting.WordHash}
	if _, err := r.words.Remove(tx, purged); err != nil {
		return fmt.Errorf("drop word by url: %w", err)
	}

	return nil
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

var _ ReferenceProjection = (*urlReferences)(nil)
