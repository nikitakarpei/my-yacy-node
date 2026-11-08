// Package companionpresence watches whether a paired companion is connected to the broker of the
// peer. It tells its observers when the first companion connects and when the last one leaves.
package companionpresence

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
)

const checkInterval = 2 * time.Second

type Broker interface {
	ConnectedUsers() map[string]bool
}

type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
}

type Observer interface {
	CompanionsConnected(ctx context.Context)
	CompanionsLeft(ctx context.Context)
}

type Presence struct {
	broker    Broker
	clock     Clock
	observers []Observer
	present   bool
}

func New(broker Broker, clock Clock, observers ...Observer) *Presence {
	return &Presence{broker: broker, clock: clock, observers: observers}
}

func (p *Presence) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		p.check(ctx)
		_ = p.clock.Sleep(ctx, checkInterval)
	}
	return nil
}

func (p *Presence) check(ctx context.Context) {
	present := p.companionConnected()
	if present == p.present {
		return
	}
	p.present = present
	for _, observer := range p.observers {
		if present {
			observer.CompanionsConnected(ctx)
		} else {
			observer.CompanionsLeft(ctx)
		}
	}
}

func (p *Presence) companionConnected() bool {
	for user := range p.broker.ConnectedUsers() {
		if companionaccess.IsDeviceUser(user) {
			return true
		}
	}
	return false
}
