package judgedqueries_test

import (
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheLinkCountsOfADocumentSurviveTheRecording(t *testing.T) {
	t.Parallel()

	read := recordedAnswersWrittenAndReadBack(t, queryanswers.FoundDocument{
		Address:    "https://example.org/weather",
		LinkCounts: yacymodel.Some(queryanswers.LinkCounts{LocalLinks: 12, ExternalLinks: 7}),
	})

	linkCounts, recorded := read.FoundDocuments[0].LinkCounts.Get()
	if !recorded || linkCounts.LocalLinks != 12 || linkCounts.ExternalLinks != 7 {
		t.Fatalf(
			"the recorded document holds the link counts %+v recorded %t, want 12 local and 7 "+
				"external",
			linkCounts,
			recorded,
		)
	}
}

func TestADocumentRecordedWithoutLinkCountsIsReadBackWithoutThem(t *testing.T) {
	t.Parallel()

	read := recordedAnswersWrittenAndReadBack(t, queryanswers.FoundDocument{
		Address: "https://example.org/weather",
	})

	if read.FoundDocuments[0].LinkCounts.Present() {
		t.Fatal("the recorded document holds link counts, want none where none were recorded")
	}
}

func recordedAnswersWrittenAndReadBack(
	t *testing.T, foundDocument queryanswers.FoundDocument,
) queryanswers.AnsweredQuery {
	t.Helper()

	hash, err := yacymodel.URLHashOf(foundDocument.Address)
	if err != nil {
		t.Fatalf("hash %s: %v", foundDocument.Address, err)
	}
	foundDocument.Hash = hash

	path := filepath.Join(t.TempDir(), "recorded"+recordedAnswersFileSuffix)
	writeRecordedAnswersFile(t, path, recordedAnswersOf("berlin", queryanswers.AnsweredQuery{
		FoundDocuments: []queryanswers.FoundDocument{foundDocument},
	}))

	return recordedAnswersInTheFile(t, path).answeredQuery()
}
