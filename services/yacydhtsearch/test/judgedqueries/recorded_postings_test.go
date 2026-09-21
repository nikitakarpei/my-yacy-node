package judgedqueries_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const weatherDocumentAddress = "https://example.org/weather"

func TestThePostingOfEveryHolderSurvivesTheRecording(t *testing.T) {
	t.Parallel()

	metadata := yacymodel.URLMetadata{Address: weatherDocumentAddress}
	firstHolder, secondHolder := yacymodel.WordHash("first"), yacymodel.WordHash("second")
	answers := answersOfOneDocumentHolding(t, []queryanswers.PostingReplica{
		{
			Holder: firstHolder,
			Word:   yacymodel.Some(yacymodel.WordHash("berlin")),
			Posting: postingOfDocument(
				hashOfWeatherDocument(t),
				yacymodel.RWIPosting{Hits: 7, TextWords: 1200, TitleWords: 4, Phrases: 90},
			),
		},
		{
			Holder: secondHolder,
			Posting: postingOfDocument(
				hashOfWeatherDocument(t), yacymodel.RWIPosting{Hits: 3, TextWords: 800},
			),
		},
	}, metadata)

	readReplicas := answersWrittenAndReadBack(
		t, answersAndPageContentsOf(answers, nil),
	).FoundDocuments[0].PostingReplicas

	if len(readReplicas) != 2 || !readReplicas[0].Posting.WordHash.IsZero() ||
		readReplicas[0].Holder != firstHolder || readReplicas[0].Posting.TitleWords != 4 ||
		readReplicas[0].Posting.Phrases != 90 || readReplicas[0].Word.OrElse(yacymodel.Hash{}) !=
		yacymodel.WordHash("berlin") ||
		readReplicas[1].Holder != secondHolder || readReplicas[1].Word.Present() {
		t.Fatalf(
			"the recorded document holds the postings %+v, want both holders in order",
			readReplicas,
		)
	}
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
		FoundDocuments: []queryanswers.FoundDocument{queryanswers.FoundDocumentOf(
			hash, []queryanswers.MetadataReplica{{Metadata: metadata}}, replicas,
		)},
	}
}

func postingOfDocument(
	document yacymodel.URLHash, posting yacymodel.RWIPosting,
) yacymodel.RWIPosting {
	posting.URLHash = document

	return posting
}

func hashOfWeatherDocument(t *testing.T) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(weatherDocumentAddress)
	if err != nil {
		t.Fatalf("URLHashOf: %v", err)
	}

	return hash
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

	readReplicas := answersWrittenAndReadBack(
		t, answersAndPageContentsOf(answers, nil),
	).FoundDocuments[0].MetadataReplicas

	if len(readReplicas) != 1 || readReplicas[0].Metadata.Author != "A writer" ||
		len(readReplicas[0].Metadata.Tags) != 2 || readReplicas[0].Metadata.WordCount != 1200 ||
		readReplicas[0].Metadata.ImageLinks != 5 ||
		readReplicas[0].Metadata.Language.OrElse(yacymodel.Language{}) != german {
		t.Fatalf(
			"the recorded document holds the metadata %+v, want what the peer reported",
			readReplicas,
		)
	}
}

type recordedPostingReplica struct {
	Holder  yacymodel.Hash                     `json:"holder"`
	Word    yacymodel.Optional[yacymodel.Hash] `json:"word,omitempty"`
	Posting recordedPosting                    `json:"posting"`
}

type recordedPosting struct {
	yacymodel.RWIPosting

	WordHash *yacymodel.Hash `json:"WordHash,omitempty"`
}

func recordedPostingsOf(
	foundDocument queryanswers.FoundDocument,
) []recordedPostingReplica {
	postings := make([]recordedPostingReplica, 0, len(foundDocument.PostingReplicas))
	for _, replica := range foundDocument.PostingReplicas {
		postings = append(postings, recordedPostingReplica{
			Holder:  replica.Holder,
			Word:    replica.Word,
			Posting: recordedPostingOf(replica.Posting),
		})
	}

	return postings
}

func recordedPostingOf(posting yacymodel.RWIPosting) recordedPosting {
	recorded := recordedPosting{RWIPosting: posting}
	if !posting.WordHash.IsZero() {
		wordHash := posting.WordHash
		recorded.WordHash = &wordHash
	}
	recorded.RWIPosting.WordHash = yacymodel.Hash{}

	return recorded
}

func (recorded recordedPosting) posting() yacymodel.RWIPosting {
	posting := recorded.RWIPosting
	if recorded.WordHash != nil {
		posting.WordHash = *recorded.WordHash
	}

	return posting
}
