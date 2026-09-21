package judgedqueries_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const gainTolerance = 1e-9

func TestTheSecondDocumentOfASiteCountsHalfOfADocumentOfAnotherSite(t *testing.T) {
	t.Parallel()

	gainOfOneSite := gainOf([]gradedDocument{
		{grade: 2, site: "one.example"},
		{grade: 2, site: "one.example"},
	})
	gainOfTwoSites := gainOf([]gradedDocument{
		{grade: 2, site: "one.example"},
		{grade: 2, site: "two.example"},
	})

	wantOfOneSite := 2.0 + 2.0*discountOfRepeatedSite/math.Log2(3)
	wantOfTwoSites := 2.0 + 2.0/math.Log2(3)
	if math.Abs(gainOfOneSite-wantOfOneSite) > gainTolerance {
		t.Errorf(
			"two documents of one site reach the gain %.6f, want %.6f",
			gainOfOneSite, wantOfOneSite,
		)
	}
	if math.Abs(gainOfTwoSites-wantOfTwoSites) > gainTolerance {
		t.Errorf(
			"two documents of two sites reach the gain %.6f, want %.6f",
			gainOfTwoSites, wantOfTwoSites,
		)
	}
}

func TestTheIdealOrderPutsTheFirstDocumentOfAnotherSiteFirst(t *testing.T) {
	t.Parallel()

	got := normalizedGainOf(t,
		documentInTheOrder{address: "https://one.example/a", grade: yacymodel.Some(2)},
		documentInTheOrder{address: "https://one.example/b", grade: yacymodel.Some(2)},
		documentInTheOrder{address: "https://two.example/a", grade: yacymodel.Some(2)},
	)

	idealGain := 2.0 + 2.0/math.Log2(3) + 2.0*discountOfRepeatedSite/math.Log2(4)
	gainOfTheOrder := 2.0 + 2.0*discountOfRepeatedSite/math.Log2(3) + 2.0/math.Log2(4)
	want := gainOfTheOrder / idealGain
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order that repeats a site before another site reaches %.6f, want %.6f",
			got, want,
		)
	}
	if got >= 1 {
		t.Errorf("the order that repeats a site reaches %.6f, want less than the ideal", got)
	}
}

func TestAnUngradedDocumentOfTheSameSiteDiscountsNothing(t *testing.T) {
	t.Parallel()

	got := normalizedGainOf(t,
		documentInTheOrder{address: "https://one.example/a"},
		documentInTheOrder{address: "https://one.example/b", grade: yacymodel.Some(2)},
		documentInTheOrder{address: "https://two.example/a", grade: yacymodel.Some(2)},
	)

	want := 1.0
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order under an ungraded document of the same site reaches %.6f, want %.6f",
			got, want,
		)
	}
}

type documentInTheOrder struct {
	address string
	grade   yacymodel.Optional[int]
}

func normalizedGainOf(t *testing.T, documentsInOrder ...documentInTheOrder) float64 {
	t.Helper()

	graded := make(gradedDocuments, len(documentsInOrder))
	orderedDocuments := make([]queryanswers.FoundDocument, 0, len(documentsInOrder))
	for _, document := range documentsInOrder {
		hash, err := yacymodel.URLHashOf(document.address)
		if err != nil {
			t.Fatalf("hash %s: %v", document.address, err)
		}
		if grade, present := document.grade.Get(); present {
			graded[hash] = gradedDocument{
				grade: grade,
				site:  yacymodel.SiteOf(document.address),
			}
		}
		orderedDocuments = append(
			orderedDocuments, queryanswers.FoundDocument{Hash: hash, Address: document.address},
		)
	}

	return graded.normalizedGainOf(orderedDocuments)
}
