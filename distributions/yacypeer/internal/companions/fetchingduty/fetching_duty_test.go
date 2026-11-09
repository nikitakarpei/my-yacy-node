package fetchingduty_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/fetchingduty"
)

var (
	onlyCompanionsFetch = companionsettings.Fetching{CompanionsFetch: true}
	peerFillsIn         = companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}
)

func peerFetches(duty *fetchingduty.Duty) bool {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return !errors.Is(duty.AwaitAdmission(ctx), context.Canceled)
}

func pairedDuty() *fetchingduty.Duty {
	duty := fetchingduty.New()
	duty.FirstDevicePaired(context.Background())
	return duty
}

func TestThePeerFetchesWhileNoDeviceIsPaired(t *testing.T) {
	duty := fetchingduty.New()

	duty.FetchingChanged(context.Background(), onlyCompanionsFetch)

	if !peerFetches(duty) {
		t.Error("the intake is held back although no device is paired")
	}
}

func TestThePeerFetchesOnceTheLastDeviceIsUnpaired(t *testing.T) {
	duty := pairedDuty()
	duty.FetchingChanged(context.Background(), onlyCompanionsFetch)
	if peerFetches(duty) {
		t.Fatal("the intake is admitted before the last device is unpaired")
	}

	duty.LastDeviceUnpaired(context.Background())

	if !peerFetches(duty) {
		t.Error("the intake is held back after the last device is unpaired")
	}
}

func TestThePeerFetchesWhileCompanionsDoNotFetch(t *testing.T) {
	duty := pairedDuty()
	duty.FirstCompanionConnected(context.Background())

	duty.FetchingChanged(context.Background(), companionsettings.Fetching{PeerFillsIn: true})

	if !peerFetches(duty) {
		t.Error("the intake is held back while companions do not fetch")
	}
}

func TestThePeerHoldsBackWhileOnlyCompanionsFetch(t *testing.T) {
	duty := pairedDuty()

	duty.FetchingChanged(context.Background(), onlyCompanionsFetch)

	if peerFetches(duty) {
		t.Error("the intake is admitted while only companions fetch and none is connected")
	}
}

func TestThePeerFetchesWhileItFillsInForAbsentCompanions(t *testing.T) {
	duty := pairedDuty()

	duty.FetchingChanged(context.Background(), peerFillsIn)

	if !peerFetches(duty) {
		t.Error("the intake is held back although no companion is connected")
	}
}

func TestThePeerHoldsBackOnceACompanionConnects(t *testing.T) {
	duty := pairedDuty()
	duty.FetchingChanged(context.Background(), peerFillsIn)

	duty.FirstCompanionConnected(context.Background())

	if peerFetches(duty) {
		t.Error("the intake is admitted while a companion is connected")
	}
}

func TestThePeerFetchesOnceTheLastCompanionDisconnects(t *testing.T) {
	duty := pairedDuty()
	duty.FetchingChanged(context.Background(), peerFillsIn)
	duty.FirstCompanionConnected(context.Background())
	if peerFetches(duty) {
		t.Fatal("the intake is admitted before the last companion disconnects")
	}

	duty.LastCompanionDisconnected(context.Background())

	if !peerFetches(duty) {
		t.Error("the intake is held back after the last companion disconnects")
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
