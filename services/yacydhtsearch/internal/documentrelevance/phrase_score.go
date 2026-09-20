package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const saturationOfTheQueryPhraseHits = 1.0

func phraseScoreOf(foundDocument queryanswers.FoundDocument) yacymodel.Optional[float64] {
	hits, counted := foundDocument.QueryPhraseHitsOfTheReadPage.Get()
	if !counted {
		return yacymodel.None[float64]()
	}
	queryPhraseHits := float64(hits)

	return yacymodel.Some(queryPhraseHits / (queryPhraseHits + saturationOfTheQueryPhraseHits))
}
