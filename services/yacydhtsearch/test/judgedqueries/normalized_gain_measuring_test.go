package judgedqueries_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const gainTolerance = 1e-9

func TestTheSecondDocumentOfAHostCountsHalfOfADocumentOfAnotherHost(t *testing.T) {
	t.Parallel()

	gainOfOneHost := gainDiscountedPerRepeatedSubjectOf([]gradedDocument{
		{grade: 2, host: "one.example"},
		{grade: 2, host: "one.example"},
	}, hostOf)
	gainOfTwoHosts := gainDiscountedPerRepeatedSubjectOf([]gradedDocument{
		{grade: 2, host: "one.example"},
		{grade: 2, host: "two.example"},
	}, hostOf)

	wantOfOneHost := 2.0 + 2.0*discountOfARepeatedSubject/math.Log2(3)
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

	got := normalizedGainDiscountedPerHostOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2)},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2)},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2)},
	)

	idealGain := 2.0 + 2.0/math.Log2(3) + 2.0*discountOfARepeatedSubject/math.Log2(4)
	gainOfTheOrder := 2.0 + 2.0*discountOfARepeatedSubject/math.Log2(3) + 2.0/math.Log2(4)
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

	got := normalizedGainDiscountedPerHostOfTheDocumentsInOrder(t,
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

func TestTwoSubtopicsOfOneHostCountInFull(t *testing.T) {
	t.Parallel()

	got := normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2), subtopic: "cat"},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2), subtopic: "car"},
	)

	want := 1.0
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"two subtopics of one host reach %.6f of the ideal gain, want %.6f", got, want,
		)
	}
}

func TestOneSubtopicOnTwoHostsDiscountsTheSecondDocument(t *testing.T) {
	t.Parallel()

	got := normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2), subtopic: "cat"},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(1), subtopic: "cat"},
		documentToMeasure{address: "https://three.example/a", grade: gradeOf(1), subtopic: "car"},
	)

	idealGain := 2.0 + 1.0/math.Log2(3) + 1.0*discountOfARepeatedSubject/math.Log2(4)
	gainOfTheOrder := 2.0 + 1.0*discountOfARepeatedSubject/math.Log2(3) + 1.0/math.Log2(4)
	want := gainOfTheOrder / idealGain
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order that repeats a subtopic across hosts reaches %.6f, want %.6f", got, want,
		)
	}
}

func TestTheHostStandsInForTheSubtopicOfAnUnjudgedDocument(t *testing.T) {
	t.Parallel()

	got := normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2)},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2)},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2)},
	)
	want := normalizedGainDiscountedPerHostOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2)},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2)},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2)},
	)

	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"documents of no judged subtopic reach %.6f, want the gain per host %.6f", got, want,
		)
	}
}

func TestTheGainWithNoDiscountCountsEveryDocumentInFull(t *testing.T) {
	t.Parallel()

	got := normalizedGainWithNoDiscountOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2)},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2)},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2)},
	)

	want := 1.0
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"three documents of falling grade reach %.6f of the ideal gain, want %.6f", got, want,
		)
	}
}

type documentToMeasure struct {
	address  string
	grade    *int
	subtopic string
}

func gradeOf(grade int) *int {
	return &grade
}

func normalizedGainDiscountedPerHostOfTheDocumentsInOrder(
	t *testing.T, documentsInOrder ...documentToMeasure,
) float64 {
	t.Helper()

	graded, orderedItems := gradedDocumentsInTheOrderToMeasure(t, documentsInOrder)

	return graded.normalizedGainDiscountedPerHostOf(orderedItems)
}

func normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(
	t *testing.T, documentsInOrder ...documentToMeasure,
) float64 {
	t.Helper()

	graded, orderedItems := gradedDocumentsInTheOrderToMeasure(t, documentsInOrder)

	return graded.normalizedGainDiscountedPerSubtopicOf(orderedItems)
}

func normalizedGainWithNoDiscountOfTheDocumentsInOrder(
	t *testing.T, documentsInOrder ...documentToMeasure,
) float64 {
	t.Helper()

	graded, orderedItems := gradedDocumentsInTheOrderToMeasure(t, documentsInOrder)

	return graded.normalizedGainWithNoDiscountOf(orderedItems)
}

func gradedDocumentsInTheOrderToMeasure(
	t *testing.T, documentsInOrder []documentToMeasure,
) (gradedDocuments, []queryanswers.AnsweredItem) {
	t.Helper()

	graded := make(gradedDocuments, len(documentsInOrder))
	orderedItems := make([]queryanswers.AnsweredItem, 0, len(documentsInOrder))
	for _, document := range documentsInOrder {
		hash, err := yacymodel.URLHashOf(document.address)
		if err != nil {
			t.Fatalf("hash %s: %v", document.address, err)
		}
		if document.grade != nil {
			graded[hash] = gradedDocument{
				grade:          *document.grade,
				host:           hostOfTheAddress(document.address),
				judgedSubtopic: document.subtopic,
			}
		}
		orderedItems = append(orderedItems, queryanswers.AnsweredItem{
			Metadata: yacymodel.URLMetadata{Hash: hash, Address: document.address},
		})
	}

	return graded, orderedItems
}
