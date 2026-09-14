package rwipostingimpactorder

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	titleAppearanceImpact = 10.0
	textHitsWeight        = 0.1
	impactScale           = 1_000_000
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
