package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestEachAssessedDocumentCarriesItsSpamVerdict(t *testing.T) {
	t.Parallel()

	answers := answersOfDocumentsAt(
		t, "https://spam.example/", "https://clean.example/", "https://unread.example/",
	)

	assessed := answers.WithSpamVerdicts(map[yacymodel.URLHash]spamassessment.Verdict{
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
