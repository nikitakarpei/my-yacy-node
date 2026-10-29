package peerroster_test

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

var signingStart = time.Unix(defaultClockStart, 0).UTC()

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
		"older signature over other addresses": {
			forgedSeed: func(t *testing.T) yacymodel.Seed {
				return signedSeedAt(t, "198.51.100.9", signingStart.Add(-time.Minute))
			},
			wantReason: peerroster.DropOfOlderAddressSignature,
		},
		"signature from the future": {
			forgedSeed: func(t *testing.T) yacymodel.Seed {
				return signedSeedAt(t, "198.51.100.9", signingStart.Add(time.Hour))
			},
			wantReason: peerroster.DropOfFutureAddressSignature,
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

func TestRosterDropsAnOlderCopyOfTheBoundSeedOnAnyAddresses(t *testing.T) {
	ctx := context.Background()
	observer := &droppingObserver{}
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4, observer: observer})
	bound := signedSeedAt(t, "203.0.113.1", signingStart.Add(time.Second))
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))
	roster.ConfirmUnreachable(ctx, bound.Hash)

	sameAddresses := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, sameAddresses, httpBaseURLOf(sameAddresses))
	otherAddresses := signedSeedAt(t, "203.0.113.2", signingStart)
	roster.ConfirmReachable(ctx, otherAddresses, httpBaseURLOf(otherAddresses))

	assertHostOf(t, roster, bound.Hash, "203.0.113.1")
	if roster.IsReachable(ctx, bound.Hash) {
		t.Error("IsReachable = true, want no older copy confirmed")
	}
	want := []peerroster.SeedDropReason{
		peerroster.DropOfOlderAddressSignature,
		peerroster.DropOfOlderAddressSignature,
	}
	if dropReasons := observer.observedDropReasons(); !slices.Equal(dropReasons, want) {
		t.Errorf("drop reasons = %v, want %v", dropReasons, want)
	}
}

func TestRosterDropsAReplayOfTheBoundSeedOnceGossipBroughtANewerOne(t *testing.T) {
	ctx := context.Background()
	observer := &droppingObserver{}
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4, observer: observer})
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))

	roster.Discover(
		ctx,
		signedSeedWithSecurePortAt(t, "203.0.113.1", signingStart.Add(time.Minute)),
	)
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))

	record, _ := roster.RecordOf(ctx, bound.Hash)
	if !record.Addresses.SecurePort.Present() {
		t.Errorf("addresses = %v, want the secure port of the newer seed", record.Addresses)
	}
	if dropReasons := observer.observedDropReasons(); !slices.Equal(
		dropReasons, []peerroster.SeedDropReason{peerroster.DropOfOlderAddressSignature},
	) {
		t.Errorf("drop reasons = %v, want the replayed seed dropped as older", dropReasons)
	}
}

func TestDiscoverIgnoresASeedSignedInTheFuture(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	future := signedSeedAt(t, "203.0.113.1", signingStart.Add(time.Hour))

	roster.Discover(ctx, future)

	if addresses, known := addressesOf(ctx, roster, future.Hash); known {
		t.Errorf("addresses = %v, want the peer unknown", addresses)
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

func TestDiscoverMovesAnUnreachableBoundPeerOnItsNewerSignedSeed(t *testing.T) {
	for name, test := range map[string]struct {
		gossipedHost   string
		wantAnswerKept bool
	}{
		"to a new host":    {gossipedHost: "203.0.113.2", wantAnswerKept: false},
		"on the same host": {gossipedHost: "203.0.113.1", wantAnswerKept: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			storagePath := filepath.Join(t.TempDir(), "peer-roster")
			bound := signedSeedAt(t, "203.0.113.1", signingStart)
			confirmedAndClosedAt(t, storagePath, bound)
			roster := restoredRosterAt(t, storagePath)

			roster.Discover(ctx, signedSeedAt(t, test.gossipedHost, signingStart.Add(time.Minute)))

			assertHostOf(t, roster, bound.Hash, test.gossipedHost)
			if _, kept := answeredBaseURLOf(ctx, roster, bound.Hash); kept != test.wantAnswerKept {
				t.Errorf("answered base URL kept = %v, want %v", kept, test.wantAnswerKept)
			}
		})
	}
}

func TestDiscoverMovesAReachablePeerAndLeavesItUnconfirmedAtItsNewAddresses(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))

	roster.Discover(ctx, signedSeedAt(t, "203.0.113.2", signingStart.Add(time.Minute)))

	assertHostOf(t, roster, bound.Hash, "203.0.113.2")
	if reachable := roster.ReachablePeers(ctx); len(reachable) != 0 {
		t.Errorf("ReachablePeers = %v, want the moved peer no longer confirmed", reachable)
	}
}

func TestDiscoverKeepsAReachablePeerAtItsAddressesAndTakesItsNewerSeed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))
	newer := signedSeedAt(t, "203.0.113.1", signingStart.Add(time.Minute))

	roster.Discover(ctx, newer)

	if reachable := roster.ReachablePeers(
		ctx,
	); !reflect.DeepEqual(
		reachable,
		[]yacymodel.Seed{newer},
	) {
		t.Errorf("ReachablePeers = %v, want the newer seed", reachable)
	}
}

func TestRosterKeepsAPeerReachableWhenItsConfirmationDoesNotMoveIt(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))

	roster.Discover(ctx, bound)

	if reachable := roster.ReachablePeers(
		ctx,
	); !reflect.DeepEqual(
		reachable,
		[]yacymodel.Seed{bound},
	) {
		t.Errorf("ReachablePeers = %v, want the confirmed seed", reachable)
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
			roster.ConfirmUnreachable(ctx, bound.Hash)

			roster.Discover(ctx, gossiped(t))

			assertHostOf(t, roster, bound.Hash, "203.0.113.1")
			if dropReasons := observer.observedDropReasons(); len(dropReasons) != 0 {
				t.Errorf("drop reasons = %v, want none for gossip", dropReasons)
			}
		})
	}
}

func confirmedAndClosedAt(t *testing.T, storagePath string, seed yacymodel.Seed) {
	t.Helper()
	storedVault := openRosterVault(t, storagePath)
	storedRoster := openRoster(
		t,
		rosterFixture{storage: storedVault, reservoirCap: 8, reachableCap: 4},
	)
	storedRoster.ConfirmReachable(context.Background(), seed, httpBaseURLOf(seed))
	if err := storedVault.Close(); err != nil {
		t.Fatalf("Close stored roster: %v", err)
	}
}

func TestRosterKeepsTheBoundIdentityAcrossARestart(t *testing.T) {
	ctx := context.Background()
	storagePath := filepath.Join(t.TempDir(), "peer-roster")
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	confirmedAndClosedAt(t, storagePath, bound)

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

func TestRosterRecordsTheBoundIdentityOfAPeer(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	seed := signedSeedAt(t, "203.0.113.1", signingStart)

	roster.Discover(ctx, seed)

	record, _ := roster.RecordOf(ctx, seed.Hash)
	if record.Callee != peerwire.IdentifiedCallee(peerkeytest.Key(0).Identity()) {
		t.Errorf("Callee = %v, want the identity of the signed seed", record.Callee)
	}
}

func TestRosterKeepsOnlyAnAnsweredBaseURLThatTheBindingAllows(t *testing.T) {
	for name, test := range map[string]struct {
		answered       peerwire.BaseURL
		wantAnswerKept bool
	}{
		"plain HTTP to a peer bound to a secure port": {
			answered: peerwire.BaseURL{Scheme: peerwire.HTTP, Host: hostOf(t, "203.0.113.1"), Port: 8090},
		},
		"HTTPS at the secure port": {
			answered:       peerwire.BaseURL{Scheme: peerwire.HTTPS, Host: hostOf(t, "203.0.113.1"), Port: 8443},
			wantAnswerKept: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
			roster.Discover(ctx, unsignedSeedAt(t, "203.0.113.1"))

			roster.ConfirmReachable(
				ctx,
				signedSeedWithSecurePortAt(t, "203.0.113.1", signingStart),
				test.answered,
			)

			if _, kept := answeredBaseURLOf(
				ctx,
				roster,
				peerkeytest.Key(0).Hash(),
			); kept != test.wantAnswerKept {
				t.Errorf("answered base URL kept = %v, want %v", kept, test.wantAnswerKept)
			}
		})
	}
}

func TestRosterKeepsTheSeedOfAPeerWhenAConfirmationRepeatsItsSignature(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))

	roster.ConfirmReachable(ctx, rewrittenSeedOf(bound), httpBaseURLOf(bound))

	if reachable := roster.ReachablePeers(
		ctx,
	); !reflect.DeepEqual(
		reachable,
		[]yacymodel.Seed{bound},
	) {
		t.Errorf("ReachablePeers = %v, want the seed the peer gave itself", reachable)
	}
}

func TestRosterRecordsAnUnreachablePeerAsAnsweredWhenAConfirmationRepeatsItsSignature(
	t *testing.T,
) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	bound := signedSeedAt(t, "203.0.113.1", signingStart)
	roster.ConfirmReachable(ctx, bound, httpBaseURLOf(bound))
	roster.ConfirmUnreachable(ctx, bound.Hash)

	roster.ConfirmReachable(ctx, rewrittenSeedOf(bound), httpBaseURLOf(bound))

	if reachable := roster.ReachablePeers(ctx); len(reachable) != 0 {
		t.Errorf("ReachablePeers = %v, want no copy of the signed seed", reachable)
	}
	if !roster.IsRecentlyReachable(ctx, bound.Hash) {
		t.Error("IsRecentlyReachable = false, want the answered back-ping recorded")
	}
}

func TestRosterReplacesTheSeedOfAPeerOnAConfirmationWithoutItsSignature(t *testing.T) {
	for name, seedsOf := range map[string]func(t *testing.T) (earlier, later yacymodel.Seed){
		"newer signature": func(t *testing.T) (yacymodel.Seed, yacymodel.Seed) {
			return signedSeedAt(t, "203.0.113.1", signingStart),
				rewrittenSeedOf(signedSeedAt(t, "203.0.113.1", signingStart.Add(time.Minute)))
		},
		"legacy peer": func(t *testing.T) (yacymodel.Seed, yacymodel.Seed) {
			legacy := seniorSeed(t, "legacy", "203.0.113.1", 8090)

			return legacy, rewrittenSeedOf(legacy)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
			earlier, later := seedsOf(t)
			roster.ConfirmReachable(ctx, earlier, httpBaseURLOf(earlier))

			roster.ConfirmReachable(ctx, later, httpBaseURLOf(later))

			if reachable := roster.ReachablePeers(
				ctx,
			); !reflect.DeepEqual(
				reachable,
				[]yacymodel.Seed{later},
			) {
				t.Errorf("ReachablePeers = %v, want the later seed", reachable)
			}
		})
	}
}

func rewrittenSeedOf(seed yacymodel.Seed) yacymodel.Seed {
	seed.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true})

	return seed
}

func signedSeedWithSecurePortAt(t *testing.T, ip string, issuedAt time.Time) yacymodel.Seed {
	t.Helper()
	seed := unsignedSeedAt(t, ip)
	seed.SecurePort = yacymodel.Some(yacymodel.Port(8443))
	seed.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{SSLAvailable: true})

	return peerkeytest.Key(0).SignedSeedFrom(seed, rosterNetworkName, issuedAt)
}

func signedSeedAt(t *testing.T, ip string, issuedAt time.Time) yacymodel.Seed {
	t.Helper()

	return peerkeytest.Key(0).SignedSeedFrom(unsignedSeedAt(t, ip), rosterNetworkName, issuedAt)
}

func unsignedSeedAt(t *testing.T, ip string) yacymodel.Seed {
	t.Helper()
	seed := seniorSeed(t, "keyed", ip, 8090)
	seed.Hash = peerkeytest.Key(0).Hash()

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
	addresses, known := addressesOf(context.Background(), roster, peer)
	if !known || len(addresses.Hosts) == 0 || addresses.Hosts[0].String() != want {
		t.Errorf("addresses = %v (known %v), want host %s", addresses, known, want)
	}
}
