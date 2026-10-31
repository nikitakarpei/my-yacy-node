package rwipostingimpactorder_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

func TestEveryHitCountAPostingHoldsHasAnImpactOfItsOwn(t *testing.T) {
	t.Parallel()

	previousImpact := rwipostingimpactorder.ImpactOf(postingOf("w1", "u1", 1))
	for hits := 2; hits <= math.MaxUint8; hits++ {
		impact := rwipostingimpactorder.ImpactOf(postingOf("w1", "u1", uint8(hits)))
		if impact <= previousImpact {
			t.Fatalf(
				"impact of %d hits = %d, want more than the impact of %d hits (%d)",
				hits, impact, hits-1, previousImpact,
			)
		}
		previousImpact = impact
	}
}

func TestAPostingOfATitleOutweighsTheMostHitsAPostingHolds(t *testing.T) {
	t.Parallel()

	titlePosting := postingOf("w1", "u1", 1)
	titlePosting.Appearance.AppearsInTitle = true

	textPosting := postingOf("w1", "u2", math.MaxUint8)
	if rwipostingimpactorder.ImpactOf(titlePosting) <=
		rwipostingimpactorder.ImpactOf(textPosting) {
		t.Fatalf(
			"title impact %d, text impact %d, want the title posting to outweigh the text posting",
			rwipostingimpactorder.ImpactOf(titlePosting),
			rwipostingimpactorder.ImpactOf(textPosting),
		)
	}
}

func TestAPostingWithoutHitsHasNoImpact(t *testing.T) {
	t.Parallel()

	if impact := rwipostingimpactorder.ImpactOf(postingOf("w1", "u1", 0)); impact != 0 {
		t.Fatalf("impact = %d, want 0", impact)
	}
}
