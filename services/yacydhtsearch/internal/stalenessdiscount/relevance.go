// Package stalenessdiscount tells how well each answered document answers the
// query, with the relevance of an older document made smaller. Half of the
// relevance of a document is at stake to its age, and the document keeps half
// of that share for each year since the day the peers say it was modified. A
// document the peers do not date, or date on the day of the search or after
// it, keeps its whole relevance.
package stalenessdiscount

import (
	"math"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shareOfRelevanceKeptByTheStalestDocument   = 0.5
	shareOfTheRelevanceAtStakeKeptPerYearOfAge = 0.5
	yearOfAge                                  = 365 * 24 * time.Hour
)

type DocumentRelevance interface {
	RelevancePerDocumentOf(answers peeranswers.AnsweredQuery) map[yacymodel.URLHash]float64
}

type Relevance struct {
	documentRelevance DocumentRelevance
	now               func() time.Time
}

func New(documentRelevance DocumentRelevance, now func() time.Time) Relevance {
	return Relevance{documentRelevance: documentRelevance, now: now}
}

func (relevance Relevance) RelevancePerDocumentOf(
	answers peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	relevancePerDocument := relevance.documentRelevance.RelevancePerDocumentOf(answers)
	items := answers.ItemOfEachAnsweredDocument()
	dayOfTheSearch := yacymodel.CalendarDayOf(relevance.now())

	discountedRelevancePerDocument := make(map[yacymodel.URLHash]float64, len(items))
	for _, item := range items {
		discountedRelevancePerDocument[item.Metadata.Hash] =
			relevancePerDocument[item.Metadata.Hash] *
				shareOfRelevanceKeptFrom(item.Metadata.Modified, dayOfTheSearch)
	}

	return discountedRelevancePerDocument
}

func shareOfRelevanceKeptFrom(
	dayTheDocumentWasModified yacymodel.Optional[yacymodel.CalendarDay],
	dayOfTheSearch yacymodel.CalendarDay,
) float64 {
	modified, dated := dayTheDocumentWasModified.Get()
	if !dated {
		return 1
	}

	return shareOfRelevanceKeptAtTheAgeOf(yearsOfAgeBetween(modified, dayOfTheSearch))
}

func yearsOfAgeBetween(
	dayTheDocumentWasModified yacymodel.CalendarDay,
	dayOfTheSearch yacymodel.CalendarDay,
) float64 {
	age := dayOfTheSearch.Time().Sub(dayTheDocumentWasModified.Time())

	return float64(max(age, 0)) / float64(yearOfAge)
}

func shareOfRelevanceKeptAtTheAgeOf(yearsOfAge float64) float64 {
	shareOfTheRelevanceAtStake := 1 - shareOfRelevanceKeptByTheStalestDocument

	return shareOfRelevanceKeptByTheStalestDocument +
		shareOfTheRelevanceAtStake*
			math.Pow(shareOfTheRelevanceAtStakeKeptPerYearOfAge, yearsOfAge)
}
