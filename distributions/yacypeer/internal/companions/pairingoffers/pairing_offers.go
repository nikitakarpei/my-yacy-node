// Package pairingoffers keeps the pairing secrets that open pairing pages offer. Each open page
// holds one offer. An offer gets a new secret when its secret expires, when a device that began
// pairing sends no pairing request in time, and when the peer refuses a device. While the peer
// decides on a pairing request, the offer waits for the verdict and does not expire. The offer
// tells its page each new secret, when a device begins pairing, and how the pairing ends.
package pairingoffers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

const eventBuffer = 16

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type OngoingPairing struct {
	Secret     companionprotocol.Secret
	DeviceName companionprotocol.DeviceName
}

type Offers struct {
	clock          Clock
	bySecretID     map[companionprotocol.SecretID]*Offer
	secretLifetime time.Duration
	pairingWait    time.Duration
	mu             sync.Mutex
}

type Offer struct {
	events         chan Event
	stopTimer      func()
	close          func()
	secret         companionprotocol.Secret
	deviceName     companionprotocol.DeviceName
	pairingBegun   bool
	heldForVerdict bool
	closed         bool
}

func New(clock Clock, secretLifetime time.Duration, pairingWait time.Duration) *Offers {
	return &Offers{
		clock:          clock,
		bySecretID:     map[companionprotocol.SecretID]*Offer{},
		secretLifetime: secretLifetime,
		pairingWait:    pairingWait,
	}
}

func (o *Offers) Open() *Offer {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := &Offer{events: make(chan Event, eventBuffer), stopTimer: func() {}}
	offer.close = func() { o.close(offer) }
	o.renew(context.Background(), offer)
	return offer
}

func (o *Offers) Begin(
	ctx context.Context,
	secret companionprotocol.Secret,
	deviceName companionprotocol.DeviceName,
) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.bySecretID[companionprotocol.SecretIDOf(secret)]
	switch {
	case offer == nil || offer.secret != secret:
		slog.WarnContext(ctx, "no open pairing page offers the secret of the device",
			slog.String("deviceName", string(deviceName)),
		)
		return false
	case offer.pairingBegun:
		slog.WarnContext(ctx, "another device already pairs with the secret of the device",
			slog.String("deviceName", string(deviceName)),
			slog.String("pairingDeviceName", string(offer.deviceName)),
		)
		return false
	}
	offer.pairingBegun = true
	offer.deviceName = deviceName
	o.restartTimerOf(offer, o.pairingWait)
	offer.tell(ctx, PairingBegun{DeviceName: deviceName})
	return true
}

func (o *Offers) HoldForVerdict(
	ctx context.Context,
	secretID companionprotocol.SecretID,
) (OngoingPairing, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.bySecretID[secretID]
	switch {
	case offer == nil || !offer.pairingBegun:
		slog.WarnContext(ctx, "no pairing waits for the pairing request",
			slog.String("secretId", string(secretID)),
		)
		return OngoingPairing{}, false
	case offer.heldForVerdict:
		slog.WarnContext(ctx, "the pairing already waits for a verdict",
			slog.String("deviceName", string(offer.deviceName)),
		)
		return OngoingPairing{}, false
	}
	offer.heldForVerdict = true
	offer.stopTimer()
	return OngoingPairing{Secret: offer.secret, DeviceName: offer.deviceName}, true
}

func (o *Offers) Accept(
	ctx context.Context,
	secretID companionprotocol.SecretID,
	device companionprotocol.DeviceID,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.heldOfferWith(ctx, secretID)
	if offer == nil {
		return
	}
	offer.tell(ctx, DevicePaired{DeviceName: offer.deviceName, Device: device})
	o.end(offer)
}

func (o *Offers) heldOfferWith(
	ctx context.Context,
	secretID companionprotocol.SecretID,
) *Offer {
	offer := o.bySecretID[secretID]
	switch {
	case offer == nil:
		slog.DebugContext(ctx, "the pairing page closed before the verdict",
			slog.String("secretId", string(secretID)),
		)
		return nil
	case !offer.heldForVerdict:
		slog.WarnContext(ctx, "a verdict on a pairing that waits for none",
			slog.String("secretId", string(secretID)),
		)
		return nil
	}
	return offer
}

func (o *Offers) Refuse(
	ctx context.Context,
	secretID companionprotocol.SecretID,
	reason companionprotocol.RefusalReason,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.heldOfferWith(ctx, secretID)
	if offer == nil {
		return
	}
	offer.tell(ctx, DeviceRefused{DeviceName: offer.deviceName, Reason: reason})
	o.renew(ctx, offer)
}

func (o *Offers) renew(ctx context.Context, offer *Offer) {
	delete(o.bySecretID, companionprotocol.SecretIDOf(offer.secret))
	offer.secret = companionprotocol.RandomSecret()
	offer.pairingBegun = false
	offer.heldForVerdict = false
	offer.deviceName = ""
	o.bySecretID[companionprotocol.SecretIDOf(offer.secret)] = offer
	o.restartTimerOf(offer, o.secretLifetime)
	offer.tell(ctx, SecretRenewed{Secret: offer.secret})
}

func (o *Offers) expire(offer *Offer, secret companionprotocol.Secret) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if offer.closed || offer.heldForVerdict || offer.secret != secret {
		return
	}
	o.renew(context.Background(), offer)
}

func (o *Offers) end(offer *Offer) {
	offer.closed = true
	offer.stopTimer()
	delete(o.bySecretID, companionprotocol.SecretIDOf(offer.secret))
}

func (o *Offers) restartTimerOf(offer *Offer, timeout time.Duration) {
	offer.stopTimer()
	secret := offer.secret
	offer.stopTimer = o.clock.After(timeout, func() {
		o.expire(offer, secret)
	})
}

func (o *Offers) close(offer *Offer) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.end(offer)
}

func (offer *Offer) tell(ctx context.Context, event Event) {
	select {
	case offer.events <- event:
	default:
		slog.WarnContext(ctx, "the pairing page missed an event",
			slog.String("event", fmt.Sprintf("%T", event)),
		)
	}
}

func (offer *Offer) Events() <-chan Event {
	return offer.events
}

func (offer *Offer) Close() {
	offer.close()
}
