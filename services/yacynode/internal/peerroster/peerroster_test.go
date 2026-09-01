package peerroster_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
)

const hashFiller = "AAAAAAAAAAAA"

func hashFor(base string) yacymodel.Hash {
	padded := base + hashFiller
	hash, err := yacymodel.ParseHash(padded[:yacymodel.HashLength])
	if err != nil {
		panic(err)
	}

	return hash
}

func selfHash() yacymodel.Hash {
	return hashFor("self")
}

func seniorSeed(t testing.TB, hash, ip string, port int) yacymodel.Seed {
	t.Helper()

	name, err := yacymodel.ParsePeerName(hashFor(hash).String())
	if err != nil {
		t.Fatalf("parse peer name: %v", err)
	}
	seed := yacymodel.Seed{
		Hash:     hashFor(hash),
		Name:     name,
		PeerType: yacymodel.PeerSenior,
	}
	if ip != "" {
		host, err := yacymodel.ParseHost(ip)
		if err != nil {
			t.Fatalf("parse host: %v", err)
		}
		seed.PrimaryAddress = yacymodel.Some(host)
	}
	if port != 0 {
		seed.Port = yacymodel.Some(yacymodel.Port(port))
	}

	return seed
}

type tickingClock struct {
	now time.Time
}

func (c *tickingClock) Now() time.Time {
	c.now = c.now.Add(time.Second)

	return c.now
}

const defaultAnnounceInterval = 10 * time.Minute

const defaultClockStart = 1_000

func openRoster(
	t *testing.T,
	reservoirCap, reachableCap int,
	announceInterval time.Duration,
) peerroster.Roster {
	t.Helper()

	return openRosterClockedFrom(
		t, time.Unix(defaultClockStart, 0), reservoirCap, reachableCap, announceInterval,
	)
}

func openRosterClockedFrom(
	t *testing.T,
	clockStart time.Time,
	reservoirCap, reachableCap int,
	announceInterval time.Duration,
) peerroster.Roster {
	t.Helper()

	return rosterFixture{
		clockStart:       clockStart,
		reservoirCap:     reservoirCap,
		reachableCap:     reachableCap,
		announceInterval: announceInterval,
		observer:         peerroster.DiscardObserver,
	}.open(t)
}

func openRosterObservedBy(
	t *testing.T,
	observer peerroster.RosterObserver,
	reservoirCap, reachableCap int,
	announceInterval time.Duration,
) peerroster.Roster {
	t.Helper()

	return rosterFixture{
		clockStart:       time.Unix(defaultClockStart, 0),
		reservoirCap:     reservoirCap,
		reachableCap:     reachableCap,
		announceInterval: announceInterval,
		observer:         observer,
	}.open(t)
}

type rosterFixture struct {
	clockStart       time.Time
	reservoirCap     int
	reachableCap     int
	announceInterval time.Duration
	observer         peerroster.RosterObserver
}

func (f rosterFixture) open(t *testing.T) peerroster.Roster {
	t.Helper()

	v, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(0)),
		nil,
	)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	clock := &tickingClock{now: f.clockStart}
	roster, err := peerroster.Open(
		v, clock.Now, f.reservoirCap, f.reachableCap, f.announceInterval, selfHash(),
		f.observer,
	)
	if err != nil {
		t.Fatalf("peerroster.Open: %v", err)
	}

	return roster
}

func hashes(seeds []yacymodel.Seed) map[yacymodel.Hash]struct{} {
	out := make(map[yacymodel.Hash]struct{}, len(seeds))
	for _, seed := range seeds {
		out[seed.Hash] = struct{}{}
	}

	return out
}

func hashSet(peerHashes []yacymodel.Hash) map[yacymodel.Hash]struct{} {
	set := make(map[yacymodel.Hash]struct{}, len(peerHashes))
	for _, peerHash := range peerHashes {
		set[peerHash] = struct{}{}
	}

	return set
}
