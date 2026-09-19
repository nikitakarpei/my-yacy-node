package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const saturationOfTheQueryPhraseHits = 1.0

func phraseScoreOf(foundDocument queryanswers.FoundDocument) float64 {
	queryPhraseHits := float64(foundDocument.QueryPhraseHits)

	return queryPhraseHits / (queryPhraseHits + saturationOfTheQueryPhraseHits)
}
