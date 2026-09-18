package judgedqueries_test

import (
	"math"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	judgedItemsCeiling          = 10
	gradeOfARelevantDocument    = 1
	discountOfARepeatedSubtopic = 0.5
)

type gradedDocument struct {
	grade          int
	hash           yacymodel.URLHash
	judgedSubtopic string
}

type gradedDocuments map[yacymodel.URLHash]gradedDocument

func (documents gradedDocuments) normalizedGainDiscountedPerSubtopicOf(
	orderedItems []queryanswers.AnsweredItem,
) float64 {
	idealGain := gainDiscountedPerRepeatedSubtopicOf(
		documents.gradedDocumentsInTheIdealOrderPerSubtopic(),
	)
	if idealGain == 0 {
		return 0
	}

	return gainDiscountedPerRepeatedSubtopicOf(
		documents.gradedDocumentsInTheOrderOf(orderedItems),
	) / idealGain
}

func gainDiscountedPerRepeatedSubtopicOf(rankedDocuments []gradedDocument) float64 {
	gain := 0.0
	amountOfRelevantDocumentsPerSubtopic := map[string]int{}
	for rank, ranked := range rankedDocuments[:min(judgedItemsCeiling, len(rankedDocuments))] {
		subtopic := subtopicOf(ranked)
		gain += ranked.gainAfter(amountOfRelevantDocumentsPerSubtopic[subtopic]) /
			math.Log2(float64(rank)+2)
		if ranked.grade >= gradeOfARelevantDocument {
			amountOfRelevantDocumentsPerSubtopic[subtopic]++
		}
	}

	return gain
}

func subtopicOf(document gradedDocument) string {
	if document.judgedSubtopic == "" {
		return document.hash.String()
	}

	return document.judgedSubtopic
}

func (document gradedDocument) gainAfter(
	amountOfRelevantDocumentsOfTheSameSubtopicAbove int,
) float64 {
	return float64(document.grade) * math.Pow(
		1-discountOfARepeatedSubtopic, float64(amountOfRelevantDocumentsOfTheSameSubtopicAbove),
	)
}

func (documents gradedDocuments) gradedDocumentsInTheIdealOrderPerSubtopic() []gradedDocument {
	candidates := documents.gradedDocumentsInFallingOrderOfGrade()

	amountOfRelevantDocumentsPerSubtopic := map[string]int{}
	idealDocuments := make([]gradedDocument, 0, min(judgedItemsCeiling, len(candidates)))
	for range cap(idealDocuments) {
		chosen := placeOfTheMostGainingDocumentAmong(
			candidates, amountOfRelevantDocumentsPerSubtopic,
		)
		idealDocuments = append(idealDocuments, candidates[chosen])
		if candidates[chosen].grade >= gradeOfARelevantDocument {
			amountOfRelevantDocumentsPerSubtopic[subtopicOf(candidates[chosen])]++
		}
		candidates = slices.Delete(candidates, chosen, chosen+1)
	}

	return idealDocuments
}

func placeOfTheMostGainingDocumentAmong(
	candidates []gradedDocument,
	amountOfRelevantDocumentsPerSubtopic map[string]int,
) int {
	mostGaining := 0
	for candidate := range candidates {
		gainOfTheCandidate := candidates[candidate].gainAfter(
			amountOfRelevantDocumentsPerSubtopic[subtopicOf(candidates[candidate])],
		)
		gainOfTheMostGaining := candidates[mostGaining].gainAfter(
			amountOfRelevantDocumentsPerSubtopic[subtopicOf(candidates[mostGaining])],
		)
		if gainOfTheCandidate > gainOfTheMostGaining {
			mostGaining = candidate
		}
	}

	return mostGaining
}

func (documents gradedDocuments) normalizedGainWithNoDiscountOf(
	orderedItems []queryanswers.AnsweredItem,
) float64 {
	idealGain := gainWithNoDiscountOf(documents.gradedDocumentsInFallingOrderOfGrade())
	if idealGain == 0 {
		return 0
	}

	return gainWithNoDiscountOf(documents.gradedDocumentsInTheOrderOf(orderedItems)) / idealGain
}

func gainWithNoDiscountOf(rankedDocuments []gradedDocument) float64 {
	gain := 0.0
	for rank, ranked := range rankedDocuments[:min(judgedItemsCeiling, len(rankedDocuments))] {
		gain += float64(ranked.grade) / math.Log2(float64(rank)+2)
	}

	return gain
}

func (documents gradedDocuments) gradedDocumentsInFallingOrderOfGrade() []gradedDocument {
	inFallingOrderOfGrade := make([]gradedDocument, 0, len(documents))
	for _, document := range documents {
		inFallingOrderOfGrade = append(inFallingOrderOfGrade, document)
	}
	slices.SortFunc(inFallingOrderOfGrade, func(one, other gradedDocument) int {
		if one.grade != other.grade {
			return other.grade - one.grade
		}
		if one.judgedSubtopic != other.judgedSubtopic {
			return strings.Compare(one.judgedSubtopic, other.judgedSubtopic)
		}

		return strings.Compare(one.hash.String(), other.hash.String())
	})

	return inFallingOrderOfGrade
}

func (documents gradedDocuments) gradedDocumentsInTheOrderOf(
	orderedItems []queryanswers.AnsweredItem,
) []gradedDocument {
	inTheOrderOfTheItems := make([]gradedDocument, 0, len(orderedItems))
	for _, orderedItem := range orderedItems {
		document, graded := documents[orderedItem.Metadata.Hash]
		if !graded {
			continue
		}
		inTheOrderOfTheItems = append(inTheOrderOfTheItems, document)
	}

	return inTheOrderOfTheItems
}

func (documents gradedDocuments) holdARelevantDocument() bool {
	for _, document := range documents {
		if document.grade >= gradeOfARelevantDocument {
			return true
		}
	}

	return false
}

func (documents gradedDocuments) amountOfUngradedItemsAmong(
	orderedItems []queryanswers.AnsweredItem,
) int {
	amountOfUngradedItems := 0
	for _, orderedItem := range orderedItems {
		if _, graded := documents[orderedItem.Metadata.Hash]; graded {
			continue
		}
		amountOfUngradedItems++
	}

	return amountOfUngradedItems
}
