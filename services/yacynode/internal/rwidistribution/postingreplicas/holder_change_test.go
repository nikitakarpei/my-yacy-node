package postingreplicas_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
)

func TestHolderChangeAddsTheAcceptingPeers(t *testing.T) {
	held, accepting := yacymodel.WordHash("held"), yacymodel.WordHash("accepting")
	change := postingreplicas.HolderChange{AcceptingPeers: []yacymodel.Hash{accepting}}

	holders := change.HoldersFrom([]yacymodel.Hash{held})

	if !slices.Equal(holders, []yacymodel.Hash{held, accepting}) {
		t.Fatalf("holders = %v, want [held accepting]", holders)
	}
}

func TestHolderChangeRecordsAPeerAcceptingAgainOnce(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	change := postingreplicas.HolderChange{AcceptingPeers: []yacymodel.Hash{peer}}

	holders := change.HoldersFrom([]yacymodel.Hash{peer})

	if !slices.Equal(holders, []yacymodel.Hash{peer}) {
		t.Fatalf("holders = %v, want [peer]", holders)
	}
}

func TestHolderChangeDropsTheStaleHolders(t *testing.T) {
	alive, stale := yacymodel.WordHash("alive"), yacymodel.WordHash("stale")
	change := postingreplicas.HolderChange{StaleHolders: []yacymodel.Hash{stale}}
	recordedHolders := []yacymodel.Hash{alive, stale}

	holders := change.HoldersFrom(recordedHolders)

	if !slices.Equal(holders, []yacymodel.Hash{alive}) {
		t.Fatalf("holders = %v, want [alive]", holders)
	}
	if dropped := change.AmountOfStaleHoldersIn(recordedHolders); dropped != 1 {
		t.Fatalf("dropped = %d, want 1", dropped)
	}
}

func TestHolderChangeDropsNothingAStaleHolderNeverHeld(t *testing.T) {
	change := postingreplicas.HolderChange{
		StaleHolders: []yacymodel.Hash{yacymodel.WordHash("stale")},
	}

	if dropped := change.AmountOfStaleHoldersIn(nil); dropped != 0 {
		t.Fatalf("dropped = %d, want 0", dropped)
	}
}
