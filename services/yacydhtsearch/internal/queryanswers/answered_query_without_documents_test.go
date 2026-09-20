package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answersOfTheDocumentsAt(t *testing.T, addresses ...string) queryanswers.AnsweredQuery {
	t.Helper()

	answers := queryanswers.AnsweredQuery{FactsPerDocument: queryanswers.FactsPerDocument{}}
	for _, address := range addresses {
		document := documentOf(t, address)
		answers.FoundDocuments = append(
			answers.FoundDocuments, queryanswers.FoundDocument{Hash: document, Address: address},
		)
		answers.FactsPerDocument[document] = queryanswers.DocumentFacts{
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 1},
		}
	}

	return answers
}

func TestTheAnswersWithoutADocumentKeepEveryOtherDocumentInOrder(t *testing.T) {
	t.Parallel()

	answers := answersOfTheDocumentsAt(
		t, "https://first.example/", "https://gone.example/", "https://third.example/",
	)

	left := answers.WithoutDocuments(
		map[yacymodel.URLHash]struct{}{documentOf(t, "https://gone.example/"): {}},
	)

	if len(left.FoundDocuments) != 2 ||
		left.FoundDocuments[0].Address != "https://first.example/" ||
		left.FoundDocuments[1].Address != "https://third.example/" {
		t.Fatalf("the answers hold %+v, want the first and the third document", left.FoundDocuments)
	}
}

func TestTheAnswersWithoutADocumentCountNoFactOfIt(t *testing.T) {
	t.Parallel()

	answers := answersOfTheDocumentsAt(t, "https://first.example/", "https://gone.example/")

	left := answers.WithoutDocuments(
		map[yacymodel.URLHash]struct{}{documentOf(t, "https://gone.example/"): {}},
	)

	if _, counted := left.FactsPerDocument[documentOf(t, "https://gone.example/")]; counted {
		t.Fatalf(
			"the answers count the facts %+v of the document that left them",
			left.FactsPerDocument,
		)
	}
}
