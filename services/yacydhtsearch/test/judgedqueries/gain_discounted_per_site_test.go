package judgedqueries_test

import (
	"math"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	judgedDocumentsCeiling  = 10
	gradeOfRelevantDocument = 1
	discountOfRepeatedSite  = 0.5

	severalRelevantDocuments = 2
)

type gradedDocument struct {
	grade int
	site  string
	spam  bool
}

type gradedDocuments map[yacymodel.URLHash]gradedDocument

func (documents gradedDocuments) normalizedGainOf(
	orderedDocuments []queryanswers.FoundDocument,
) float64 {
	idealGain := gainOf(documents.inTheIdealOrder())
	if idealGain == 0 {
		return 0
	}

	return gainOf(documents.inTheOrderOf(orderedDocuments)) / idealGain
}

func gainOf(rankedDocuments []gradedDocument) float64 {
	gain := 0.0
	sites := relevantDocumentsPerSite{}
	for rank, document := range rankedDocuments[:min(judgedDocumentsCeiling, len(rankedDocuments))] {
		gain += sites.discountedGainOf(document) / math.Log2(float64(rank)+2)
		sites.count(document)
	}

	return gain
}

type relevantDocumentsPerSite map[string]int

func (sites relevantDocumentsPerSite) discountedGainOf(document gradedDocument) float64 {
	return float64(document.grade) * math.Pow(
		1-discountOfRepeatedSite, float64(sites[document.site]),
	)
}

func (sites relevantDocumentsPerSite) count(document gradedDocument) {
	if document.grade < gradeOfRelevantDocument {
		return
	}
	sites[document.site]++
}

func (sites relevantDocumentsPerSite) placeOfTheMostGainingAmong(
	candidates []gradedDocument,
) int {
	mostGaining := 0
	for candidate := range candidates {
		if sites.discountedGainOf(candidates[candidate]) >
			sites.discountedGainOf(candidates[mostGaining]) {
			mostGaining = candidate
		}
	}

	return mostGaining
}

func (documents gradedDocuments) inTheIdealOrder() []gradedDocument {
	candidates := documents.inFallingOrderOfGrade()

	sites := relevantDocumentsPerSite{}
	idealDocuments := make([]gradedDocument, 0, min(judgedDocumentsCeiling, len(candidates)))
	for range cap(idealDocuments) {
		chosen := sites.placeOfTheMostGainingAmong(candidates)
		idealDocuments = append(idealDocuments, candidates[chosen])
		sites.count(candidates[chosen])
		candidates = slices.Delete(candidates, chosen, chosen+1)
	}

	return idealDocuments
}

func (documents gradedDocuments) inFallingOrderOfGrade() []gradedDocument {
	inFallingOrderOfGrade := make([]gradedDocument, 0, len(documents))
	for _, document := range documents {
		inFallingOrderOfGrade = append(inFallingOrderOfGrade, document)
	}
	slices.SortFunc(inFallingOrderOfGrade, func(one, other gradedDocument) int {
		if one.grade != other.grade {
			return other.grade - one.grade
		}

		return strings.Compare(one.site, other.site)
	})

	return inFallingOrderOfGrade
}

func (documents gradedDocuments) inTheOrderOf(
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
	return documents.amountOfRelevantDocuments() > 0
}

func (documents gradedDocuments) holdSeveralRelevantDocuments() bool {
	return documents.amountOfRelevantDocuments() >= severalRelevantDocuments
}

func (documents gradedDocuments) amountOfRelevantDocuments() int {
	amountOfRelevantDocuments := 0
	for _, document := range documents {
		if document.grade < gradeOfRelevantDocument {
			continue
		}
		amountOfRelevantDocuments++
	}

	return amountOfRelevantDocuments
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
	theFirstDocuments := orderedDocuments[:min(judgedDocumentsCeiling, len(orderedDocuments))]
	for _, orderedDocument := range theFirstDocuments {
		if !documents[orderedDocument.Hash].spam {
			continue
		}
		amountOfSpamDocuments++
	}

	return amountOfSpamDocuments
}
