// Package documentrelevance tells how well each answered document answers the
// query. Its relevance adds up the share of the rarity of the query words its
// title holds, the share its text holds under a BM25 saturation, the query
// phrases, and how likely it is the entry page of a site the query names. From
// that sum it takes a penalty for a document that holds few links for the amount
// of words it holds. A query word nobody counted for a document scores as a word
// a node counted no hit of. A length or a link density nobody counted scores as
// the average of the documents a node counted.
package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Relevance struct {
	scoreWeights ScoreWeights
}

func New(scoreWeights ScoreWeights) Relevance {
	return Relevance{scoreWeights: scoreWeights}
}

func (relevance Relevance) RelevancePerDocumentOf(
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	foundDocuments := answers.FoundDocuments
	rarityOfTheQuery := queryRarityOf(
		answers.DocumentsHeldPerQueryWord, answers.QueryWords,
	)
	averageFacts := averageFactsAnyoneCountedAmong(answers.FactsPerDocument)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		relevancePerDocument[foundDocument.Hash] = relevance.scoreWeights.relevanceOf(
			foundDocument,
			answers.FactsPerDocument[foundDocument.Hash],
			rarityOfTheQuery,
			averageFacts,
			answers.QueryWords,
		)
	}

	return relevancePerDocument
}
