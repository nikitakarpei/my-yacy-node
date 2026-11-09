package fetchingduty_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/fetchingduty"
)

var (
	onlyCompanionsFetch = companionsettings.Fetching{CompanionsFetch: true}
	peerFillsIn         = companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}
)

const admissionWait = 5 * time.Second

func peerFetches(t *testing.T, duty *fetchingduty.Duty) bool {
	t.Helper()
	admission := make(chan error, 1)
	go func() { admission <- duty.AwaitAdmission(t.Context()) }()
	select {
	case err := <-admission:
		return err == nil
	case <-time.After(admissionWait):
		return false
	}
}

func peerHoldsBack(duty *fetchingduty.Duty) bool {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return errors.Is(duty.AwaitAdmission(ctx), context.Canceled)
}

type waitingContext struct {
	context.Context

	waiting chan struct{}
}

func (c waitingContext) Done() <-chan struct{} {
	select {
	case c.waiting <- struct{}{}:
	default:
	}
	return c.Context.Done()
}

func pairedDuty() *fetchingduty.Duty {
	duty := fetchingduty.New()
	duty.FirstDevicePaired(context.Background())
	return duty
}

func TestThePeerFetchesWhileNoDeviceIsPaired(t *testing.T) {
	duty := fetchingduty.New()

	duty.FetchingChanged(context.Background(), onlyCompanionsFetch)

	if !peerFetches(t, duty) {
		t.Error("the intake is held back although no device is paired")
	}
}

func TestThePeerFetchesOnceTheLastDeviceIsUnpaired(t *testing.T) {
	duty := pairedDuty()
	duty.FetchingChanged(context.Background(), onlyCompanionsFetch)
	if !peerHoldsBack(duty) {
		t.Fatal("the intake is admitted before the last device is unpaired")
	}

	duty.LastDeviceUnpaired(context.Background())

	if !peerFetches(t, duty) {
		t.Error("the intake is held back after the last device is unpaired")
	}
}

func TestThePeerFetchesWhileCompanionsDoNotFetch(t *testing.T) {
	duty := pairedDuty()
	duty.FirstCompanionConnected(context.Background())

	duty.FetchingChanged(context.Background(), companionsettings.Fetching{PeerFillsIn: true})

	if !peerFetches(t, duty) {
		t.Error("the intake is held back while companions do not fetch")
	}
}

func TestThePeerFetchesWhileItFillsInForAbsentCompanions(t *testing.T) {
	duty := pairedDuty()

	duty.FetchingChanged(context.Background(), peerFillsIn)

	if !peerFetches(t, duty) {
		t.Error("the intake is held back although no companion is connected")
	}
}

func TestThePeerFetchesOnceTheLastCompanionDisconnects(t *testing.T) {
	duty := pairedDuty()
	duty.FetchingChanged(context.Background(), peerFillsIn)
	duty.FirstCompanionConnected(context.Background())
	if !peerHoldsBack(duty) {
		t.Fatal("the intake is admitted before the last companion disconnects")
	}
	waiting := make(chan struct{}, 1)
	admission := make(chan error, 1)
	go func() {
		admission <- duty.AwaitAdmission(waitingContext{Context: t.Context(), waiting: waiting})
	}()
	<-waiting

	duty.LastCompanionDisconnected(context.Background())

	select {
	case err := <-admission:
		if err != nil {
			t.Errorf("the held intake ended with %v, want it admitted", err)
		}
	case <-time.After(admissionWait):
		t.Error("the held intake stays held after the last companion disconnects")
	}
}

func TestFetchingIsPausedOnlyWhileOnlyAbsentCompanionsFetchForAPairedDevice(t *testing.T) {
	duty := fetchingduty.New()
	duty.FetchingChanged(context.Background(), onlyCompanionsFetch)
	if duty.FetchingPaused() {
		t.Error("fetching is paused although no device is paired")
	}

	duty.FirstDevicePaired(context.Background())
	if !duty.FetchingPaused() {
		t.Error("fetching is not paused while only absent companions fetch")
	}

	duty.FirstCompanionConnected(context.Background())
	if duty.FetchingPaused() {
		t.Error("fetching is paused while a companion is connected")
	}

	duty.LastCompanionDisconnected(context.Background())
	duty.FetchingChanged(context.Background(), peerFillsIn)
	if duty.FetchingPaused() {
		t.Error("fetching is paused while the peer fills in")
	}
}
