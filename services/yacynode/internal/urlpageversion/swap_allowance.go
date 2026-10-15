package urlpageversion

import (
	"sync"
	"time"
)

const swapWindowLength = time.Hour

type swapWindow struct {
	startedAt time.Time
	swaps     int
}

type swapAllowance struct {
	mutex             sync.Mutex
	now               func() time.Time
	floorPerHour      int
	sharePerHour      float64
	windowsPerNetwork map[string]swapWindow
}

func newSwapAllowance(now func() time.Time, floorPerHour int, sharePerHour float64) *swapAllowance {
	return &swapAllowance{
		now:               now,
		floorPerHour:      floorPerHour,
		sharePerHour:      sharePerHour,
		windowsPerNetwork: map[string]swapWindow{},
	}
}

func (a *swapAllowance) allows(network string, indexedURLs int) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.currentWindowOf(
		network,
	).swaps < max(
		a.floorPerHour,
		int(float64(indexedURLs)*a.sharePerHour),
	)
}

func (a *swapAllowance) count(network string) {
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
