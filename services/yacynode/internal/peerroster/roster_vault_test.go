package peerroster

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func contactedSeed(t *testing.T) yacymodel.Seed {
	t.Helper()

	hash, err := yacymodel.ParseHash("AAAAAAAAAAAA")
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}
	name, err := yacymodel.ParsePeerName(hash.String())
	if err != nil {
		t.Fatalf("parse peer name: %v", err)
	}

	return yacymodel.Seed{Hash: hash, Name: name, PeerType: yacymodel.PeerSenior}
}

func TestRosterEntryRoundTripsContactTimesOutsideTheNanosecondEpochRange(t *testing.T) {
	for _, instant := range []time.Time{
		{},
		time.Date(1500, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 9, 12, 0, 0, 1, time.UTC),
		time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC),
	} {
		encoded, err := rosterEntryValueCodec{}.Encode(rosterEntry{
			seed:          contactedSeed(t),
			lastReachable: instant,
			lastContacted: instant.Add(time.Second),
		})
		if err != nil {
			t.Fatalf("Encode(%s) failed: %v", instant, err)
		}

		decoded, err := rosterEntryValueCodec{}.Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(%s) failed: %v", instant, err)
		}
		if !decoded.lastReachable.Equal(instant) {
			t.Fatalf("lastReachable = %s, want %s", decoded.lastReachable, instant)
		}
		if !decoded.lastContacted.Equal(instant.Add(time.Second)) {
			t.Fatalf("lastContacted = %s, want %s", decoded.lastContacted, instant.Add(time.Second))
		}
	}
}
