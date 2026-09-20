// Package documentrelevance tells how well each answered document answers the
// query. Its relevance adds up the share of the rarity of the query words its
// title holds, a BM25 score of the query words in its text, the query phrases,
// and how likely it is the entry page of a site the query names. From that sum
// it takes a penalty for a document that holds few links for the amount of words
// it holds.
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
	rarityOfTheQueryWords := queryWordRarityOf(
		answers.DocumentsHeldPerQueryWord, answers.QueryWords,
	)
	averageDocumentLength := averageDocumentLengthOf(foundDocuments)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		relevancePerDocument[foundDocument.Hash] = relevance.scoreWeights.relevanceOf(
			foundDocument,
			rarityOfTheQueryWords,
			averageDocumentLength,
			answers.QueryWords,
		)
	}

	return relevancePerDocument
}
