package remoteindexadmission

import (
	"context"
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
	vault   *vault.Vault
	windows *vault.Collection[string, swapWindow]
	urls    urlmeta.URLDirectory
	now     func() time.Time
	limit   SwapLimit
}

func (a swapAllowance) hourlyLimitOf(ctx context.Context) (int, error) {
	var storedURLs int
	err := a.vault.View(ctx, func(tx *vault.Txn) error {
		count, err := a.urls.Count(tx)
		storedURLs = count

		return err
	})
	if err != nil {
		return 0, fmt.Errorf("count stored urls: %w", err)
	}

	return max(a.limit.FloorPerHour, int(float64(storedURLs)*a.limit.SharePerHour)), nil
}

func (a swapAllowance) allows(tx *vault.Txn, network string, hourlyLimit int) (bool, error) {
	window, err := a.currentWindowOf(tx, network)
	if err != nil {
		return false, err
	}

	return window.Swaps < hourlyLimit, nil
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
