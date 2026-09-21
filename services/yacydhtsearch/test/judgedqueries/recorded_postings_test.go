package judgedqueries_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answersWrittenAndReadBack(
	t *testing.T, answers queryanswers.AnsweredQuery,
) queryanswers.AnsweredQuery {
	t.Helper()

	return answersOfADocumentWrittenAndReadBack(t, answersAndTheirReadPages{answeredQuery: answers})
}

func answersOfOneDocumentHolding(
	t *testing.T,
	replicas []queryanswers.PostingReplica,
	metadata yacymodel.URLMetadata,
) queryanswers.AnsweredQuery {
	t.Helper()

	hash, err := yacymodel.URLHashOf(metadata.Address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", metadata.Address, err)
	}
	metadata.Hash = hash

	return queryanswers.AnsweredQuery{
		FoundDocuments:             []queryanswers.FoundDocument{{Hash: hash}},
		PostingReplicasPerDocument: queryanswers.PostingReplicasPerDocument{hash: replicas},
		MetadataPerDocument:        queryanswers.MetadataPerDocument{hash: metadata},
	}
}

func hashOfTheWeatherDocument(t *testing.T) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf("https://example.org/weather")
	if err != nil {
		t.Fatalf("URLHashOf: %v", err)
	}

	return hash
}

func postingOfTheDocument(
	document yacymodel.URLHash, posting yacymodel.RWIPosting,
) yacymodel.RWIPosting {
	posting.URLHash = document

	return posting
}

func TestThePostingOfEveryHolderSurvivesTheRecording(t *testing.T) {
	t.Parallel()

	metadata := yacymodel.URLMetadata{Address: "https://example.org/weather"}
	firstHolder, secondHolder := yacymodel.WordHash("first"), yacymodel.WordHash("second")
	answers := answersOfOneDocumentHolding(t, []queryanswers.PostingReplica{
		{
			Holder: firstHolder,
			Word:   yacymodel.Some(yacymodel.WordHash("berlin")),
			Posting: postingOfTheDocument(
				hashOfTheWeatherDocument(t),
				yacymodel.RWIPosting{Hits: 7, TextWords: 1200, TitleWords: 4, Phrases: 90},
			),
		},
		{
			Holder: secondHolder,
			Posting: postingOfTheDocument(
				hashOfTheWeatherDocument(t), yacymodel.RWIPosting{Hits: 3, TextWords: 800},
			),
		},
	}, metadata)

	read := answersWrittenAndReadBack(t, answers).
		PostingReplicasPerDocument[answers.FoundDocuments[0].Hash]

	if len(read) != 2 || !read[0].Posting.WordHash.IsZero() ||
		read[0].Holder != firstHolder || read[0].Posting.TitleWords != 4 ||
		read[0].Posting.Phrases != 90 || read[0].Word.OrElse(yacymodel.Hash{}) !=
		yacymodel.WordHash("berlin") ||
		read[1].Holder != secondHolder || read[1].Word.Present() {
		t.Fatalf("the recorded document holds the postings %+v, want both holders in order", read)
	}
}

func TestTheMetadataAPeerReportedSurvivesTheRecording(t *testing.T) {
	t.Parallel()

	german, err := yacymodel.ParseLanguage("de")
	if err != nil {
		t.Fatalf("ParseLanguage(de): %v", err)
	}
	answers := answersOfOneDocumentHolding(t, nil, yacymodel.URLMetadata{
		Address:       "https://example.org/wetter",
		Author:        "A writer",
		Tags:          []string{"weather", "berlin"},
		Language:      yacymodel.Some(german),
		WordCount:     1200,
		ImageLinks:    5,
		LocalLinks:    12,
		ExternalLinks: 7,
	})

	read := answersWrittenAndReadBack(t, answers).
		MetadataPerDocument[answers.FoundDocuments[0].Hash]

	if read.Author != "A writer" || len(read.Tags) != 2 || read.WordCount != 1200 ||
		read.ImageLinks != 5 || read.Language.OrElse(yacymodel.Language{}) != german {
		t.Fatalf("the recorded document holds the metadata %+v, want what the peer reported", read)
	}
}
