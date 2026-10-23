package frontier

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	pageVisitPauseBucket = "page_visit_pause"
	pageVisitPauseKey    = "every-page-visit"
)

type pageVisitPause struct {
	pauses *vault.Set[string]
}

func registerPageVisitPause(storage *vault.Vault) (pageVisitPause, error) {
	pauses, err := storage.RegisterSet(
		pageVisitPauseBucket, vault.SingleKey(vault.TextKeyPart).KeyLayout(),
	)
	if err != nil {
		return pageVisitPause{}, fmt.Errorf("register %s: %w", pageVisitPauseBucket, err)
	}
	return pageVisitPause{pauses: pauses}, nil
}

func (pause pageVisitPause) isOn(tx *vault.Txn) (bool, error) {
	amountOfPauses, err := pause.pauses.Len(tx)
	return amountOfPauses > 0, err
}

func (pause pageVisitPause) turnOn(tx *vault.Txn) error {
	_, err := pause.pauses.Add(tx, pageVisitPauseKey)
	return err
}

func (pause pageVisitPause) turnOff(tx *vault.Txn) error {
	_, err := pause.pauses.Remove(tx, pageVisitPauseKey)
	return err
}
