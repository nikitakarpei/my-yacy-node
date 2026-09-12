// Package documentrelevance tells how well each answered document answers the
// query. Its relevance adds up how high the peers placed the document, the
// share of the rarity of the query words its title holds, the query words its
// host holds, a BM25 score of the query words in its text, the query phrases
// and the share of the query words its text holds.
package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Relevance struct {
	scoreWeights ScoreWeights
}

func New(scoreWeights ScoreWeights) Relevance {
	return Relevance{scoreWeights: scoreWeights}
}

func (relevance Relevance) RelevancePerDocumentOf(
	answers peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	items := answers.ItemOfEachAnsweredDocument()
	placeScorePerDocument := placeScorePerDocumentOf(answers.ItemsInTheOrderOfEachPeerRanking)
	rarityOfTheQueryWords := queryWordRarityOf(
		answers.DocumentsHeldPerQueryWord, answers.QueryWords,
	)
	averageDocumentLength := averageDocumentLengthOf(items)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(items))
	for _, item := range items {
		relevancePerDocument[item.Metadata.Hash] = relevance.scoreWeights.relevanceOf(
			item,
			placeScorePerDocument[item.Metadata.Hash],
			rarityOfTheQueryWords,
			averageDocumentLength,
			answers.QueryWords,
		)
	}

	return relevancePerDocument
}
