package relevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
)

const saturationOfTheQueryPhraseHits = 1.0

func phraseScoreOf(item peeranswers.AnsweredItem) float64 {
	queryPhraseHits := float64(item.QueryPhraseHits)

	return queryPhraseHits / (queryPhraseHits + saturationOfTheQueryPhraseHits)
}
