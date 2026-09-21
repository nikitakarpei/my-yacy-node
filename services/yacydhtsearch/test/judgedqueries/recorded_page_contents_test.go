package judgedqueries_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var timeThePeersAnswered = time.Date(2026, time.September, 21, 0, 15, 42, 0, time.UTC)

func TestTheContentsOfTheReadPageOfADocumentSurviveTheRecording(t *testing.T) {
	t.Parallel()

	document := hashOfWeatherDocument(t)
	word := yacymodel.WordHash("berlin")
	readAnswers := answersWrittenAndReadBack(t, answersAndPageContentsOf(
		queryanswers.AnsweredQuery{
			FoundDocuments: []queryanswers.FoundDocument{queryanswers.FoundDocumentOf(
				document,
				[]queryanswers.MetadataReplica{{Metadata: yacymodel.URLMetadata{
					Hash:    document,
					Address: weatherDocumentAddress,
					Title:   "What a peer calls it",
					Snippet: "What a peer sent",
				}}},
				nil,
			)},
		},
		map[yacymodel.URLHash]pagecontents.PageContents{
			document: {
				Title:            "Weather in Berlin",
				Snippet:          "Rain is falling over the whole city today.",
				HitsPerQueryWord: map[yacymodel.Hash]int{word: 9},
				QueryPhraseHits:  4,
				AmountOfWords:    1200,
				LinkCounts:       pagecontents.LinkCounts{LocalLinks: 12, ExternalLinks: 7},
			},
		},
	))

	facts := readAnswers.FoundDocuments[0].Facts
	if facts.HitsPerQueryWord[word] != 9 || facts.QueryPhraseHits.OrElse(0) != 4 ||
		facts.AmountOfWords.OrElse(0) != 1200 || facts.AmountOfLinks.OrElse(0) != 19 {
		t.Fatalf("the recorded document counts %+v, want what the read page counted", facts)
	}
	if readAnswers.FoundDocuments[0].Title != "Weather in Berlin" ||
		readAnswers.FoundDocuments[0].Snippet != "Rain is falling over the whole city today." {
		t.Fatalf(
			"the recorded document shows %+v, want the title and the snippet of the read page",
			readAnswers.FoundDocuments[0],
		)
	}
}

func TestADocumentOfWhichNoPageWasReadCountsTheFactsOfItsPostings(t *testing.T) {
	t.Parallel()

	document := hashOfWeatherDocument(t)
	word := yacymodel.WordHash("berlin")
	readAnswers := answersWrittenAndReadBack(t, answersAndPageContentsOf(
		queryanswers.AnsweredQuery{
			FoundDocuments: []queryanswers.FoundDocument{queryanswers.FoundDocumentOf(
				document,
				[]queryanswers.MetadataReplica{{Metadata: yacymodel.URLMetadata{
					Hash:    document,
					Address: weatherDocumentAddress,
					Title:   "What a peer calls it",
				}}},
				[]queryanswers.PostingReplica{{
					Holder: yacymodel.WordHash("holder"),
					Word:   yacymodel.Some(word),
					Posting: yacymodel.RWIPosting{
						URLHash: document, Hits: 22, LocalLinks: 15, ExternalLinks: 14,
					},
				}},
			)},
		},
		nil,
	))

	facts := readAnswers.FoundDocuments[0].Facts
	if facts.HitsPerQueryWord[word] != 22 || facts.AmountOfLinks.OrElse(0) != 29 ||
		facts.AmountOfWords.Present() || facts.QueryPhraseHits.Present() {
		t.Fatalf("the recorded document counts %+v, want what the posting counted", facts)
	}
	if readAnswers.FoundDocuments[0].Title != "What a peer calls it" {
		t.Fatalf(
			"the recorded document shows the title %q, want the title the peer sent",
			readAnswers.FoundDocuments[0].Title,
		)
	}
}

func TestReadingThePagesAgainKeepsTheTimeThePeersAnswered(t *testing.T) {
	t.Parallel()

	document := hashOfWeatherDocument(t)
	recorded := recordedAnswersOf("berlin", answersAndPageContentsOf(
		queryanswers.AnsweredQuery{
			FoundDocuments: []queryanswers.FoundDocument{queryanswers.FoundDocumentOf(
				document,
				[]queryanswers.MetadataReplica{{Metadata: yacymodel.URLMetadata{
					Hash:    document,
					Address: weatherDocumentAddress,
					Title:   "What a peer calls it",
				}}},
				nil,
			)},
		},
		nil,
	))
	recorded.RecordedAt = timeThePeersAnswered

	readAgain := recorded.withPageContentsReadAgain(answersAndPageContentsOf(
		recorded.answers(),
		map[yacymodel.URLHash]pagecontents.PageContents{
			document: {Title: "Weather in Berlin", AmountOfWords: 1200},
		},
	))

	if !readAgain.RecordedAt.Equal(timeThePeersAnswered) {
		t.Fatalf("the answers read again carry the time %s, want the time %s the peers answered",
			readAgain.RecordedAt, timeThePeersAnswered)
	}
	metadataReadAgain, reported := readAgain.FoundDocuments[0].Metadata.Get()
	if !reported || metadataReadAgain.Title != "What a peer calls it" ||
		!readAgain.FoundDocuments[0].PageContents.Present() {
		t.Fatalf("the answers read again show %+v, want the contents of the page read again",
			readAgain.FoundDocuments[0])
	}
}

type recordedPageContents struct {
	Address          string                 `json:"address,omitempty"`
	Title            string                 `json:"title"`
	Snippet          string                 `json:"snippet"`
	HitsPerQueryWord map[yacymodel.Hash]int `json:"hitsPerQueryWord"`
	QueryPhraseHits  int                    `json:"queryPhraseHits"`
	AmountOfWords    int                    `json:"amountOfWords"`
	LocalLinks       int                    `json:"localLinks"`
	ExternalLinks    int                    `json:"externalLinks"`
}

func recordedPageContentsFor(
	document yacymodel.URLHash,
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) yacymodel.Optional[recordedPageContents] {
	pageContents, read := pageContentsPerDocument[document]
	if !read {
		return yacymodel.None[recordedPageContents]()
	}

	return yacymodel.Some(recordedPageContentsOf(pageContents))
}

func recordedPageContentsOf(pageContents pagecontents.PageContents) recordedPageContents {
	return recordedPageContents{
		Address:          pageContents.Address,
		Title:            pageContents.Title,
		Snippet:          pageContents.Snippet,
		HitsPerQueryWord: pageContents.HitsPerQueryWord,
		QueryPhraseHits:  pageContents.QueryPhraseHits,
		AmountOfWords:    pageContents.AmountOfWords,
		LocalLinks:       pageContents.LinkCounts.LocalLinks,
		ExternalLinks:    pageContents.LinkCounts.ExternalLinks,
	}
}

func (recorded recordedPageContents) pageContents() pagecontents.PageContents {
	return pagecontents.PageContents{
		Address:          recorded.Address,
		Title:            recorded.Title,
		Snippet:          recorded.Snippet,
		HitsPerQueryWord: recorded.HitsPerQueryWord,
		QueryPhraseHits:  recorded.QueryPhraseHits,
		AmountOfWords:    recorded.AmountOfWords,
		LinkCounts: pagecontents.LinkCounts{
			LocalLinks:    recorded.LocalLinks,
			ExternalLinks: recorded.ExternalLinks,
		},
	}
}
