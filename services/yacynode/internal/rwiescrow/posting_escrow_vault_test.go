package rwiescrow

import (
	"bytes"
	"testing"
	"time"
)

func TestEscrowedPostingHoldTimeSurvivesTheValueRoundTrip(t *testing.T) {
	entry := posting("w1", "u1")
	identity := postingIdentity{Word: entry.WordHash, URL: entry.URLHash}

	for name, heldAt := range map[string]time.Time{
		"zero time":      {},
		"recent instant": time.Unix(1700000000, 123456789).UTC(),
		"far future":     time.Date(2500, time.January, 1, 0, 0, 0, 0, time.UTC),
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := escrowedPostingValueCodec{}.Encode(
				escrowedPosting{HeldAt: heldAt, Posting: entry},
			)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			escrowed, err := escrowedPostingValueCodec{}.Decode(raw)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			if !escrowed.HeldAt.Equal(heldAt) {
				t.Fatalf("hold time = %v, want %v", escrowed.HeldAt, heldAt)
			}
			decodedHold := postingHold{HeldAt: escrowed.HeldAt, Posting: identity}
			storedHold := postingHold{HeldAt: heldAt, Posting: identity}
			if !bytes.Equal(
				postingHoldKeyCodec{}.Encode(decodedHold).Bytes(),
				postingHoldKeyCodec{}.Encode(storedHold).Bytes(),
			) {
				t.Fatal(
					"the decoded hold time addresses a different hold row than the stored one",
				)
			}
		})
	}
}
