package queryfindings_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestEachAssessedDocumentCarriesItsSpamVerdict(t *testing.T) {
	t.Parallel()

	findings := findingsOfDocumentsAt(
		t, "https://spam.example/", "https://clean.example/", "https://unread.example/",
	)

	assessed := findings.WithSpamVerdicts(map[yacymodel.URLHash]spamassessment.Verdict{
		documentOf(t, "https://spam.example/"):  spamassessment.Spam,
		documentOf(t, "https://clean.example/"): spamassessment.Clean,
	})

	verdicts := make([]spamassessment.Verdict, 0, len(assessed.FoundDocuments))
	for _, foundDocument := range assessed.FoundDocuments {
		verdicts = append(verdicts, foundDocument.SpamVerdict)
	}
	if len(verdicts) != 3 || verdicts[0] != spamassessment.Spam ||
		verdicts[1] != spamassessment.Clean || verdicts[2] != spamassessment.Unassessed {
		t.Fatalf("the documents carry %v, want spam, clean and unassessed", verdicts)
	}
}

func TestOnlyADocumentWhosePageIsCalledSpamIsSpam(t *testing.T) {
	t.Parallel()

	findings := findingsOfDocumentsAt(
		t, "https://spam.example/", "https://clean.example/", "https://unread.example/",
	)

	assessed := findings.WithSpamVerdicts(map[yacymodel.URLHash]spamassessment.Verdict{
		documentOf(t, "https://spam.example/"):  spamassessment.Spam,
		documentOf(t, "https://clean.example/"): spamassessment.Clean,
	})

	spam := make([]bool, 0, len(assessed.FoundDocuments))
	for _, foundDocument := range assessed.FoundDocuments {
		spam = append(spam, foundDocument.IsSpam())
	}
	if !slices.Equal(spam, []bool{true, false, false}) {
		t.Fatalf("the documents are spam %v, want only the first", spam)
	}
}
