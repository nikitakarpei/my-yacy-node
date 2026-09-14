package stalenessdiscount_test

import (
	"math"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalenessdiscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	addressOfTheDocument   = "https://one.example/a"
	relevanceOfTheDocument = 10.0
	toleratedDifference    = 1e-9
)

var dayOfTheSearch = yacymodel.NewCalendarDay(2026, time.September, 14)

type relevanceOfTheGivenDocuments struct {
	relevancePerDocument map[yacymodel.URLHash]float64
}

func (given relevanceOfTheGivenDocuments) RelevancePerDocumentOf(
	_ peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	return given.relevancePerDocument
}

func relevanceOfADocumentModified(
	t *testing.T, dayTheDocumentWasModified yacymodel.Optional[yacymodel.CalendarDay],
) float64 {
	t.Helper()

	hash, err := yacymodel.URLHashOf(addressOfTheDocument)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", addressOfTheDocument, err)
	}

	relevancePerDocument := stalenessdiscount.New(
		relevanceOfTheGivenDocuments{
			relevancePerDocument: map[yacymodel.URLHash]float64{
				hash: relevanceOfTheDocument,
			},
		},
		dayOfTheSearch.Time,
	).RelevancePerDocumentOf(peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{{
			{Metadata: yacymodel.URLMetadata{
				Hash:     hash,
				Address:  addressOfTheDocument,
				Modified: dayTheDocumentWasModified,
			}},
		}},
	})

	return relevancePerDocument[hash]
}

func daysBeforeTheSearch(amountOfDays int) yacymodel.Optional[yacymodel.CalendarDay] {
	return yacymodel.Some(
		yacymodel.CalendarDayOf(dayOfTheSearch.Time().AddDate(0, 0, -amountOfDays)),
	)
}

func TestADocumentNoPeerDatesKeepsItsWholeRelevance(t *testing.T) {
	t.Parallel()

	got := relevanceOfADocumentModified(t, yacymodel.None[yacymodel.CalendarDay]())
	if math.Abs(got-relevanceOfTheDocument) > toleratedDifference {
		t.Fatalf("an undated document holds the relevance %v, want %v", got, relevanceOfTheDocument)
	}
}

func TestADocumentModifiedOnTheDayOfTheSearchKeepsItsWholeRelevance(t *testing.T) {
	t.Parallel()

	got := relevanceOfADocumentModified(t, daysBeforeTheSearch(0))
	if math.Abs(got-relevanceOfTheDocument) > toleratedDifference {
		t.Fatalf(
			"a document of the day holds the relevance %v, want %v",
			got,
			relevanceOfTheDocument,
		)
	}
}

func TestADocumentDatedAfterTheDayOfTheSearchKeepsItsWholeRelevance(t *testing.T) {
	t.Parallel()

	got := relevanceOfADocumentModified(t, daysBeforeTheSearch(-3650))
	if math.Abs(got-relevanceOfTheDocument) > toleratedDifference {
		t.Fatalf("a document of a later day holds the relevance %v, want %v",
			got, relevanceOfTheDocument)
	}
}

func TestADocumentOfOneYearLosesAQuarterOfItsRelevance(t *testing.T) {
	t.Parallel()

	got := relevanceOfADocumentModified(t, daysBeforeTheSearch(365))
	if want := 0.75 * relevanceOfTheDocument; math.Abs(got-want) > toleratedDifference {
		t.Fatalf("a document of one year holds the relevance %v, want %v", got, want)
	}
}

func TestADocumentOfTwoYearsLosesThreeQuartersOfTheRelevanceAtStake(t *testing.T) {
	t.Parallel()

	got := relevanceOfADocumentModified(t, daysBeforeTheSearch(2*365))
	if want := 0.625 * relevanceOfTheDocument; math.Abs(got-want) > toleratedDifference {
		t.Fatalf("a document of two years holds the relevance %v, want %v", got, want)
	}
}

func TestTheStalestDocumentStillKeepsHalfOfItsRelevance(t *testing.T) {
	t.Parallel()

	got := relevanceOfADocumentModified(t, daysBeforeTheSearch(100*365))
	if want := 0.5 * relevanceOfTheDocument; got < want ||
		math.Abs(got-want) > toleratedDifference {
		t.Fatalf("a document of a hundred years holds the relevance %v, want %v", got, want)
	}
}

func TestAFresherDocumentKeepsMoreOfItsRelevanceThanAnOlderOne(t *testing.T) {
	t.Parallel()

	fresher := relevanceOfADocumentModified(t, daysBeforeTheSearch(30))
	older := relevanceOfADocumentModified(t, daysBeforeTheSearch(31))
	if fresher <= older {
		t.Fatalf("the fresher document holds the relevance %v, want more than %v", fresher, older)
	}
}

func TestNoAnsweredItemMakesNoDiscountedRelevance(t *testing.T) {
	t.Parallel()

	relevancePerDocument := stalenessdiscount.New(
		relevanceOfTheGivenDocuments{relevancePerDocument: map[yacymodel.URLHash]float64{}},
		dayOfTheSearch.Time,
	).RelevancePerDocumentOf(peeranswers.AnsweredQuery{})

	if len(relevancePerDocument) != 0 {
		t.Fatalf("the discounted relevance holds %v, want no document", relevancePerDocument)
	}
}
