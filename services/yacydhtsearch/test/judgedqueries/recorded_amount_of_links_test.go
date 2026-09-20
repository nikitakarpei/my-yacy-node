package judgedqueries_test

import (
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheAmountOfLinksOfADocumentSurvivesTheRecording(t *testing.T) {
	t.Parallel()

	read := recordedAnswersWrittenAndReadBack(t, queryanswers.DocumentFacts{
		AmountOfLinks: yacymodel.Some(19),
	})

	amountOfLinks, recorded := read.AmountOfLinks.Get()
	if !recorded || amountOfLinks != 19 {
		t.Fatalf(
			"the recorded document holds the amount of links %d recorded %t, want 19",
			amountOfLinks,
			recorded,
		)
	}
}

func TestADocumentRecordedWithoutAnAmountOfLinksIsReadBackWithoutOne(t *testing.T) {
	t.Parallel()

	read := recordedAnswersWrittenAndReadBack(t, queryanswers.DocumentFacts{})

	if read.AmountOfLinks.Present() {
		t.Fatal("the recorded document holds an amount of links, want none where none was recorded")
	}
}

func recordedAnswersWrittenAndReadBack(
	t *testing.T, facts queryanswers.DocumentFacts,
) queryanswers.DocumentFacts {
	t.Helper()

	address := "https://example.org/weather"
	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("hash %s: %v", address, err)
	}

	path := filepath.Join(t.TempDir(), "recorded"+recordedAnswersFileSuffix)
	writeRecordedAnswersFile(t, path, recordedAnswersOf("berlin", queryanswers.AnsweredQuery{
		FoundDocuments:   []queryanswers.FoundDocument{{Hash: hash, Address: address}},
		FactsPerDocument: queryanswers.FactsPerDocument{hash: facts},
	}))

	return recordedAnswersInTheFile(t, path).answeredQuery().FactsPerDocument[hash]
}
