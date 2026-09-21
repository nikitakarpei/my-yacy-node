package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	saturationOfQueryPhraseHits = 1.0
	phraseScoreOfUnreadDocument = 0.0
)

func phraseScoreOf(document queryanswers.FoundDocument) float64 {
	queryPhraseHits, queryPhrasesCounted := document.Facts.QueryPhraseHits.Get()
	if !queryPhrasesCounted {
		return phraseScoreOfUnreadDocument
	}

	return float64(queryPhraseHits) /
		(float64(queryPhraseHits) + saturationOfQueryPhraseHits)
}
