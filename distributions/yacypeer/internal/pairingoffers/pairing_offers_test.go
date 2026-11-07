package pairingoffers_test

import (
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pairingoffers"
)

const (
	lifetime    = time.Minute
	pairingWait = 30 * time.Second
)

type timer struct {
	expire  func()
	timeout time.Duration
	stopped bool
}

type fakeClock struct {
	timers []*timer
	mu     sync.Mutex
}

func (c *fakeClock) After(timeout time.Duration, expire func()) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	started := &timer{timeout: timeout, expire: expire}
	c.timers = append(c.timers, started)
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		started.stopped = true
	}
}

func (c *fakeClock) fireRunning(t *testing.T, timeout time.Duration) {
	t.Helper()
	c.mu.Lock()
	var due *timer
	for _, running := range c.timers {
		if !running.stopped && running.timeout == timeout {
			due = running
		}
	}
	c.mu.Unlock()
	if due == nil {
		t.Fatalf("no running timer of %v", timeout)
	}
	due.expire()
}

func newOffers() (*pairingoffers.Offers, *fakeClock) {
	clock := &fakeClock{}
	return pairingoffers.New(clock, lifetime, pairingWait), clock
}

func nextEvent(t *testing.T, offer *pairingoffers.Offer) pairingoffers.Event {
	t.Helper()
	select {
	case event := <-offer.Events():
		return event
	default:
		t.Fatal("the offer told nothing")
		return pairingoffers.Event{}
	}
}

func openedSecret(t *testing.T, offers *pairingoffers.Offers) (*pairingoffers.Offer, string) {
	t.Helper()
	offer := offers.Open()
	event := nextEvent(t, offer)
	if event.Kind != pairingoffers.SecretRenewed || event.Secret == "" {
		t.Fatalf("first event = %+v, want a secret", event)
	}
	return offer, event.Secret
}

func TestTwoOpenPagesOfferTwoSecrets(t *testing.T) {
	offers, _ := newOffers()

	_, first := openedSecret(t, offers)
	_, second := openedSecret(t, offers)

	if first == second {
		t.Error("two pages offer the same secret")
	}
}

func TestADeviceBeginsPairingWithASecretOnlyOnce(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := openedSecret(t, offers)

	if !offers.Begin(secret, "Pixel 8") {
		t.Fatal("the offered secret does not begin a pairing")
	}
	if event := nextEvent(t, offer); event.Kind != pairingoffers.PairingBegun ||
		event.DeviceName != "Pixel 8" {
		t.Errorf("event = %+v, want pairing begun by Pixel 8", event)
	}
	if offers.Begin(secret, "Galaxy Tab S9") {
		t.Error("the secret begins a second pairing")
	}
	if offers.Begin(companionprotocol.NewSecret(), "Galaxy Tab S9") {
		t.Error("a secret that no page offers begins a pairing")
	}
}

func TestAnExpiredSecretGivesWayToANewOne(t *testing.T) {
	offers, clock := newOffers()
	offer, secret := openedSecret(t, offers)

	clock.fireRunning(t, lifetime)

	event := nextEvent(t, offer)
	if event.Kind != pairingoffers.SecretRenewed || event.Secret == secret {
		t.Fatalf("event = %+v, want a new secret", event)
	}
	if offers.Begin(secret, "Pixel 8") {
		t.Error("the expired secret begins a pairing")
	}
}

func TestAPairingWithoutOutcomeGivesWayToANewSecret(t *testing.T) {
	offers, clock := newOffers()
	offer, secret := openedSecret(t, offers)
	offers.Begin(secret, "Pixel 8")
	nextEvent(t, offer)

	clock.fireRunning(t, pairingWait)

	if event := nextEvent(t, offer); event.Kind != pairingoffers.SecretRenewed {
		t.Errorf("event = %+v, want a new secret", event)
	}
}

func TestAnAcceptedDeviceEndsTheOffer(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := openedSecret(t, offers)
	offers.Begin(secret, "Pixel 8")
	nextEvent(t, offer)
	secretID := companionprotocol.SecretIDOf(secret)
	pairing, found := offers.PairingWith(secretID)
	if !found || pairing.Secret != secret || pairing.DeviceName != "Pixel 8" {
		t.Fatalf("pairing = %+v (%v), want Pixel 8 with the secret", pairing, found)
	}

	offers.Accept(secretID, "device-a")

	event := nextEvent(t, offer)
	if event.Kind != pairingoffers.DevicePaired || event.Device != "device-a" ||
		event.DeviceName != "Pixel 8" {
		t.Errorf("event = %+v, want Pixel 8 paired", event)
	}
	if _, found := offers.PairingWith(secretID); found {
		t.Error("the ended offer still pairs")
	}
}

func TestARefusedDeviceGivesWayToANewSecret(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := openedSecret(t, offers)
	offers.Begin(secret, "Pixel 8")
	nextEvent(t, offer)

	offers.Refuse(companionprotocol.SecretIDOf(secret), companionprotocol.ReasonAppOutdated)

	refused := nextEvent(t, offer)
	if refused.Kind != pairingoffers.DeviceRefused || refused.DeviceName != "Pixel 8" ||
		refused.Reason != companionprotocol.ReasonAppOutdated {
		t.Errorf("event = %+v, want Pixel 8 refused as outdated", refused)
	}
	if renewed := nextEvent(t, offer); renewed.Kind != pairingoffers.SecretRenewed ||
		renewed.Secret == secret {
		t.Errorf("event = %+v, want a new secret", renewed)
	}
}

func TestAnOutcomeForAPairingThatNeverBeganChangesNothing(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := openedSecret(t, offers)
	secretID := companionprotocol.SecretIDOf(secret)

	offers.Accept(secretID, "device-a")
	offers.Refuse(secretID, companionprotocol.ReasonRequestInvalid)

	select {
	case event := <-offer.Events():
		t.Errorf("event = %+v, want none", event)
	default:
	}
	if _, found := offers.PairingWith(secretID); found {
		t.Error("a pairing that never began is found")
	}
}

func TestAClosedPageOffersItsSecretNoMore(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := openedSecret(t, offers)

	offer.Close()

	if offers.Begin(secret, "Pixel 8") {
		t.Error("the secret of a closed page begins a pairing")
	}
}
