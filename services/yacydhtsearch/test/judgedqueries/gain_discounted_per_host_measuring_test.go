package judgedqueries_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const gainTolerance = 1e-9

func TestTheSecondDocumentOfAHostCountsHalfOfADocumentOfAnotherHost(t *testing.T) {
	t.Parallel()

	gainOfOneHost := gainDiscountedPerHostOf([]gradedDocument{
		{grade: 2, host: "one.example"},
		{grade: 2, host: "one.example"},
	})
	gainOfTwoHosts := gainDiscountedPerHostOf([]gradedDocument{
		{grade: 2, host: "one.example"},
		{grade: 2, host: "two.example"},
	})

	wantOfOneHost := 2.0 + 2.0*discountOfARepeatedHost/math.Log2(3)
	wantOfTwoHosts := 2.0 + 2.0/math.Log2(3)
	if math.Abs(gainOfOneHost-wantOfOneHost) > gainTolerance {
		t.Errorf(
			"two documents of one host reach the gain %.6f, want %.6f",
			gainOfOneHost, wantOfOneHost,
		)
	}
	if math.Abs(gainOfTwoHosts-wantOfTwoHosts) > gainTolerance {
		t.Errorf(
			"two documents of two hosts reach the gain %.6f, want %.6f",
			gainOfTwoHosts, wantOfTwoHosts,
		)
	}
}

func TestTheIdealOrderPutsTheFirstDocumentOfAnotherHostFirst(t *testing.T) {
	t.Parallel()

	got := normalizedGainOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2)},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2)},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2)},
	)

	idealGain := 2.0 + 2.0/math.Log2(3) + 2.0*discountOfARepeatedHost/math.Log2(4)
	gainOfTheOrder := 2.0 + 2.0*discountOfARepeatedHost/math.Log2(3) + 2.0/math.Log2(4)
	want := gainOfTheOrder / idealGain
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order that repeats a host before another host reaches %.6f, want %.6f",
			got, want,
		)
	}
	if got >= 1 {
		t.Errorf("the order that repeats a host reaches %.6f, want less than the ideal", got)
	}
}

func TestAnUngradedDocumentOfTheSameHostDiscountsNothing(t *testing.T) {
	t.Parallel()

	got := normalizedGainOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a"},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2)},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2)},
	)

	want := 1.0
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order under an ungraded document of the same host reaches %.6f, want %.6f",
			got, want,
		)
	}
}

type documentToMeasure struct {
	address string
	grade   *int
}

func gradeOf(grade int) *int {
	return &grade
}

func normalizedGainOfTheDocumentsInOrder(
	t *testing.T, documentsInOrder ...documentToMeasure,
) float64 {
	t.Helper()

	graded := make(gradedDocuments, len(documentsInOrder))
	orderedItems := make([]peeranswers.AnsweredItem, 0, len(documentsInOrder))
	for _, document := range documentsInOrder {
		hash, err := yacymodel.URLHashOf(document.address)
		if err != nil {
			t.Fatalf("hash %s: %v", document.address, err)
		}
		if document.grade != nil {
			graded[hash] = gradedDocument{
				grade: *document.grade,
				host:  hostOf(document.address),
			}
		}
		orderedItems = append(orderedItems, peeranswers.AnsweredItem{
			Metadata: yacymodel.URLMetadata{Hash: hash, Address: document.address},
		})
	}

	return graded.normalizedGainDiscountedPerHostOf(orderedItems)
}
