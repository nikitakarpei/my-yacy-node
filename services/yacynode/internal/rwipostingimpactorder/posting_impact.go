package rwipostingimpactorder

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// The impact is the stored key of the order: a change here needs the order
// rebuilt, or old and new keys interleave.
const (
	// The unit the other two are set against; only its ratio to
	// textHitsWeight carries meaning.
	titleAppearanceImpact = 10.0
	// The largest power of ten that keeps the hits term below
	// titleAppearanceImpact for every hit count an int64 can hold: the
	// largest count gives 0.1 * ln(2^63) = 4.4, so a word in the title
	// always outranks a word found only in the text.
	textHitsWeight = 0.1
	// The key is an integer, so the fraction of the curve is lost. The wire
	// carries at most 255 hits, and the curve rises least between 254 and 255,
	// by 0.00039; the scale must turn that into at least one key unit so no
	// two hit counts share a key. One million turns it into 391.
	impactScale = 1_000_000
)

type Impact int64

func ImpactOf(posting yacymodel.RWIPosting) Impact {
	impact := textHitsWeight * diminishedTextHitsOf(posting.Hits)
	if posting.Appearance.AppearsInTitle {
		impact += titleAppearanceImpact
	}

	return Impact(impact * impactScale)
}

func diminishedTextHitsOf(hits int) float64 {
	return math.Log1p(float64(max(hits, 0)))
}
