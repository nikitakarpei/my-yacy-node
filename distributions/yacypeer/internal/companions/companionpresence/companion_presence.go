// Package companionpresence watches whether a paired companion is connected to the broker of the
// peer. It tells its observers when the first companion connects and when the last one leaves.
package companionpresence

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

const checkInterval = 2 * time.Second

type DeviceConnections interface {
	ConnectedDevices() map[companionprotocol.DeviceID]bool
}

type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
}

type Observer interface {
	FirstCompanionConnected(ctx context.Context)
	LastCompanionDisconnected(ctx context.Context)
}

type Watcher struct {
	connections        DeviceConnections
	clock              Clock
	observers          []Observer
	companionConnected bool
}

func New(connections DeviceConnections, clock Clock, observers ...Observer) *Watcher {
	return &Watcher{connections: connections, clock: clock, observers: observers}
}

func (w *Watcher) Run(ctx context.Context) error {
	w.check(ctx)
	for w.clock.Sleep(ctx, checkInterval) == nil {
		w.check(ctx)
	}
	return nil
}

func (w *Watcher) check(ctx context.Context) {
	connected := w.anyCompanionConnected()
	if connected == w.companionConnected {
		return
	}
	w.companionConnected = connected
	for _, observer := range w.observers {
		if connected {
			observer.FirstCompanionConnected(ctx)
		} else {
			observer.LastCompanionDisconnected(ctx)
		}
	}
}

func (w *Watcher) anyCompanionConnected() bool {
	return len(w.connections.ConnectedDevices()) > 0
}
