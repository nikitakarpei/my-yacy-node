package rwiadmission

import (
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const swapWindowLength = time.Hour

type swapWindow struct {
	startedAt time.Time
	swaps     int
}

type swapAllowance struct {
	mutex             sync.Mutex
	indexedVersions   IndexedVersions
	now               func() time.Time
	floorPerHour      int
	sharePerHour      float64
	windowsPerNetwork map[string]swapWindow
}

func newSwapAllowance(
	indexedVersions IndexedVersions,
	now func() time.Time,
	floorPerHour int,
	sharePerHour float64,
) *swapAllowance {
	return &swapAllowance{
		indexedVersions:   indexedVersions,
		now:               now,
		floorPerHour:      floorPerHour,
		sharePerHour:      sharePerHour,
		windowsPerNetwork: map[string]swapWindow{},
	}
}

func (a *swapAllowance) allows(tx *vault.Txn, network string) (bool, error) {
	indexedURLs, err := a.indexedVersions.Count(tx)
	if err != nil {
		return false, err
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.currentWindowOf(
		network,
	).swaps < max(
		a.floorPerHour,
		int(float64(indexedURLs)*a.sharePerHour),
	), nil
}

func (a *swapAllowance) spend(tx *vault.Txn, network string) {
	tx.RunAfterCommit(func() { a.spendCommitted(network) })
}

func (a *swapAllowance) spendCommitted(network string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	window := a.currentWindowOf(network)
	window.swaps++
	a.windowsPerNetwork[network] = window
}

func (a *swapAllowance) currentWindowOf(network string) swapWindow {
	now := a.now()
	window := a.windowsPerNetwork[network]
	if now.Sub(window.startedAt) >= swapWindowLength {
		return swapWindow{startedAt: now}
	}

	return window
}
