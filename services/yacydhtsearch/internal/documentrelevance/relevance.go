// Package documentrelevance tells how well each answered document answers the
// query. Its relevance adds up the share of the rarity of the query its title
// holds, the share its text holds under a BM25 saturation, the query phrases,
// and how likely it is the entry page of a site the query names, and takes a
// penalty from that sum for a document that holds few links for its words. A
// document without a title loses a full share of the rarity of the query.
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
	rarityOfTheQuery := queryRarityOf(answers.DocumentsHeldPerQueryWord, answers.QueryWords)
	averagesAnyoneCounted := averagesAnyoneCountedAmong(answers.FactsPerDocument)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		relevancePerDocument[foundDocument.Hash] = relevance.relevanceOf(
			foundDocument,
			answers.FactsPerDocument[foundDocument.Hash],
			rarityOfTheQuery,
			averagesAnyoneCounted,
			answers.QueryWords,
		)
	}

	return relevancePerDocument
}

func (relevance Relevance) relevanceOf(
	foundDocument queryanswers.FoundDocument,
	facts queryanswers.DocumentFacts,
	rarityOfTheQuery queryRarity,
	averagesAnyoneCounted averagesAnyoneCounted,
	queryWords []yacymodel.Hash,
) float64 {
	weights := relevance.scoreWeights

	return weights.WeightOfTheTitleScore*
		titleScoreOf(foundDocument, rarityOfTheQuery, queryWords) +
		weights.WeightOfTheTextScore*
			textScoreOf(facts, rarityOfTheQuery, averagesAnyoneCounted.amountOfWords, queryWords) +
		weights.WeightOfThePhraseScore*phraseScoreOf(facts) +
		weights.WeightOfTheNamedSiteEntryScore*
			namedSiteEntryScoreOf(foundDocument, queryWords) -
		weights.WeightOfTheLinkSparsityPenalty*
			linkSparsityPenaltyOf(facts, averagesAnyoneCounted.linkSparsityPenalty)
}
