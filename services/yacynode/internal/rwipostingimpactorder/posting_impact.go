package rwipostingimpactorder

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

const (
	titleAppearanceImpact = 10.0
	textHitsWeight        = 0.25
	textHitsSaturation    = 1.2
	impactScale           = 1000
)

type Impact int64

func ImpactOf(posting yacymodel.RWIPosting) Impact {
	impact := textHitsWeight * saturatedTextHitsOf(posting.Hits)
	if posting.Appearance.AppearsInTitle {
		impact += titleAppearanceImpact
	}

	return Impact(impact * impactScale)
}

func saturatedTextHitsOf(hits int) float64 {
	countedHits := float64(hits)

	return countedHits * (textHitsSaturation + 1) / (countedHits + textHitsSaturation)
}
