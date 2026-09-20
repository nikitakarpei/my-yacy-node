package judgedqueries_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const everySecondDocument = 2

func TestTheOrderingReportsThePlacesLostWithoutTheFactsOfAPageRead(t *testing.T) {
	t.Parallel()

	judged := judgedQueriesRecorded(t)
	ordering := orderingOfTheServiceFrom(documentrelevance.DefaultScoreWeights())
	placesLostByThePhrasedDocuments, phrasedDocumentsHidden := 0, 0
	placesLostByTheOtherDocuments, otherDocumentsHidden := 0, 0
	documentsLeavingTheFirstTen := 0
	sumOfGains, sumOfGainsWithoutTheFacts := 0.0, 0.0
	for _, judgedQuery := range judged {
		hidden := everySecondDocumentWhosePageANodeReadOf(judgedQuery.answers)
		placesOfTheAnswers := placePerDocumentIn(
			ordering.OrderedDocumentsOf(judgedQuery.answers),
		)
		answersWithoutTheFacts := answersWithoutTheFactsOfThePageReadsOf(
			judgedQuery.answers, hidden,
		)
		placesWithoutTheFacts := placePerDocumentIn(
			ordering.OrderedDocumentsOf(answersWithoutTheFacts),
		)
		for document, phrased := range hidden {
			if phrased {
				phrasedDocumentsHidden++
				placesLostByThePhrasedDocuments +=
					placesWithoutTheFacts[document] - placesOfTheAnswers[document]
			} else {
				otherDocumentsHidden++
				placesLostByTheOtherDocuments +=
					placesWithoutTheFacts[document] - placesOfTheAnswers[document]
			}
			if placesOfTheAnswers[document] < judgedDocumentsCeiling &&
				placesWithoutTheFacts[document] >= judgedDocumentsCeiling {
				documentsLeavingTheFirstTen++
			}
		}
		sumOfGains += judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
			ordering.OrderedDocumentsOf(judgedQuery.answers),
		)
		sumOfGainsWithoutTheFacts += judgedQuery.gradedDocuments.
			normalizedGainDiscountedPerHostOf(
				ordering.OrderedDocumentsOf(answersWithoutTheFacts),
			)
	}
	t.Logf(
		"over %d judged queries, hiding the facts of the page read costs the %d documents whose "+
			"text holds a query phrase %.2f places each, and the %d other documents %.2f places "+
			"each; %d documents leave the first ten",
		len(judged),
		phrasedDocumentsHidden,
		float64(placesLostByThePhrasedDocuments)/float64(max(phrasedDocumentsHidden, 1)),
		otherDocumentsHidden,
		float64(placesLostByTheOtherDocuments)/float64(max(otherDocumentsHidden, 1)),
		documentsLeavingTheFirstTen,
	)
	t.Logf(
		"the mean gain falls from %.4f to %.4f",
		sumOfGains/float64(len(judged)),
		sumOfGainsWithoutTheFacts/float64(len(judged)),
	)
}

func everySecondDocumentWhosePageANodeReadOf(
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]bool {
	hidden := map[yacymodel.URLHash]bool{}
	documentsRead := 0
	for _, foundDocument := range answers.FoundDocuments {
		if _, read := foundDocument.LinkCounts.Get(); !read {
			continue
		}
		documentsRead++
		if documentsRead%everySecondDocument != 0 {
			continue
		}
		hitsOfTheReadPage, counted := foundDocument.QueryPhraseHitsOfTheReadPage.Get()
		hidden[foundDocument.Hash] = counted && hitsOfTheReadPage > 0
	}

	return hidden
}

func answersWithoutTheFactsOfThePageReadsOf(
	answers queryanswers.AnsweredQuery,
	hidden map[yacymodel.URLHash]bool,
) queryanswers.AnsweredQuery {
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(answers.FoundDocuments))
	for _, foundDocument := range answers.FoundDocuments {
		if _, hiddenDocument := hidden[foundDocument.Hash]; hiddenDocument {
			foundDocument.QueryPhraseHitsOfTheReadPage = yacymodel.None[int]()
			foundDocument.LinkCounts = yacymodel.None[pagecontents.LinkCounts]()
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	answers.FoundDocuments = foundDocuments

	return answers
}

func placePerDocumentIn(
	orderedDocuments []queryanswers.FoundDocument,
) map[yacymodel.URLHash]int {
	placePerDocument := make(map[yacymodel.URLHash]int, len(orderedDocuments))
	for place, orderedDocument := range orderedDocuments {
		placePerDocument[orderedDocument.Hash] = place
	}

	return placePerDocument
}
