package pairingoffers_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
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

func (c *fakeClock) fireEveryStartedEvenStopped() {
	c.mu.Lock()
	started := append([]*timer{}, c.timers...)
	c.mu.Unlock()
	for _, each := range started {
		each.expire()
	}
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
		return nil
	}
}

func openedSecret(
	t *testing.T,
	offers *pairingoffers.Offers,
) (*pairingoffers.Offer, companionprotocol.Secret) {
	t.Helper()
	offer := offers.Open()
	renewed, isRenewal := nextEvent(t, offer).(pairingoffers.SecretRenewed)
	if !isRenewal || renewed.Secret == "" {
		t.Fatalf("first event = %+v, want a secret", renewed)
	}
	return offer, renewed.Secret
}

func begunPairing(
	t *testing.T,
	offers *pairingoffers.Offers,
) (*pairingoffers.Offer, companionprotocol.Secret) {
	t.Helper()
	offer, secret := openedSecret(t, offers)
	if !offers.Begin(context.Background(), secret, "Pixel 8") {
		t.Fatal("the offered secret does not begin a pairing")
	}
	nextEvent(t, offer)
	return offer, secret
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

	if !offers.Begin(t.Context(), secret, "Pixel 8") {
		t.Fatal("the offered secret does not begin a pairing")
	}
	if event := nextEvent(t, offer); event != (pairingoffers.PairingBegun{DeviceName: "Pixel 8"}) {
		t.Errorf("event = %+v, want pairing begun by Pixel 8", event)
	}
	if offers.Begin(t.Context(), secret, "Galaxy Tab S9") {
		t.Error("the secret begins a second pairing")
	}
	if offers.Begin(t.Context(), companionprotocol.RandomSecret(), "Galaxy Tab S9") {
		t.Error("a secret that no page offers begins a pairing")
	}
}

func TestAnExpiredSecretGivesWayToANewOne(t *testing.T) {
	offers, clock := newOffers()
	offer, secret := openedSecret(t, offers)

	clock.fireRunning(t, lifetime)

	renewed, isRenewal := nextEvent(t, offer).(pairingoffers.SecretRenewed)
	if !isRenewal || renewed.Secret == secret {
		t.Fatalf("event = %+v, want a new secret", renewed)
	}
	if offers.Begin(t.Context(), secret, "Pixel 8") {
		t.Error("the expired secret begins a pairing")
	}
}

func TestAPairingWithoutARequestGivesWayToANewSecret(t *testing.T) {
	offers, clock := newOffers()
	offer, _ := begunPairing(t, offers)

	clock.fireRunning(t, pairingWait)

	if _, isRenewal := nextEvent(t, offer).(pairingoffers.SecretRenewed); !isRenewal {
		t.Error("the pairing without a request keeps its secret")
	}
}

func TestAPairingThatWaitsForAVerdictDoesNotExpire(t *testing.T) {
	offers, clock := newOffers()
	offer, secret := begunPairing(t, offers)
	secretID := companionprotocol.SecretIDOf(secret)
	if _, held := offers.HoldForVerdict(t.Context(), secretID); !held {
		t.Fatal("the begun pairing is not held for a verdict")
	}

	clock.fireEveryStartedEvenStopped()
	offers.Accept(t.Context(), secretID, "device-a")

	if _, isPaired := nextEvent(t, offer).(pairingoffers.DevicePaired); !isPaired {
		t.Error("the held pairing expired before its verdict")
	}
}

func TestAnAcceptedDeviceEndsTheOffer(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := begunPairing(t, offers)
	secretID := companionprotocol.SecretIDOf(secret)
	pairing, held := offers.HoldForVerdict(t.Context(), secretID)
	if !held || pairing != (pairingoffers.OngoingPairing{Secret: secret, DeviceName: "Pixel 8"}) {
		t.Fatalf("pairing = %+v (%v), want Pixel 8 with the secret", pairing, held)
	}

	offers.Accept(t.Context(), secretID, "device-a")

	want := pairingoffers.DevicePaired{DeviceName: "Pixel 8", Device: "device-a"}
	if event := nextEvent(t, offer); event != want {
		t.Errorf("event = %+v, want Pixel 8 paired", event)
	}
	if _, held := offers.HoldForVerdict(t.Context(), secretID); held {
		t.Error("the ended offer still pairs")
	}
}

func TestARefusedDeviceGivesWayToANewSecret(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := begunPairing(t, offers)
	secretID := companionprotocol.SecretIDOf(secret)
	offers.HoldForVerdict(t.Context(), secretID)

	offers.Refuse(t.Context(), secretID, companionprotocol.ReasonAppOutdated)

	want := pairingoffers.DeviceRefused{
		DeviceName: "Pixel 8", Reason: companionprotocol.ReasonAppOutdated,
	}
	if refused := nextEvent(t, offer); refused != want {
		t.Errorf("event = %+v, want Pixel 8 refused as outdated", refused)
	}
	renewed, isRenewal := nextEvent(t, offer).(pairingoffers.SecretRenewed)
	if !isRenewal || renewed.Secret == secret {
		t.Errorf("event = %+v, want a new secret", renewed)
	}
}

func TestAPairingRequestIsHeldForOneVerdictOnly(t *testing.T) {
	offers, _ := newOffers()
	_, secret := begunPairing(t, offers)
	secretID := companionprotocol.SecretIDOf(secret)

	offers.HoldForVerdict(t.Context(), secretID)

	if _, held := offers.HoldForVerdict(t.Context(), secretID); held {
		t.Error("a second request is held while the first waits for its verdict")
	}
}

func TestAVerdictOnAPairingThatWaitsForNoneChangesNothing(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := begunPairing(t, offers)
	secretID := companionprotocol.SecretIDOf(secret)

	offers.Accept(t.Context(), secretID, "device-a")
	offers.Refuse(t.Context(), secretID, companionprotocol.ReasonRequestInvalid)

	select {
	case event := <-offer.Events():
		t.Errorf("event = %+v, want none", event)
	default:
	}
}

func TestAPairingThatNeverBeganIsNotHeld(t *testing.T) {
	offers, _ := newOffers()
	_, secret := openedSecret(t, offers)

	if _, held := offers.HoldForVerdict(t.Context(), companionprotocol.SecretIDOf(secret)); held {
		t.Error("a pairing that never began is held")
	}
}

func TestAClosedPageOffersItsSecretNoMore(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := openedSecret(t, offers)

	offer.Close()

	if offers.Begin(t.Context(), secret, "Pixel 8") {
		t.Error("the secret of a closed page begins a pairing")
	}
}

func TestAVerdictAfterThePageClosedTellsNothing(t *testing.T) {
	offers, _ := newOffers()
	offer, secret := begunPairing(t, offers)
	secretID := companionprotocol.SecretIDOf(secret)
	offers.HoldForVerdict(t.Context(), secretID)

	offer.Close()
	offers.Accept(t.Context(), secretID, "device-a")

	select {
	case event := <-offer.Events():
		t.Errorf("event = %+v, want none", event)
	default:
	}
}

func TestAPageThatReadsNoEventsMissesTheEventsPastItsBuffer(t *testing.T) {
	offers, clock := newOffers()
	offer := offers.Open()

	for range 32 {
		clock.fireRunning(t, lifetime)
	}

	told := 0
	for range len(offer.Events()) {
		<-offer.Events()
		told++
	}
	if told == 0 || told >= 33 {
		t.Errorf("told %d events, want a full buffer and the rest missed", told)
	}
}
