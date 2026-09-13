package rwiimpactorder

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

const (
	titleAppearanceRelevance = 10.0
	textHitsWeight           = 0.25
	textHitsSaturation       = 1.2
	impactPerRelevancePoint  = 1000
)

type Impact int64

func ImpactOf(posting yacymodel.RWIPosting) Impact {
	relevance := textHitsWeight * saturatedTextHitsOf(posting.Hits)
	if posting.Appearance.AppearsInTitle {
		relevance += titleAppearanceRelevance
	}

	return Impact(relevance * impactPerRelevancePoint)
}

func saturatedTextHitsOf(hits int) float64 {
	countedHits := float64(hits)

	return countedHits * (textHitsSaturation + 1) / (countedHits + textHitsSaturation)
}
