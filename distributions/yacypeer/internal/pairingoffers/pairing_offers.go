// Package pairingoffers keeps the pairing secrets that open pairing pages offer. Each open page
// holds one offer. An offer gets a new secret when its secret expires, when a device that began
// pairing gives no outcome in time, and when the peer refuses a device. The offer tells its page
// each new secret, when a device begins pairing, and how the pairing ends.
package pairingoffers

import (
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

const eventBuffer = 16

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type EventKind int

const (
	SecretRenewed EventKind = iota
	PairingBegun
	DevicePaired
	DeviceRefused
)

type Event struct {
	Secret     string
	DeviceName string
	Device     string
	Reason     companionprotocol.Reason
	Kind       EventKind
}

type Pairing struct {
	Secret     string
	DeviceName string
}

type Offers struct {
	clock       Clock
	bySecretID  map[string]*Offer
	lifetime    time.Duration
	pairingWait time.Duration
	mu          sync.Mutex
}

type Offer struct {
	offers     *Offers
	events     chan Event
	stopTimer  func()
	secret     string
	deviceName string
	begun      bool
	ended      bool
}

func New(clock Clock, lifetime time.Duration, pairingWait time.Duration) *Offers {
	return &Offers{
		clock:       clock,
		bySecretID:  map[string]*Offer{},
		lifetime:    lifetime,
		pairingWait: pairingWait,
	}
}

func (o *Offers) Open() *Offer {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := &Offer{offers: o, events: make(chan Event, eventBuffer), stopTimer: func() {}}
	o.renew(offer)
	return offer
}

func (o *Offers) Begin(secret string, deviceName string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.bySecretID[companionprotocol.SecretIDOf(secret)]
	if offer == nil || offer.secret != secret || offer.begun {
		return false
	}
	offer.begun = true
	offer.deviceName = deviceName
	offer.restartTimer(o.pairingWait)
	offer.tell(Event{Kind: PairingBegun, DeviceName: deviceName})
	return true
}

func (o *Offers) PairingWith(secretID string) (Pairing, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.bySecretID[secretID]
	if offer == nil || !offer.begun {
		return Pairing{}, false
	}
	return Pairing{Secret: offer.secret, DeviceName: offer.deviceName}, true
}

func (o *Offers) Accept(secretID string, device string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.bySecretID[secretID]
	if offer == nil || !offer.begun {
		return
	}
	offer.tell(Event{Kind: DevicePaired, Device: device, DeviceName: offer.deviceName})
	o.end(offer)
}

func (o *Offers) Refuse(secretID string, reason companionprotocol.Reason) {
	o.mu.Lock()
	defer o.mu.Unlock()
	offer := o.bySecretID[secretID]
	if offer == nil || !offer.begun {
		return
	}
	offer.tell(Event{Kind: DeviceRefused, DeviceName: offer.deviceName, Reason: reason})
	o.renew(offer)
}

func (o *Offers) renew(offer *Offer) {
	delete(o.bySecretID, companionprotocol.SecretIDOf(offer.secret))
	offer.secret = companionprotocol.NewSecret()
	offer.begun = false
	offer.deviceName = ""
	o.bySecretID[companionprotocol.SecretIDOf(offer.secret)] = offer
	offer.restartTimer(o.lifetime)
	offer.tell(Event{Kind: SecretRenewed, Secret: offer.secret})
}

func (o *Offers) expire(offer *Offer, secret string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if offer.ended || offer.secret != secret {
		return
	}
	o.renew(offer)
}

func (o *Offers) end(offer *Offer) {
	offer.ended = true
	offer.stopTimer()
	delete(o.bySecretID, companionprotocol.SecretIDOf(offer.secret))
}

func (offer *Offer) restartTimer(timeout time.Duration) {
	offer.stopTimer()
	secret := offer.secret
	offer.stopTimer = offer.offers.clock.After(timeout, func() {
		offer.offers.expire(offer, secret)
	})
}

func (offer *Offer) tell(event Event) {
	select {
	case offer.events <- event:
	default:
	}
}

func (offer *Offer) Events() <-chan Event {
	return offer.events
}

func (offer *Offer) Close() {
	offer.offers.mu.Lock()
	defer offer.offers.mu.Unlock()
	offer.offers.end(offer)
}
