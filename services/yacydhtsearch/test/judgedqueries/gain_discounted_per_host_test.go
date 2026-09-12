package judgedqueries_test

import (
	"math"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	judgedItemsCeiling       = 10
	gradeOfARelevantDocument = 1
	discountOfARepeatedHost  = 0.5
)

type gradedDocument struct {
	grade int
	host  string
}

type gradedDocuments map[yacymodel.URLHash]gradedDocument

func (documents gradedDocuments) normalizedGainDiscountedPerHostOf(
	orderedItems []peeranswers.AnsweredItem,
) float64 {
	idealGain := gainDiscountedPerHostOf(documents.gradedDocumentsInTheIdealOrder())
	if idealGain == 0 {
		return 0
	}

	return gainDiscountedPerHostOf(
		documents.gradedDocumentsInTheOrderOf(orderedItems),
	) / idealGain
}

func gainDiscountedPerHostOf(rankedDocuments []gradedDocument) float64 {
	gain := 0.0
	amountOfRelevantDocumentsPerHost := map[string]int{}
	for rank, ranked := range rankedDocuments[:min(judgedItemsCeiling, len(rankedDocuments))] {
		gain += ranked.gainAfter(amountOfRelevantDocumentsPerHost[ranked.host]) /
			math.Log2(float64(rank)+2)
		if ranked.grade >= gradeOfARelevantDocument {
			amountOfRelevantDocumentsPerHost[ranked.host]++
		}
	}

	return gain
}

func (document gradedDocument) gainAfter(
	amountOfRelevantDocumentsOfTheSameHostAbove int,
) float64 {
	return float64(document.grade) * math.Pow(
		1-discountOfARepeatedHost, float64(amountOfRelevantDocumentsOfTheSameHostAbove),
	)
}

func (documents gradedDocuments) gradedDocumentsInTheIdealOrder() []gradedDocument {
	candidates := documents.gradedDocumentsInFallingOrderOfGrade()

	amountOfRelevantDocumentsPerHost := map[string]int{}
	idealDocuments := make([]gradedDocument, 0, min(judgedItemsCeiling, len(candidates)))
	for range cap(idealDocuments) {
		chosen := placeOfTheMostGainingDocumentAmong(
			candidates, amountOfRelevantDocumentsPerHost,
		)
		idealDocuments = append(idealDocuments, candidates[chosen])
		if candidates[chosen].grade >= gradeOfARelevantDocument {
			amountOfRelevantDocumentsPerHost[candidates[chosen].host]++
		}
		candidates = slices.Delete(candidates, chosen, chosen+1)
	}

	return idealDocuments
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

func placeOfTheMostGainingDocumentAmong(
	candidates []gradedDocument,
	amountOfRelevantDocumentsPerHost map[string]int,
) int {
	mostGaining := 0
	for candidate := range candidates {
		gainOfTheCandidate := candidates[candidate].gainAfter(
			amountOfRelevantDocumentsPerHost[candidates[candidate].host],
		)
		gainOfTheMostGaining := candidates[mostGaining].gainAfter(
			amountOfRelevantDocumentsPerHost[candidates[mostGaining].host],
		)
		if gainOfTheCandidate > gainOfTheMostGaining {
			mostGaining = candidate
		}
	}

	return mostGaining
}

func (documents gradedDocuments) gradedDocumentsInTheOrderOf(
	orderedItems []peeranswers.AnsweredItem,
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
	orderedItems []peeranswers.AnsweredItem,
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
