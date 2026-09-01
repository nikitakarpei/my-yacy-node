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

type rosterFixture struct {
	storage          *vault.Vault
	reservoirCap     int
	reachableCap     int
	announceInterval time.Duration
	clockStart       time.Time
	observer         peerroster.RosterObserver
}

func openRoster(t *testing.T, fixture rosterFixture) peerroster.Roster {
	t.Helper()

	roster, err := peerroster.Open(
		fixture.openStorage(t),
		fixture.clock(),
		fixture.reservoirCap,
		fixture.reachableCap,
		fixture.announcementInterval(),
		selfHash(),
		fixture.rosterObserver(),
	)
	if err != nil {
		t.Fatalf("peerroster.Open: %v", err)
	}

	return roster
}

func (f rosterFixture) openStorage(t *testing.T) *vault.Vault {
	t.Helper()

	if f.storage != nil {
		return f.storage
	}

	storage, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(0)),
		nil,
	)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return storage
}

func (f rosterFixture) clock() func() time.Time {
	start := f.clockStart
	if start.IsZero() {
		start = time.Unix(defaultClockStart, 0)
	}
	clock := &tickingClock{now: start}

	return clock.Now
}

func (f rosterFixture) announcementInterval() time.Duration {
	if f.announceInterval == 0 {
		return defaultAnnounceInterval
	}

	return f.announceInterval
}

func (f rosterFixture) rosterObserver() peerroster.RosterObserver {
	if f.observer == nil {
		return peerroster.DiscardObserver
	}

	return f.observer
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
