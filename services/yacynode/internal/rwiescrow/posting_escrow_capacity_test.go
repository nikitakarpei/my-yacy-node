package rwiescrow

import (
	"testing"
	"time"
)

func TestEscrowedPostingFitsItsByteBudget(t *testing.T) {
	entry := posting("w1", "u1")
	identity := postingIdentity{Word: entry.WordHash, URL: entry.URLHash}
	heldAt := time.Unix(1700000000, 0)

	value, err := escrowedPostingValueCodec{}.Encode(
		escrowedPosting{HeldAt: heldAt, Posting: entry},
	)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	hold := postingHold{HeldAt: heldAt, Posting: identity}
	footprint := len(escrowedPostingKeyCodec{}.Encode(identity).Bytes()) +
		len(value) +
		len(postingHoldKeyCodec{}.Encode(hold).Bytes())
	if footprint > escrowedPostingBytes {
		t.Fatalf(
			"escrowed posting footprint = %d bytes, want at most %d",
			footprint,
			escrowedPostingBytes,
		)
	}
}

func TestCapacityIsUnlimitedWithoutAQuota(t *testing.T) {
	if got := capacityWithin(0, capacityFraction); got != 0 {
		t.Fatalf("capacity = %d without a quota, want 0 for unlimited", got)
	}
	if got := capacityWithin(quotaHolding(2), 0); got != 0 {
		t.Fatalf("capacity = %d without a fraction, want 0 for unlimited", got)
	}
}
