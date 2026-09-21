package documentrelevance

type ScoreWeights struct {
	WeightOfTheTitleScore          float64
	WeightOfTheTextScore           float64
	WeightOfThePhraseScore         float64
	WeightOfTheNamedSiteEntryScore float64
	WeightOfTheLinkSparsityPenalty float64
}

func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		WeightOfTheTitleScore:          10.0,
		WeightOfTheTextScore:           0.25,
		WeightOfThePhraseScore:         3.0,
		WeightOfTheNamedSiteEntryScore: 5.0,
		WeightOfTheLinkSparsityPenalty: 1.5,
	}
}
