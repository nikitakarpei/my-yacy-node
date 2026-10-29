package peerroster_test

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
)

var signingStart = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

type droppingObserver struct {
	countingObserver

	mu          sync.Mutex
	dropReasons []peerroster.SeedDropReason
}

func (o *droppingObserver) ObserveDroppedSeed(reason peerroster.SeedDropReason) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.dropReasons = append(o.dropReasons, reason)
}

func (o *droppingObserver) observedDropReasons() []peerroster.SeedDropReason {
	o.mu.Lock()
	defer o.mu.Unlock()

	return slices.Clone(o.dropReasons)
}

func TestRosterBindsTheFirstVerifiedSeedAndMovesThePeerOnItsNewerSeeds(t *testing.T) {
	ctx := context.Background()
	observer := &droppingObserver{}
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4, observer: observer})

	first := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, first, httpBaseURLOf(first))
	moved := signedSeedAt(t, "203.0.113.2", signingStart.Add(time.Minute))
	roster.ConfirmReachable(ctx, moved, httpBaseURLOf(moved))

	assertHostOf(t, roster, moved.Hash, "203.0.113.2")
	if !roster.IsReachable(ctx, moved.Hash) {
		t.Error("IsReachable = false, want the bound peer reachable")
	}
	if dropReasons := observer.observedDropReasons(); len(dropReasons) != 0 {
		t.Errorf("drop reasons = %v, want none", dropReasons)
	}
}

func TestRosterKeepsAnUnverifiedPeerAsLegacy(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	first := unsignedSeedAt(t, "203.0.113.1")
	roster.ConfirmReachable(ctx, first, httpBaseURLOf(first))
	moved := unsignedSeedAt(t, "203.0.113.2")
	roster.ConfirmReachable(ctx, moved, httpBaseURLOf(moved))

	assertHostOf(t, roster, moved.Hash, "203.0.113.2")
}

func TestRosterDropsSeedsThatBreakTheBoundIdentity(t *testing.T) {
	for name, test := range map[string]struct {
		forgedSeed func(t *testing.T) yacymodel.Seed
		wantReason peerroster.SeedDropReason
	}{
		"unsigned": {
			forgedSeed: func(t *testing.T) yacymodel.Seed {
				return unsignedSeedAt(t, "198.51.100.9")
			},
			wantReason: peerroster.DropOfSeedWithoutBoundIdentity,
		},
		"signature over other addresses": {
			forgedSeed: func(t *testing.T) yacymodel.Seed {
				seed := signedSeedAt(t, "203.0.113.1", signingStart.Add(time.Minute))
				seed.PrimaryHost = yacymodel.Some(hostOf(t, "198.51.100.9"))

				return seed
			},
			wantReason: peerroster.DropOfSeedWithoutBoundIdentity,
		},
		"older signature": {
			forgedSeed: func(t *testing.T) yacymodel.Seed {
				return signedSeedAt(t, "198.51.100.9", signingStart.Add(-time.Minute))
			},
			wantReason: peerroster.DropOfOlderAddressSignature,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			observer := &droppingObserver{}
			roster := openRoster(
				t,
				rosterFixture{reservoirCap: 8, reachableCap: 4, observer: observer},
			)
			bound := signedSeedAt(t, "203.0.113.1", signingStart)
			roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))
			roster.ConfirmUnreachable(ctx, bound.Hash)

			forged := test.forgedSeed(t)
			roster.ConfirmReachable(ctx, forged, httpBaseURLOf(forged))

			assertHostOf(t, roster, bound.Hash, "203.0.113.1")
			if roster.IsReachable(ctx, bound.Hash) {
				t.Error("IsReachable = true, want the dropped seed not confirmed")
			}
			if dropReasons := observer.observedDropReasons(); !slices.Equal(
				dropReasons, []peerroster.SeedDropReason{test.wantReason},
			) {
				t.Errorf("drop reasons = %v, want [%s]", dropReasons, test.wantReason)
			}
		})
	}
}

func TestRosterBindsAPeerDiscoveredFromAVerifiedSeed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	roster.Discover(ctx, signedSeedAt(t, "203.0.113.1", signingStart))
	forged := unsignedSeedAt(t, "198.51.100.9")
	roster.ConfirmReachable(ctx, forged, httpBaseURLOf(forged))

	assertHostOf(t, roster, forged.Hash, "203.0.113.1")
}

func TestDiscoverBindsAKnownLegacyPeerFromItsSignedSeed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	roster.Discover(ctx, unsignedSeedAt(t, "203.0.113.1"))
	roster.Discover(ctx, signedSeedAt(t, "203.0.113.2", signingStart))
	forged := unsignedSeedAt(t, "198.51.100.9")
	roster.ConfirmReachable(ctx, forged, httpBaseURLOf(forged))

	assertHostOf(t, roster, forged.Hash, "203.0.113.2")
}

func TestDiscoverMovesABoundPeerOnItsNewerSignedSeed(t *testing.T) {
	for name, test := range map[string]struct {
		gossipedHost   string
		wantAnswerKept bool
	}{
		"to a new host":    {gossipedHost: "203.0.113.2", wantAnswerKept: false},
		"on the same host": {gossipedHost: "203.0.113.1", wantAnswerKept: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
			bound := signedSeedAt(t, "203.0.113.1", signingStart)
			roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))

			roster.Discover(ctx, signedSeedAt(t, test.gossipedHost, signingStart.Add(time.Minute)))

			assertHostOf(t, roster, bound.Hash, test.gossipedHost)
			if _, kept := roster.AnsweredBaseURLOf(ctx, bound.Hash); kept != test.wantAnswerKept {
				t.Errorf("answered base URL kept = %v, want %v", kept, test.wantAnswerKept)
			}
		})
	}
}

func TestDiscoverIgnoresGossipThatDoesNotSupersedeTheBinding(t *testing.T) {
	for name, gossiped := range map[string]func(t *testing.T) yacymodel.Seed{
		"unsigned": func(t *testing.T) yacymodel.Seed {
			return unsignedSeedAt(t, "198.51.100.9")
		},
		"older signature": func(t *testing.T) yacymodel.Seed {
			return signedSeedAt(t, "198.51.100.9", signingStart.Add(-time.Minute))
		},
		"signature of the same second": func(t *testing.T) yacymodel.Seed {
			return signedSeedAt(t, "198.51.100.9", signingStart)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			observer := &droppingObserver{}
			roster := openRoster(
				t,
				rosterFixture{reservoirCap: 8, reachableCap: 4, observer: observer},
			)
			bound := signedSeedAt(t, "203.0.113.1", signingStart)
			roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))

			roster.Discover(ctx, gossiped(t))

			assertHostOf(t, roster, bound.Hash, "203.0.113.1")
			if _, kept := roster.AnsweredBaseURLOf(ctx, bound.Hash); !kept {
				t.Error("answered base URL dropped, want it kept")
			}
			if dropReasons := observer.observedDropReasons(); len(dropReasons) != 0 {
				t.Errorf("drop reasons = %v, want none for gossip", dropReasons)
			}
		})
	}
}

func TestRosterKeepsTheBoundIdentityAcrossARestart(t *testing.T) {
	ctx := context.Background()
	storagePath := filepath.Join(t.TempDir(), "peer-roster")
	storedVault := openRosterVault(t, storagePath)
	storedRoster := openRoster(
		t,
		rosterFixture{storage: storedVault, reservoirCap: 8, reachableCap: 4},
	)
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	storedRoster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))
	if err := storedVault.Close(); err != nil {
		t.Fatalf("Close stored roster: %v", err)
	}

	restoredRoster := restoredRosterAt(t, storagePath)
	forged := unsignedSeedAt(t, "198.51.100.9")
	restoredRoster.ConfirmReachable(ctx, forged, httpBaseURLOf(forged))

	assertHostOf(t, restoredRoster, bound.Hash, "203.0.113.1")
}

func restoredRosterAt(t *testing.T, storagePath string) peerroster.Roster {
	t.Helper()
	restoredVault := openRosterVault(t, storagePath)
	t.Cleanup(func() {
		if err := restoredVault.Close(); err != nil {
			t.Errorf("Close restored roster: %v", err)
		}
	})

	return openRoster(t, rosterFixture{storage: restoredVault, reservoirCap: 8, reachableCap: 4})
}

func signedSeedAt(t *testing.T, ip string, issuedAt time.Time) yacymodel.Seed {
	t.Helper()

	return peerkeytest.Key(0).SignedSeedFrom(unsignedSeedAt(t, ip), rosterNetworkName, issuedAt)
}

func unsignedSeedAt(t *testing.T, ip string) yacymodel.Seed {
	t.Helper()
	seed := seniorSeed(t, "keyed", ip, 8090)
	seed.Hash = peerkeytest.Key(0).Identity().Hash()

	return seed
}

func hostOf(t *testing.T, text string) yacymodel.Host {
	t.Helper()
	host, err := yacymodel.ParseHost(text)
	if err != nil {
		t.Fatalf("parse host %q: %v", text, err)
	}

	return host
}

func assertHostOf(t *testing.T, roster peerroster.Roster, peer yacymodel.Hash, want string) {
	t.Helper()
	addresses, known := roster.AddressesOf(context.Background(), peer)
	if !known || len(addresses.Hosts) == 0 || addresses.Hosts[0].String() != want {
		t.Errorf("AddressesOf = %v (known %v), want host %s", addresses, known, want)
	}
}
