package documentrelevance

type RelevanceWeights struct {
	WeightOfTitleScore          float64
	WeightOfTextScore           float64
	WeightOfPhraseScore         float64
	WeightOfNamedSiteEntryScore float64
	WeightOfLinkSparsityPenalty float64
}

func DefaultRelevanceWeights() RelevanceWeights {
	return RelevanceWeights{
		WeightOfTitleScore:          10.0,
		WeightOfTextScore:           0.25,
		WeightOfPhraseScore:         3.0,
		WeightOfNamedSiteEntryScore: 5.0,
		WeightOfLinkSparsityPenalty: 1.5,
	}
}
