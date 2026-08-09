package rwiescrow

import (
	"bytes"
	"testing"
	"time"
)

func TestHeldPostingHoldTimeSurvivesTheValueRoundTrip(t *testing.T) {
	entry := posting("w1", "u1")
	identity := postingIdentity{Word: entry.WordHash, URL: entry.URLHash}

	for name, heldAt := range map[string]time.Time{
		"zero time":      {},
		"recent instant": time.Unix(1700000000, 123456789).UTC(),
		"far future":     time.Date(2500, time.January, 1, 0, 0, 0, 0, time.UTC),
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := heldPostingValueCodec{}.Encode(heldPosting{HeldAt: heldAt, Posting: entry})
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			held, err := heldPostingValueCodec{}.Decode(raw)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			if !held.HeldAt.Equal(heldAt) {
				t.Fatalf("hold time = %v, want %v", held.HeldAt, heldAt)
			}
			decodedHold := postingHold{HeldAt: held.HeldAt, Posting: identity}
			storedHold := postingHold{HeldAt: heldAt, Posting: identity}
			if !bytes.Equal(
				expiryKeyCodec{}.Encode(decodedHold).Bytes(),
				expiryKeyCodec{}.Encode(storedHold).Bytes(),
			) {
				t.Fatal(
					"the decoded hold time addresses a different expiry row than the stored one",
				)
			}
		})
	}
}
