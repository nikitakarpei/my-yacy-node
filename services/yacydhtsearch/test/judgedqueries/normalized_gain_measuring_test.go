package judgedqueries_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const gainTolerance = 1e-9

func TestTheSecondDocumentOfASubtopicCountsHalfOfADocumentOfAnotherSubtopic(t *testing.T) {
	t.Parallel()

	gainOfOneSubtopic := gainDiscountedPerRepeatedSubtopicOf([]gradedDocument{
		{grade: 2, judgedSubtopic: "cat"},
		{grade: 2, judgedSubtopic: "cat"},
	})
	gainOfTwoSubtopics := gainDiscountedPerRepeatedSubtopicOf([]gradedDocument{
		{grade: 2, judgedSubtopic: "cat"},
		{grade: 2, judgedSubtopic: "car"},
	})

	wantOfOneSubtopic := 2.0 + 2.0*discountOfARepeatedSubtopic/math.Log2(3)
	wantOfTwoSubtopics := 2.0 + 2.0/math.Log2(3)
	if math.Abs(gainOfOneSubtopic-wantOfOneSubtopic) > gainTolerance {
		t.Errorf(
			"two documents of one subtopic reach the gain %.6f, want %.6f",
			gainOfOneSubtopic, wantOfOneSubtopic,
		)
	}
	if math.Abs(gainOfTwoSubtopics-wantOfTwoSubtopics) > gainTolerance {
		t.Errorf(
			"two documents of two subtopics reach the gain %.6f, want %.6f",
			gainOfTwoSubtopics, wantOfTwoSubtopics,
		)
	}
}

func TestTheIdealOrderPutsTheFirstDocumentOfAnotherSubtopicFirst(t *testing.T) {
	t.Parallel()

	got := normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2), subtopic: "cat"},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2), subtopic: "cat"},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2), subtopic: "car"},
	)

	idealGain := 2.0 + 2.0/math.Log2(3) + 2.0*discountOfARepeatedSubtopic/math.Log2(4)
	gainOfTheOrder := 2.0 + 2.0*discountOfARepeatedSubtopic/math.Log2(3) + 2.0/math.Log2(4)
	want := gainOfTheOrder / idealGain
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order that repeats a subtopic before another subtopic reaches %.6f, want %.6f",
			got, want,
		)
	}
	if got >= 1 {
		t.Errorf("the order that repeats a subtopic reaches %.6f, want less than the ideal", got)
	}
}

func TestAnUngradedDocumentTakesNoPlaceInTheGain(t *testing.T) {
	t.Parallel()

	got := normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a"},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2), subtopic: "cat"},
		documentToMeasure{address: "https://two.example/a", grade: gradeOf(2), subtopic: "cat"},
	)

	want := 1.0
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order under an ungraded document reaches %.6f, want %.6f", got, want,
		)
	}
}

func TestTwoUnjudgedDocumentsOfOneHostCountInFull(t *testing.T) {
	t.Parallel()

	got := normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(t,
		documentToMeasure{address: "https://one.example/a", grade: gradeOf(2)},
		documentToMeasure{address: "https://one.example/b", grade: gradeOf(2)},
	)

	want := 1.0
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"two unjudged documents of one host reach %.6f of the ideal gain, want %.6f",
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

	idealGain := 2.0 + 1.0/math.Log2(3) + 1.0*discountOfARepeatedSubtopic/math.Log2(4)
	gainOfTheOrder := 2.0 + 1.0*discountOfARepeatedSubtopic/math.Log2(3) + 1.0/math.Log2(4)
	want := gainOfTheOrder / idealGain
	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"the order that repeats a subtopic across hosts reaches %.6f, want %.6f", got, want,
		)
	}
}

func TestDocumentsOfNoJudgedSubtopicGainAsMuchAsUnderNoDiscount(t *testing.T) {
	t.Parallel()

	documentsInOrder := []documentToMeasure{
		{address: "https://one.example/a", grade: gradeOf(2)},
		{address: "https://one.example/b", grade: gradeOf(1)},
		{address: "https://two.example/a", grade: gradeOf(2)},
	}

	got := normalizedGainDiscountedPerSubtopicOfTheDocumentsInOrder(t, documentsInOrder...)
	want := normalizedGainWithNoDiscountOfTheDocumentsInOrder(t, documentsInOrder...)

	if math.Abs(got-want) > gainTolerance {
		t.Errorf(
			"documents of no judged subtopic reach %.6f per subtopic, want the gain with no "+
				"discount %.6f",
			got, want,
		)
	}
	if got >= 1 {
		t.Errorf("the order that holds back a document reaches %.6f, want less than the ideal", got)
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
				hash:           hash,
				judgedSubtopic: document.subtopic,
			}
		}
		orderedItems = append(orderedItems, queryanswers.AnsweredItem{
			Metadata: yacymodel.URLMetadata{Hash: hash, Address: document.address},
		})
	}

	return graded, orderedItems
}
