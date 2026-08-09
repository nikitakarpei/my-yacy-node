package urlreferences

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	wordsByURLBucket    vault.Name = "urlreferences_words"
	referencedURLBucket vault.Name = "rwi_refs"
)

type urlReferences struct {
	vault      *vault.Vault
	words      *vault.Set[wordByURL]
	referenced *vault.Set[yacymodel.URLHash]
}

func openURLReferences(v *vault.Vault) (*urlReferences, error) {
	words, err := vault.RegisterSet(v, wordsByURLBucket, wordByURLKeyCodec{})
	if err != nil {
		return nil, fmt.Errorf("register words by url: %w", err)
	}
	referenced, err := vault.RegisterSet(v, referencedURLBucket, referencedURLKeyCodec{})
	if err != nil {
		return nil, fmt.Errorf("register referenced urls: %w", err)
	}

	return &urlReferences{vault: v, words: words, referenced: referenced}, nil
}

func (r *urlReferences) PostingStored(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if err := r.words.Add(tx, wordByURL{url: url, word: word}); err != nil {
		return fmt.Errorf("record word by url: %w", err)
	}
	if err := r.referenced.Add(tx, url); err != nil {
		return fmt.Errorf("record referenced url: %w", err)
	}

	return nil
}

func (r *urlReferences) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if _, err := r.words.Remove(tx, wordByURL{url: url, word: word}); err != nil {
		return fmt.Errorf("drop word by url: %w", err)
	}

	remaining, err := r.WordsReferencing(tx, url)
	if err != nil {
		return err
	}
	if len(remaining) > 0 {
		return nil
	}
	if _, err := r.referenced.Remove(tx, url); err != nil {
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
		wordKeysOf(url),
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

func (r *urlReferences) ReferencedURLCount(ctx context.Context) (int, error) {
	var count int
	err := r.vault.View(ctx, func(tx *vault.Txn) error {
		measured, err := r.referenced.Len(tx)
		if err != nil {
			return fmt.Errorf("read referenced url count: %w", err)
		}
		count = measured

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("referenced url count: %w", err)
	}

	return count, nil
}

var _ ReferenceProjection = (*urlReferences)(nil)
