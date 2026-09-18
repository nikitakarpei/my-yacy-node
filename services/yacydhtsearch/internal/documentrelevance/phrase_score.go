package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const saturationOfTheQueryPhraseHits = 1.0

func phraseScoreOf(item queryanswers.FoundDocument) float64 {
	queryPhraseHits := float64(item.QueryPhraseHits)

	return queryPhraseHits / (queryPhraseHits + saturationOfTheQueryPhraseHits)
}
