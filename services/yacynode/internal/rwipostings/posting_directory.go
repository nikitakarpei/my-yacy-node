package rwipostings

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingIdentity struct {
	word yacymodel.Hash
	url  yacymodel.URLHash
}

type postingDirectory struct {
	vault     *vault.Vault
	postings  *vault.Collection[postingIdentity, yacymodel.RWIPosting]
	observers postingObservers
}

func (d postingDirectory) RWICount(tx *vault.Txn) (int, error) {
	return collectionLength(tx, d.postings)
}

func (d postingDirectory) PostingOf(
	ctx context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	var (
		entry yacymodel.RWIPosting
		found bool
	)
	err := d.vault.View(ctx, func(tx *vault.Txn) error {
		stored, ok, err := d.postings.Get(tx, postingIdentity{word: word, url: url})
		if err != nil {
			return fmt.Errorf("read rwi posting: %w", err)
		}
		if !ok {
			return nil
		}
		stored.WordHash = word
		stored.URLHash = url
		entry, found = stored, true

		return nil
	})
	if err != nil {
		return yacymodel.RWIPosting{}, false, fmt.Errorf("posting: %w", err)
	}

	return entry, found, nil
}

func (d postingDirectory) Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	key := postingIdentity{word: posting.WordHash, url: posting.URLHash}
	if err := d.postings.Put(tx, key, posting); err != nil {
		return fmt.Errorf("store rwi posting: %w", err)
	}
	if err := d.observers.stored(tx, posting.WordHash, posting.URLHash); err != nil {
		return err
	}

	return nil
}

func (d postingDirectory) PurgePosting(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (bool, error) {
	deleted, err := d.postings.Delete(tx, postingIdentity{word: word, url: url})
	if err != nil {
		return false, fmt.Errorf("delete rwi posting: %w", err)
	}
	if err := d.observers.purged(tx, word, url); err != nil {
		return false, err
	}

	return deleted, nil
}

func (d postingDirectory) ScanWord(
	ctx context.Context,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	err := d.vault.View(ctx, func(tx *vault.Txn) error {
		return d.postings.Scan(
			tx,
			everyPostingOf(word),
			func(identity postingIdentity, entry yacymodel.RWIPosting) (bool, error) {
				if err := ctx.Err(); err != nil {
					return false, fmt.Errorf("context: %w", err)
				}
				entry.WordHash = word
				entry.URLHash = identity.url

				return visit(entry)
			},
		)
	})
	if err != nil {
		return fmt.Errorf("scan word postings: %w", err)
	}

	return nil
}

func collectionLength[K, V any](
	tx *vault.Txn,
	collection *vault.Collection[K, V],
) (int, error) {
	length, err := collection.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("read length: %w", err)
	}

	return length, nil
}

var (
	_ PostingIndex    = postingDirectory{}
	_ PostingAdmitter = postingDirectory{}
	_ PostingPurger   = postingDirectory{}
)
