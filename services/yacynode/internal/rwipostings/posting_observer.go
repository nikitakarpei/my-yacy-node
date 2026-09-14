package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingObservers []PostingObserver

func (o postingObservers) stored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	for _, observer := range o {
		if err := observer.PostingStored(tx, posting); err != nil {
			return fmt.Errorf("posting observer: %w", err)
		}
	}

	return nil
}

func (o postingObservers) purged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	for _, observer := range o {
		if err := observer.PostingPurged(tx, posting); err != nil {
			return fmt.Errorf("posting observer: %w", err)
		}
	}

	return nil
}
