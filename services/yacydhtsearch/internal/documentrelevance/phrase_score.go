package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	saturationOfTheQueryPhraseHits              = 1.0
	phraseScoreWhenNobodyCountedTheQueryPhrases = 0.0
)

func phraseScoreOf(facts queryanswers.DocumentFacts) float64 {
	hits, counted := facts.QueryPhraseHits.Get()
	if !counted {
		return phraseScoreWhenNobodyCountedTheQueryPhrases
	}
	queryPhraseHits := float64(hits)

	return queryPhraseHits / (queryPhraseHits + saturationOfTheQueryPhraseHits)
}
