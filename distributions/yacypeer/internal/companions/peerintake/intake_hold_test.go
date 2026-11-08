package peerintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/fetchsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/peerintake"
)

func admits(hold *peerintake.IntakeHold) bool {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return !errors.Is(hold.AwaitAdmission(ctx), context.Canceled)
}

func pairedHold() *peerintake.IntakeHold {
	hold := peerintake.NewIntakeHold()
	hold.FirstDevicePaired(context.Background())
	return hold
}

func TestTheIntakeHoldAdmitsWhileNoDeviceIsPaired(t *testing.T) {
	hold := peerintake.NewIntakeHold()

	hold.ChoiceMade(context.Background(), fetchsettings.Choice{CompanionsFetch: true})

	if !admits(hold) {
		t.Error("the intake is held back although no device is paired")
	}
}

func TestTheIntakeHoldReleasesAWaitingIntakeOnceTheLastDeviceIsUnpaired(t *testing.T) {
	hold := pairedHold()
	hold.ChoiceMade(context.Background(), fetchsettings.Choice{CompanionsFetch: true})
	admitted := make(chan error, 1)
	go func() { admitted <- hold.AwaitAdmission(context.Background()) }()

	hold.LastDeviceUnpaired(context.Background())

	if err := <-admitted; err != nil {
		t.Errorf("AwaitAdmission = %v, want admitted", err)
	}
}

func TestTheIntakeHoldAdmitsWhileCompanionsDoNotFetch(t *testing.T) {
	hold := pairedHold()
	hold.CompanionsConnected(context.Background())

	hold.ChoiceMade(context.Background(), fetchsettings.Choice{PeerFillsIn: true})

	if !admits(hold) {
		t.Error("the intake is held back while companions do not fetch")
	}
}

func TestTheIntakeHoldHoldsBackWhileOnlyCompanionsFetch(t *testing.T) {
	hold := pairedHold()

	hold.ChoiceMade(context.Background(), fetchsettings.Choice{CompanionsFetch: true})

	if admits(hold) {
		t.Error("the intake is admitted while only companions fetch and none is connected")
	}
}

func TestTheIntakeHoldAdmitsWhileThePeerFillsInForAbsentCompanions(t *testing.T) {
	hold := pairedHold()

	hold.ChoiceMade(
		context.Background(),
		fetchsettings.Choice{CompanionsFetch: true, PeerFillsIn: true},
	)

	if !admits(hold) {
		t.Error("the intake is held back although no companion is connected")
	}
}

func TestTheIntakeHoldHoldsBackOnceACompanionConnects(t *testing.T) {
	hold := pairedHold()
	hold.ChoiceMade(
		context.Background(),
		fetchsettings.Choice{CompanionsFetch: true, PeerFillsIn: true},
	)

	hold.CompanionsConnected(context.Background())

	if admits(hold) {
		t.Error("the intake is admitted while a companion is connected")
	}
}

func TestTheIntakeHoldReleasesAWaitingIntakeOnceTheCompanionsLeft(t *testing.T) {
	hold := pairedHold()
	hold.ChoiceMade(
		context.Background(),
		fetchsettings.Choice{CompanionsFetch: true, PeerFillsIn: true},
	)
	hold.CompanionsConnected(context.Background())
	admitted := make(chan error, 1)
	go func() { admitted <- hold.AwaitAdmission(context.Background()) }()

	hold.CompanionsLeft(context.Background())

	if err := <-admitted; err != nil {
		t.Errorf("AwaitAdmission = %v, want admitted", err)
	}
}
