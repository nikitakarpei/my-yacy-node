package judgedqueries_test

import (
	"math"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	judgedDocumentsCeiling   = 10
	gradeOfARelevantDocument = 1
	discountOfARepeatedHost  = 0.5
)

type gradedDocument struct {
	grade int
	host  string
	spam  bool
}

type gradedDocuments map[yacymodel.URLHash]gradedDocument

func (documents gradedDocuments) normalizedGainDiscountedPerHostOf(
	orderedDocuments []queryanswers.FoundDocument,
) float64 {
	idealGain := gainDiscountedPerHostOf(documents.gradedDocumentsInTheIdealOrder())
	if idealGain == 0 {
		return 0
	}

	return gainDiscountedPerHostOf(
		documents.gradedDocumentsInTheOrderOf(orderedDocuments),
	) / idealGain
}

func gainDiscountedPerHostOf(rankedDocuments []gradedDocument) float64 {
	gain := 0.0
	amountOfRelevantDocumentsPerHost := map[string]int{}
	for rank, ranked := range rankedDocuments[:min(judgedDocumentsCeiling, len(rankedDocuments))] {
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
	idealDocuments := make([]gradedDocument, 0, min(judgedDocumentsCeiling, len(candidates)))
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
	orderedDocuments []queryanswers.FoundDocument,
) []gradedDocument {
	inTheOrderOfTheDocuments := make([]gradedDocument, 0, len(orderedDocuments))
	for _, orderedDocument := range orderedDocuments {
		document, graded := documents[orderedDocument.Hash]
		if !graded {
			continue
		}
		inTheOrderOfTheDocuments = append(inTheOrderOfTheDocuments, document)
	}

	return inTheOrderOfTheDocuments
}

func (documents gradedDocuments) holdARelevantDocument() bool {
	for _, document := range documents {
		if document.grade >= gradeOfARelevantDocument {
			return true
		}
	}

	return false
}

func (documents gradedDocuments) amountOfUngradedDocumentsAmong(
	orderedDocuments []queryanswers.FoundDocument,
) int {
	amountOfUngradedDocuments := 0
	for _, orderedDocument := range orderedDocuments {
		if _, graded := documents[orderedDocument.Hash]; graded {
			continue
		}
		amountOfUngradedDocuments++
	}

	return amountOfUngradedDocuments
}

func (documents gradedDocuments) amountOfSpamDocumentsAmongTheFirstOf(
	orderedDocuments []queryanswers.FoundDocument,
) int {
	amountOfSpamDocuments := 0
	for _, orderedDocument := range orderedDocuments[:min(judgedDocumentsCeiling, len(orderedDocuments))] {
		if !documents[orderedDocument.Hash].spam {
			continue
		}
		amountOfSpamDocuments++
	}

	return amountOfSpamDocuments
}
