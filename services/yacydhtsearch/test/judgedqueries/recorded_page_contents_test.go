package judgedqueries_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var timeThePeersAnswered = time.Date(2026, time.September, 21, 0, 15, 42, 0, time.UTC)

func TestTheContentsOfTheReadPageOfADocumentSurviveTheRecording(t *testing.T) {
	t.Parallel()

	document := hashOfTheWeatherDocument(t)
	word := yacymodel.WordHash("berlin")
	read := answersOfADocumentWrittenAndReadBack(t, answersAndTheirReadPages{
		answeredQuery: queryanswers.AnsweredQuery{
			FoundDocuments: []queryanswers.FoundDocument{{
				Hash:    document,
				Address: "https://example.org/weather",
				Title:   "What a peer calls it",
				Snippet: "What a peer sent",
			}},
		},
		pageContentsPerDocument: map[yacymodel.URLHash]pagecontents.PageContents{
			document: {
				Title:            "Weather in Berlin",
				Snippet:          "Rain is falling over the whole city today.",
				HitsPerQueryWord: map[yacymodel.Hash]int{word: 9},
				QueryPhraseHits:  4,
				AmountOfWords:    1200,
				LinkCounts:       pagecontents.LinkCounts{LocalLinks: 12, ExternalLinks: 7},
			},
		},
	})

	facts := read.FactsPerDocument[document]
	if facts.HitsPerQueryWord[word] != 9 || facts.QueryPhraseHits.OrElse(0) != 4 ||
		facts.AmountOfWords.OrElse(0) != 1200 || facts.AmountOfLinks.OrElse(0) != 19 {
		t.Fatalf("the recorded document counts %+v, want what the read page counted", facts)
	}
	if read.FoundDocuments[0].Title != "Weather in Berlin" ||
		read.FoundDocuments[0].Snippet != "Rain is falling over the whole city today." {
		t.Fatalf(
			"the recorded document shows %+v, want the title and the snippet of the read page",
			read.FoundDocuments[0],
		)
	}
}

func TestADocumentOfWhichNoPageWasReadCountsTheFactsOfItsPostings(t *testing.T) {
	t.Parallel()

	document := hashOfTheWeatherDocument(t)
	word := yacymodel.WordHash("berlin")
	read := answersOfADocumentWrittenAndReadBack(t, answersAndTheirReadPages{
		answeredQuery: queryanswers.AnsweredQuery{
			FoundDocuments: []queryanswers.FoundDocument{{
				Hash:    document,
				Address: "https://example.org/weather",
				Title:   "What a peer calls it",
			}},
			PostingReplicasPerDocument: queryanswers.PostingReplicasPerDocument{
				document: {{
					Holder: yacymodel.WordHash("holder"),
					Word:   yacymodel.Some(word),
					Posting: yacymodel.RWIPosting{
						URLHash: document, Hits: 22, LocalLinks: 15, ExternalLinks: 14,
					},
				}},
			},
		},
	})

	facts := read.FactsPerDocument[document]
	if facts.HitsPerQueryWord[word] != 22 || facts.AmountOfLinks.OrElse(0) != 29 ||
		facts.AmountOfWords.Present() || facts.QueryPhraseHits.Present() {
		t.Fatalf("the recorded document counts %+v, want what the posting counted", facts)
	}
	if read.FoundDocuments[0].Title != "What a peer calls it" {
		t.Fatalf(
			"the recorded document shows the title %q, want the title the peer sent",
			read.FoundDocuments[0].Title,
		)
	}
}

func TestReadingThePagesAgainKeepsTheTimeThePeersAnswered(t *testing.T) {
	t.Parallel()

	document := hashOfTheWeatherDocument(t)
	recorded := recordedAnswersOf("berlin", answersAndTheirReadPages{
		answeredQuery: queryanswers.AnsweredQuery{
			FoundDocuments: []queryanswers.FoundDocument{{
				Hash:    document,
				Address: "https://example.org/weather",
				Title:   "What a peer calls it",
			}},
		},
	})
	recorded.RecordedAt = timeThePeersAnswered

	readAgain := recorded.withThePageContentsReadAgain(answersAndTheirReadPages{
		answeredQuery: recorded.answeredQuery(),
		pageContentsPerDocument: map[yacymodel.URLHash]pagecontents.PageContents{
			document: {Title: "Weather in Berlin", AmountOfWords: 1200},
		},
	})

	if !readAgain.RecordedAt.Equal(timeThePeersAnswered) {
		t.Fatalf("the answers read again carry the time %s, want the time %s the peers answered",
			readAgain.RecordedAt, timeThePeersAnswered)
	}
	if readAgain.FoundDocuments[0].Title != "What a peer calls it" ||
		!readAgain.FoundDocuments[0].PageContents.Present() {
		t.Fatalf("the answers read again show %+v, want the contents of the page read again",
			readAgain.FoundDocuments[0])
	}
}

func answersOfADocumentWrittenAndReadBack(
	t *testing.T, answersAndTheirPages answersAndTheirReadPages,
) queryanswers.AnsweredQuery {
	t.Helper()

	path := filepath.Join(t.TempDir(), "recorded"+recordedAnswersFileSuffix)
	writeRecordedAnswersFile(t, path, recordedAnswersOf("berlin", answersAndTheirPages))

	return recordedAnswersInTheFile(t, path).answeredQuery()
}
