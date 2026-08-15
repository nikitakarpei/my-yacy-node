package rwiescrow_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestCapacityIsUnlimitedWithoutAQuota(t *testing.T) {
	withoutQuota := openCappedHarness(t, 0, 0.01)
	if got := withoutQuota.escrow.Capacity(); got != 0 {
		t.Fatalf("capacity = %d without a quota, want 0 for unlimited", got)
	}

	withoutFraction := openCappedHarness(t, 1<<20, 0)
	if got := withoutFraction.escrow.Capacity(); got != 0 {
		t.Fatalf("capacity = %d without a fraction, want 0 for unlimited", got)
	}

	withQuota := openCappedHarness(t, 1<<20, 0.5)
	if got := withQuota.escrow.Capacity(); got <= 0 {
		t.Fatalf("capacity = %d with a quota and a fraction, want a positive limit", got)
	}
}

func TestEscrowedPostingFitsItsByteBudget(t *testing.T) {
	const quotaBytes = 2560
	h := openCappedHarness(t, quotaBytes, 1.0)
	capacity := h.escrow.Capacity()
	if capacity <= 0 {
		t.Fatalf("capacity = %d, want a positive limit to hold postings against", capacity)
	}

	postings := make([]yacymodel.RWIPosting, capacity)
	for i := range postings {
		postings[i] = posting(fmt.Sprintf("w%d", i), fmt.Sprintf("u%d", i))
	}
	h.hold(t, postings...)

	used, err := h.vault.UsedBytes(context.Background())
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	if used > quotaBytes {
		t.Fatalf(
			"used = %d bytes holding postings at capacity, want at most the quota %d",
			used,
			quotaBytes,
		)
	}
}
