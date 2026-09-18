package judgedqueries_test

import (
	"math"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	judgedItemsCeiling         = 10
	gradeOfARelevantDocument   = 1
	discountOfARepeatedSubject = 0.5
)

type gradedDocument struct {
	grade          int
	host           string
	judgedSubtopic string
}

type gradedDocuments map[yacymodel.URLHash]gradedDocument

func hostOf(document gradedDocument) string {
	return document.host
}

func subtopicOf(document gradedDocument) string {
	if document.judgedSubtopic == "" {
		return document.host
	}

	return document.judgedSubtopic
}

func (documents gradedDocuments) normalizedGainDiscountedPerHostOf(
	orderedItems []queryanswers.AnsweredItem,
) float64 {
	return documents.normalizedGainDiscountedPerRepeatedSubjectOf(orderedItems, hostOf)
}

func (documents gradedDocuments) normalizedGainDiscountedPerSubtopicOf(
	orderedItems []queryanswers.AnsweredItem,
) float64 {
	return documents.normalizedGainDiscountedPerRepeatedSubjectOf(orderedItems, subtopicOf)
}

func (documents gradedDocuments) normalizedGainDiscountedPerRepeatedSubjectOf(
	orderedItems []queryanswers.AnsweredItem,
	repeatedSubjectOf func(gradedDocument) string,
) float64 {
	idealGain := gainDiscountedPerRepeatedSubjectOf(
		documents.gradedDocumentsInTheIdealOrderOf(repeatedSubjectOf), repeatedSubjectOf,
	)
	if idealGain == 0 {
		return 0
	}

	return gainDiscountedPerRepeatedSubjectOf(
		documents.gradedDocumentsInTheOrderOf(orderedItems), repeatedSubjectOf,
	) / idealGain
}

func gainDiscountedPerRepeatedSubjectOf(
	rankedDocuments []gradedDocument, repeatedSubjectOf func(gradedDocument) string,
) float64 {
	gain := 0.0
	amountOfRelevantDocumentsPerSubject := map[string]int{}
	for rank, ranked := range rankedDocuments[:min(judgedItemsCeiling, len(rankedDocuments))] {
		subject := repeatedSubjectOf(ranked)
		gain += ranked.gainAfter(amountOfRelevantDocumentsPerSubject[subject]) /
			math.Log2(float64(rank)+2)
		if ranked.grade >= gradeOfARelevantDocument {
			amountOfRelevantDocumentsPerSubject[subject]++
		}
	}

	return gain
}

func (document gradedDocument) gainAfter(
	amountOfRelevantDocumentsOfTheSameSubjectAbove int,
) float64 {
	return float64(document.grade) * math.Pow(
		1-discountOfARepeatedSubject, float64(amountOfRelevantDocumentsOfTheSameSubjectAbove),
	)
}

func (documents gradedDocuments) gradedDocumentsInTheIdealOrderOf(
	repeatedSubjectOf func(gradedDocument) string,
) []gradedDocument {
	candidates := documents.gradedDocumentsInFallingOrderOfGrade()

	amountOfRelevantDocumentsPerSubject := map[string]int{}
	idealDocuments := make([]gradedDocument, 0, min(judgedItemsCeiling, len(candidates)))
	for range cap(idealDocuments) {
		chosen := placeOfTheMostGainingDocumentAmong(
			candidates, amountOfRelevantDocumentsPerSubject, repeatedSubjectOf,
		)
		idealDocuments = append(idealDocuments, candidates[chosen])
		if candidates[chosen].grade >= gradeOfARelevantDocument {
			amountOfRelevantDocumentsPerSubject[repeatedSubjectOf(candidates[chosen])]++
		}
		candidates = slices.Delete(candidates, chosen, chosen+1)
	}

	return idealDocuments
}

func placeOfTheMostGainingDocumentAmong(
	candidates []gradedDocument,
	amountOfRelevantDocumentsPerSubject map[string]int,
	repeatedSubjectOf func(gradedDocument) string,
) int {
	mostGaining := 0
	for candidate := range candidates {
		gainOfTheCandidate := candidates[candidate].gainAfter(
			amountOfRelevantDocumentsPerSubject[repeatedSubjectOf(candidates[candidate])],
		)
		gainOfTheMostGaining := candidates[mostGaining].gainAfter(
			amountOfRelevantDocumentsPerSubject[repeatedSubjectOf(candidates[mostGaining])],
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

		return strings.Compare(one.host, other.host)
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

func (documents gradedDocuments) holdAJudgedSubtopic() bool {
	for _, document := range documents {
		if document.judgedSubtopic != "" {
			return true
		}
	}

	return false
}
