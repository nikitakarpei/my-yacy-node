package messageadmission

import (
	"sync"
	"time"
)

const (
	messagesPerSenderWindow = 5
	senderWindowDuration    = time.Hour
	trackedSenderCapacity   = 1024
)

type senderAllowance struct {
	mu      sync.Mutex
	now     func() time.Time
	windows map[string]senderWindow
}

type senderWindow struct {
	start    time.Time
	admitted int
}

func newSenderAllowance(now func() time.Time) *senderAllowance {
	return &senderAllowance{now: now, windows: make(map[string]senderWindow)}
}

func (a *senderAllowance) take(sender string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	window, tracked := a.windows[sender]
	if !tracked && a.full() {
		a.forgetWindowsEndedBy(now)
	}
	if !tracked && a.full() {
		return false
	}
	if !tracked || window.endedBy(now) {
		window = senderWindow{start: now}
	}
	if window.admitted >= messagesPerSenderWindow {
		return false
	}

	window.admitted++
	a.windows[sender] = window

	return true
}

func (a *senderAllowance) full() bool {
	return len(a.windows) >= trackedSenderCapacity
}

func (a *senderAllowance) forgetWindowsEndedBy(now time.Time) {
	for sender, window := range a.windows {
		if window.endedBy(now) {
			delete(a.windows, sender)
		}
	}
}

func (w senderWindow) endedBy(now time.Time) bool {
	return now.Sub(w.start) >= senderWindowDuration
}
