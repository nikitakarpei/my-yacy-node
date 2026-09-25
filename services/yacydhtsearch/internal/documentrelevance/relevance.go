// Package documentrelevance tells how well each document found for a query answers it.
// It weighs the title, the text, the query phrases and the site name of a document against
// how few links the document holds, with the relevance weights it is given. It takes the
// facts the peers sent for a document, or the facts of the page the service read for it.
package documentrelevance

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RelevanceScorer struct {
	relevanceWeights RelevanceWeights
}

func RelevanceScorerWeighedBy(relevanceWeights RelevanceWeights) RelevanceScorer {
	return RelevanceScorer{relevanceWeights: relevanceWeights}
}

func (relevanceScorer RelevanceScorer) RelevancePerDocumentOf(
	_ context.Context,
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	scoring := relevanceScorer.scoringOf(answers)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(answers.FoundDocuments))
	for _, document := range answers.FoundDocuments {
		relevancePerDocument[document.Hash] = scoring.relevanceOf(document)
	}

	return relevancePerDocument
}

func (relevanceScorer RelevanceScorer) scoringOf(answers queryanswers.AnsweredQuery) scoring {
	statistics := answersStatisticsFrom(answers)

	return scoring{
		relevanceWeights:     relevanceScorer.relevanceWeights,
		titleScorer:          titleScorerFrom(statistics),
		textScorer:           textScorerFrom(statistics),
		namedSiteEntryScorer: namedSiteEntryScorerFrom(statistics),
		linkSparsityPenalty:  linkSparsityPenaltyFrom(statistics),
	}
}

type scoring struct {
	relevanceWeights     RelevanceWeights
	titleScorer          titleScorer
	textScorer           textScorer
	namedSiteEntryScorer namedSiteEntryScorer
	linkSparsityPenalty  linkSparsityPenalty
}

func (scoring scoring) relevanceOf(document queryanswers.FoundDocument) float64 {
	return scoring.relevanceWeights.WeightOfTitleScore*scoring.titleScorer.scoreOf(document) +
		scoring.relevanceWeights.WeightOfTextScore*scoring.textScorer.scoreOf(document) +
		scoring.relevanceWeights.WeightOfPhraseScore*phraseScoreOf(document) +
		scoring.relevanceWeights.WeightOfNamedSiteEntryScore*
			scoring.namedSiteEntryScorer.scoreOf(document) -
		scoring.relevanceWeights.WeightOfLinkSparsityPenalty*
			scoring.linkSparsityPenalty.penaltyOf(document)
}
