package remoteindexadmission

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const swapWindowLength = time.Hour

type swapWindow struct {
	StartedAt time.Time
	Swaps     int
}

type swapAllowance struct {
	windows *vault.Collection[string, swapWindow]
	urls    urlmeta.URLDirectory
	now     func() time.Time
	limit   SwapLimit
}

func (a swapAllowance) allows(tx *vault.Txn, network string) (bool, error) {
	window, err := a.currentWindowOf(tx, network)
	if err != nil {
		return false, err
	}
	storedURLs, err := a.urls.Count(tx)
	if err != nil {
		return false, fmt.Errorf("count stored urls: %w", err)
	}

	return window.Swaps < max(
		a.limit.FloorPerHour,
		int(float64(storedURLs)*a.limit.SharePerHour),
	), nil
}

func (a swapAllowance) currentWindowOf(tx *vault.Txn, network string) (swapWindow, error) {
	window, found, err := a.windows.Get(tx, network)
	if err != nil {
		return swapWindow{}, fmt.Errorf("read swap window: %w", err)
	}
	now := a.now()
	if !found || now.Sub(window.StartedAt) >= swapWindowLength {
		return swapWindow{StartedAt: now}, nil
	}

	return window, nil
}

func (a swapAllowance) spend(tx *vault.Txn, network string) error {
	window, err := a.currentWindowOf(tx, network)
	if err != nil {
		return err
	}
	window.Swaps++
	if _, err := a.windows.Put(tx, network, window); err != nil {
		return fmt.Errorf("record swap: %w", err)
	}

	return nil
}
