package judgedqueries_test

import (
	"math"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	judgedItemsCeiling       = 10
	gradeOfARelevantDocument = 1
)

type gradedDocuments map[yacymodel.URLHash]int

func (documents gradedDocuments) normalizedGainOf(
	orderedItems []peeranswers.AnsweredItem,
) float64 {
	idealGain := discountedGainOf(documents.gradesInFallingOrder())
	if idealGain == 0 {
		return 0
	}

	return discountedGainOf(documents.gradesInTheOrderOf(orderedItems)) / idealGain
}

func (documents gradedDocuments) holdARelevantDocument() bool {
	for _, grade := range documents {
		if grade >= gradeOfARelevantDocument {
			return true
		}
	}

	return false
}

func (documents gradedDocuments) gradesInFallingOrder() []int {
	grades := make([]int, 0, len(documents))
	for _, grade := range documents {
		grades = append(grades, grade)
	}
	slices.SortFunc(grades, func(one, other int) int { return other - one })

	return grades
}

func (documents gradedDocuments) gradesInTheOrderOf(
	orderedItems []peeranswers.AnsweredItem,
) []int {
	grades := make([]int, 0, len(orderedItems))
	for _, orderedItem := range orderedItems {
		grades = append(grades, documents[orderedItem.Metadata.Hash])
	}

	return grades
}

func discountedGainOf(grades []int) float64 {
	gain := 0.0
	for place, grade := range grades[:min(judgedItemsCeiling, len(grades))] {
		gain += (math.Exp2(float64(grade)) - 1) / math.Log2(float64(place)+2)
	}

	return gain
}
