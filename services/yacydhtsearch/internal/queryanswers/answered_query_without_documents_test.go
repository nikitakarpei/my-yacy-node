package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answersOfDocumentsAt(t *testing.T, addresses ...string) queryanswers.AnsweredQuery {
	t.Helper()

	answers := queryanswers.AnsweredQuery{}
	for _, address := range addresses {
		answers.FoundDocuments = append(answers.FoundDocuments, queryanswers.FoundDocumentOf(
			documentOf(t, address),
			[]queryanswers.MetadataReplica{{Metadata: yacymodel.URLMetadata{Address: address}}},
			[]queryanswers.PostingReplica{replicaOfCountedWord(1, 1, 0)},
		))
	}

	return answers
}

func TestAnswersWithoutDocumentKeepEveryOtherDocumentInOrder(t *testing.T) {
	t.Parallel()

	answers := answersOfDocumentsAt(
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

func TestAnswersWithoutEveryDocumentNamedHoldNone(t *testing.T) {
	t.Parallel()

	answers := answersOfDocumentsAt(t, "https://first.example/", "https://gone.example/")

	left := answers.WithoutDocuments(map[yacymodel.URLHash]struct{}{
		documentOf(t, "https://first.example/"): {},
		documentOf(t, "https://gone.example/"):  {},
	})

	if len(left.FoundDocuments) != 0 {
		t.Fatalf("the answers hold %+v, want no document", left.FoundDocuments)
	}
}
