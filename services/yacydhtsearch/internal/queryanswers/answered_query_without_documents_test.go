package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheAnswersWithoutADocumentKeepEveryOtherDocumentInOrder(t *testing.T) {
	t.Parallel()

	answers := queryanswers.AnsweredQuery{
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentWithOneHitOf(t, "https://first.example/", "berlin"),
			foundDocumentWithOneHitOf(t, "https://gone.example/", "berlin"),
			foundDocumentWithOneHitOf(t, "https://third.example/", "wall"),
		},
	}

	left := answers.WithoutDocuments(
		map[yacymodel.URLHash]struct{}{documentOf(t, "https://gone.example/"): {}},
	)

	if len(left.FoundDocuments) != 2 ||
		left.FoundDocuments[0].Address != "https://first.example/" ||
		left.FoundDocuments[1].Address != "https://third.example/" {
		t.Fatalf("the answers hold %+v, want the first and the third document", left.FoundDocuments)
	}
}
