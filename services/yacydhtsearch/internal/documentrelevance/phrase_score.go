package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	saturationOfQueryPhraseHits = 1.0
	phraseScoreOfUnreadDocument = 0.0
)

func phraseScoreOf(foundDocument queryanswers.FoundDocument) float64 {
	hits, queryPhrasesCounted := foundDocument.Facts.QueryPhraseHits.Get()
	if !queryPhrasesCounted {
		return phraseScoreOfUnreadDocument
	}
	queryPhraseHits := float64(hits)

	return queryPhraseHits / (queryPhraseHits + saturationOfQueryPhraseHits)
}
