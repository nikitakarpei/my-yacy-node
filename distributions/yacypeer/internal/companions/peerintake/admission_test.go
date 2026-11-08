package peerintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/peerintake"
)

var (
	onlyCompanionsFetch = companionsettings.Fetching{CompanionsFetch: true}
	peerFillsIn         = companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}
)

func admits(admission *peerintake.Admission) bool {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return !errors.Is(admission.AwaitAdmission(ctx), context.Canceled)
}

func pairedAdmission() *peerintake.Admission {
	admission := peerintake.New()
	admission.FirstDevicePaired(context.Background())
	return admission
}

func TestTheAdmissionAdmitsWhileNoDeviceIsPaired(t *testing.T) {
	admission := peerintake.New()

	admission.FetchingChanged(context.Background(), onlyCompanionsFetch)

	if !admits(admission) {
		t.Error("the intake is held back although no device is paired")
	}
}

func TestTheAdmissionAdmitsOnceTheLastDeviceIsUnpaired(t *testing.T) {
	admission := pairedAdmission()
	admission.FetchingChanged(context.Background(), onlyCompanionsFetch)
	if admits(admission) {
		t.Fatal("the intake is admitted before the last device is unpaired")
	}

	admission.LastDeviceUnpaired(context.Background())

	if !admits(admission) {
		t.Error("the intake is held back after the last device is unpaired")
	}
}

func TestTheAdmissionAdmitsWhileCompanionsDoNotFetch(t *testing.T) {
	admission := pairedAdmission()
	admission.FirstCompanionConnected(context.Background())

	admission.FetchingChanged(context.Background(), companionsettings.Fetching{PeerFillsIn: true})

	if !admits(admission) {
		t.Error("the intake is held back while companions do not fetch")
	}
}

func TestTheAdmissionHoldsBackWhileOnlyCompanionsFetch(t *testing.T) {
	admission := pairedAdmission()

	admission.FetchingChanged(context.Background(), onlyCompanionsFetch)

	if admits(admission) {
		t.Error("the intake is admitted while only companions fetch and none is connected")
	}
}

func TestTheAdmissionAdmitsWhileThePeerFillsInForAbsentCompanions(t *testing.T) {
	admission := pairedAdmission()

	admission.FetchingChanged(context.Background(), peerFillsIn)

	if !admits(admission) {
		t.Error("the intake is held back although no companion is connected")
	}
}

func TestTheAdmissionHoldsBackOnceACompanionConnects(t *testing.T) {
	admission := pairedAdmission()
	admission.FetchingChanged(context.Background(), peerFillsIn)

	admission.FirstCompanionConnected(context.Background())

	if admits(admission) {
		t.Error("the intake is admitted while a companion is connected")
	}
}

func TestTheAdmissionAdmitsOnceTheLastCompanionDisconnects(t *testing.T) {
	admission := pairedAdmission()
	admission.FetchingChanged(context.Background(), peerFillsIn)
	admission.FirstCompanionConnected(context.Background())
	if admits(admission) {
		t.Fatal("the intake is admitted before the last companion disconnects")
	}

	admission.LastCompanionDisconnected(context.Background())

	if !admits(admission) {
		t.Error("the intake is held back after the last companion disconnects")
	}
}

func TestFetchingIsPausedOnlyWhileOnlyAbsentCompanionsFetchForAPairedDevice(t *testing.T) {
	admission := peerintake.New()
	admission.FetchingChanged(context.Background(), onlyCompanionsFetch)
	if admission.FetchingPaused() {
		t.Error("fetching is paused although no device is paired")
	}

	admission.FirstDevicePaired(context.Background())
	if !admission.FetchingPaused() {
		t.Error("fetching is not paused while only absent companions fetch")
	}

	admission.FirstCompanionConnected(context.Background())
	if admission.FetchingPaused() {
		t.Error("fetching is paused while a companion is connected")
	}

	admission.LastCompanionDisconnected(context.Background())
	admission.FetchingChanged(context.Background(), peerFillsIn)
	if admission.FetchingPaused() {
		t.Error("fetching is paused while the peer fills in")
	}
}
