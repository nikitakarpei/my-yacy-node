package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingObservers []PostingObserver

func (o postingObservers) stored(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	for _, observer := range o {
		if err := observer.PostingStored(tx, word, url); err != nil {
			return fmt.Errorf("posting observer: %w", err)
		}
	}

	return nil
}

func (o postingObservers) purged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	for _, observer := range o {
		if err := observer.PostingPurged(tx, word, url); err != nil {
			return fmt.Errorf("posting observer: %w", err)
		}
	}

	return nil
}
