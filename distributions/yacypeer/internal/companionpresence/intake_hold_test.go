package companionpresence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companionpresence"
)

func TestTheIntakeHoldAdmitsWhileNoCompanionIsConnected(t *testing.T) {
	if err := companionpresence.NewIntakeHold().AwaitAdmission(context.Background()); err != nil {
		t.Errorf("AwaitAdmission = %v, want admitted", err)
	}
}

func TestTheIntakeHoldHoldsBackWhileACompanionIsConnected(t *testing.T) {
	hold := companionpresence.NewIntakeHold()
	hold.CompanionsConnected(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := hold.AwaitAdmission(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("AwaitAdmission = %v, want held back until cancelled", err)
	}
}

func TestTheIntakeHoldAdmitsAgainOnceTheCompanionsLeft(t *testing.T) {
	hold := companionpresence.NewIntakeHold()
	hold.CompanionsConnected(context.Background())
	admitted := make(chan error, 1)
	go func() { admitted <- hold.AwaitAdmission(context.Background()) }()

	hold.CompanionsLeft(context.Background())

	if err := <-admitted; err != nil {
		t.Errorf("AwaitAdmission = %v, want admitted", err)
	}
}
