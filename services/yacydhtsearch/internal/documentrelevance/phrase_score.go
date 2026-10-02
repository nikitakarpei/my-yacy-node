package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

const (
	saturationOfQueryPhraseHits = 1.0
	phraseScoreOfUnreadDocument = 0.0
)

func phraseScoreOf(document queryfindings.FoundDocument) float64 {
	queryPhraseHits, queryPhrasesCounted := document.Facts.QueryPhraseHits.Get()
	if !queryPhrasesCounted {
		return phraseScoreOfUnreadDocument
	}

	return float64(queryPhraseHits) /
		(float64(queryPhraseHits) + saturationOfQueryPhraseHits)
}
