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
	words      *vault.Set[wordByURL]
	referenced *vault.Set[yacymodel.URLHash]
}

func openURLReferences(v *vault.Vault) (*urlReferences, error) {
	words, referenced, err := registerURLReferences(v)
	if err != nil {
		return nil, err
	}

	return &urlReferences{words: words, referenced: referenced}, nil
}

func (r *urlReferences) PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	reference := wordByURL{url: posting.URLHash, word: posting.WordHash}
	if _, err := r.words.Add(tx, reference); err != nil {
		return fmt.Errorf("record word by url: %w", err)
	}
	if _, err := r.referenced.Add(tx, posting.URLHash); err != nil {
		return fmt.Errorf("record referenced url: %w", err)
	}

	return nil
}

func (r *urlReferences) PostingUpdated(
	*vault.Txn,
	yacymodel.RWIPosting,
	yacymodel.RWIPosting,
) error {
	return nil
}

func (r *urlReferences) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	reference := wordByURL{url: posting.URLHash, word: posting.WordHash}
	if _, err := r.words.Remove(tx, reference); err != nil {
		return fmt.Errorf("drop word by url: %w", err)
	}

	remaining, err := r.WordsReferencing(tx, posting.URLHash)
	if err != nil {
		return err
	}
	if len(remaining) > 0 {
		return nil
	}
	if _, err := r.referenced.Remove(tx, posting.URLHash); err != nil {
		return fmt.Errorf("drop referenced url: %w", err)
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

func (r *urlReferences) ReferencedURLCount(tx *vault.Txn) (int, error) {
	count, err := r.referenced.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("read referenced url count: %w", err)
	}

	return count, nil
}

var _ ReferenceProjection = (*urlReferences)(nil)
