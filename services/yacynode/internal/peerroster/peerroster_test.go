package peerroster_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
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

func indexAcceptingSeed(t testing.TB, hash, ip string) yacymodel.Seed {
	t.Helper()

	seed := seniorSeed(t, hash, ip, 8090)
	seed.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true})

	return seed
}

type tickingClock struct {
	now time.Time
}

func (c *tickingClock) Now() time.Time {
	c.now = c.now.Add(time.Second)

	return c.now
}

func openRoster(t *testing.T, reservoirCap, activeCap int) peerroster.Roster {
	t.Helper()

	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("memvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	clock := &tickingClock{now: time.Unix(1_000, 0)}
	roster, err := peerroster.Open(v, clock.Now, reservoirCap, activeCap, hashFor("self"))
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
