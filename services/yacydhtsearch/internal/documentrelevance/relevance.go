// Package documentrelevance tells how well each document found for a query answers it.
// It weighs the title, the text, the query phrases and the site name of a document against
// how few links the document holds, with the relevance weights it is given. It takes the
// facts the peers sent for a document, or the facts of the page the service read for it.
package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Relevance struct {
	relevanceWeights RelevanceWeights
}

func New(relevanceWeights RelevanceWeights) Relevance {
	return Relevance{relevanceWeights: relevanceWeights}
}

func (relevance Relevance) RelevancePerDocumentOf(
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	foundDocuments := answers.FoundDocuments
	queryWordRarities := queryWordRaritiesFrom(answers)
	documentAverages := documentAveragesAmong(foundDocuments)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		relevancePerDocument[foundDocument.Hash] = relevance.relevanceOf(
			foundDocument,
			queryWordRarities,
			documentAverages,
			answers.QueryWords,
		)
	}

	return relevancePerDocument
}

func (relevance Relevance) relevanceOf(
	foundDocument queryanswers.FoundDocument,
	queryWordRarities queryWordRarities,
	documentAverages documentAverages,
	queryWords []yacymodel.Hash,
) float64 {
	return relevance.relevanceWeights.WeightOfTitleScore*
		titleScoreOf(foundDocument, queryWordRarities, queryWords) +
		relevance.relevanceWeights.WeightOfTextScore*textScoreOf(
			foundDocument, queryWordRarities, documentAverages.averageAmountOfWords, queryWords,
		) +
		relevance.relevanceWeights.WeightOfPhraseScore*phraseScoreOf(foundDocument) +
		relevance.relevanceWeights.WeightOfNamedSiteEntryScore*
			namedSiteEntryScoreOf(foundDocument, queryWords) -
		relevance.relevanceWeights.WeightOfLinkSparsityPenalty*
			linkSparsityPenaltyOf(foundDocument, documentAverages.averageLinkSparsityPenalty)
}
