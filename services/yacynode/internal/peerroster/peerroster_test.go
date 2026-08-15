package peerroster_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
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

func openRoster(
	t *testing.T,
	reservoirCap, reachableCap int,
	announceInterval time.Duration,
) peerroster.Roster {
	t.Helper()

	clockStart := time.Unix(1_000, 0)

	return openRosterClockedFrom(t, clockStart, reservoirCap, reachableCap, announceInterval)
}

func openRosterClockedFrom(
	t *testing.T,
	clockStart time.Time,
	reservoirCap, reachableCap int,
	announceInterval time.Duration,
) peerroster.Roster {
	t.Helper()

	v, err := memory.Open(0, nil)
	if err != nil {
		t.Fatalf("memory.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	clock := &tickingClock{now: clockStart}
	roster, err := peerroster.Open(
		v, clock.Now, reservoirCap, reachableCap, announceInterval, selfHash(),
		peerroster.DiscardObserver,
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
