package queryfindings_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func findingsOfDocumentsAt(t *testing.T, addresses ...string) queryfindings.Findings {
	t.Helper()

	findings := queryfindings.Findings{}
	for _, address := range addresses {
		findings.FoundDocuments = append(findings.FoundDocuments, queryfindings.FoundDocumentOf(
			documentOf(t, address),
			[]queryfindings.MetadataReplica{{Metadata: yacymodel.URLMetadata{Address: address}}},
			[]queryfindings.PostingReplica{replicaOfCountedWord(1, 1, 0)},
		))
	}

	return findings
}

func TestFindingsWithoutDocumentKeepEveryOtherDocumentInOrder(t *testing.T) {
	t.Parallel()

	findings := findingsOfDocumentsAt(
		t, "https://first.example/", "https://gone.example/", "https://third.example/",
	)

	left := findings.WithoutDocuments(
		map[yacymodel.URLHash]struct{}{documentOf(t, "https://gone.example/"): {}},
	)

	if len(left.FoundDocuments) != 2 ||
		left.FoundDocuments[0].Address != "https://first.example/" ||
		left.FoundDocuments[1].Address != "https://third.example/" {
		t.Fatalf(
			"the findings hold %+v, want the first and the third document",
			left.FoundDocuments,
		)
	}
}

func TestFindingsWithoutEveryDocumentNamedHoldNone(t *testing.T) {
	t.Parallel()

	findings := findingsOfDocumentsAt(t, "https://first.example/", "https://gone.example/")

	left := findings.WithoutDocuments(map[yacymodel.URLHash]struct{}{
		documentOf(t, "https://first.example/"): {},
		documentOf(t, "https://gone.example/"):  {},
	})

	if len(left.FoundDocuments) != 0 {
		t.Fatalf("the findings hold %+v, want no document", left.FoundDocuments)
	}
}
