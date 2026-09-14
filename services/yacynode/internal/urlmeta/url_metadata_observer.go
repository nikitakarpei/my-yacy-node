package urlmeta

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type observers []URLMetadataObserver

func (o observers) stored(
	tx *vault.Txn,
	hash yacymodel.URLHash,
	freshness yacymodel.Optional[yacymodel.CalendarDay],
) error {
	for _, observer := range o {
		if err := observer.URLStored(tx, hash, freshness); err != nil {
			return fmt.Errorf("url metadata observer: %w", err)
		}
	}

	return nil
}

func (o observers) purged(tx *vault.Txn, hash yacymodel.URLHash) error {
	for _, observer := range o {
		if err := observer.URLPurged(tx, hash); err != nil {
			return fmt.Errorf("url metadata observer: %w", err)
		}
	}

	return nil
}
