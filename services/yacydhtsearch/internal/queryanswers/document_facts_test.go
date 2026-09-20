package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const countedWord = "berlin"

func replicaOfTheCountedWord(
	hits int,
	localLinks int,
	externalLinks int,
) queryanswers.PostingReplica {
	return queryanswers.PostingReplica{
		Holder: yacymodel.WordHash("a holder"),
		Word:   yacymodel.Some(yacymodel.WordHash(countedWord)),
		Posting: yacymodel.RWIPosting{
			Hits:          hits,
			TextWords:     1200,
			LocalLinks:    localLinks,
			ExternalLinks: externalLinks,
		},
	}
}

func factsOfTheDocumentOf(
	t *testing.T, replicas ...queryanswers.PostingReplica,
) queryanswers.DocumentFacts {
	t.Helper()

	document := documentOf(t, "https://berlin.example/")
	postingReplicasPerDocument := queryanswers.PostingReplicasPerDocument{}
	for _, replica := range replicas {
		postingReplicasPerDocument.Keep(document, replica)
	}

	return queryanswers.FactsPerDocumentOf(postingReplicasPerDocument)[document]
}

func TestThePostingOfAWordCountsItsHitsAndTheLinksOfTheDocument(t *testing.T) {
	t.Parallel()

	facts := factsOfTheDocumentOf(t, replicaOfTheCountedWord(7, 20, 9))

	if facts.HitsPerQueryWord[yacymodel.WordHash(countedWord)] != 7 ||
		facts.AmountOfLinks.OrElse(0) != 29 {
		t.Fatalf("the document reads %+v, want the 7 hits and the 29 links the peer counted", facts)
	}
}

func TestOnlyTheFirstPostingOfAWordCountsForADocument(t *testing.T) {
	t.Parallel()

	facts := factsOfTheDocumentOf(
		t, replicaOfTheCountedWord(7, 20, 9), replicaOfTheCountedWord(3, 1, 0),
	)

	if facts.HitsPerQueryWord[yacymodel.WordHash(countedWord)] != 7 ||
		facts.AmountOfLinks.OrElse(0) != 29 {
		t.Fatalf("the document reads %+v, want what the first holder counted", facts)
	}
}

func TestNoPeerCountsTheWordsOrTheQueryPhrasesOfADocument(t *testing.T) {
	t.Parallel()

	facts := factsOfTheDocumentOf(t, replicaOfTheCountedWord(7, 20, 9))

	if facts.AmountOfWords.Present() || facts.QueryPhraseHits.Present() {
		t.Fatalf(
			"the document reads %+v, want no amount of words and no query phrase hits of a peer",
			facts,
		)
	}
}

func TestThePostingOfAWordNoAskNamedCountsForNoDocument(t *testing.T) {
	t.Parallel()

	replica := replicaOfTheCountedWord(7, 20, 9)
	replica.Word = yacymodel.None[yacymodel.Hash]()

	facts := factsOfTheDocumentOf(t, replica)

	if len(facts.HitsPerQueryWord) != 0 || facts.AmountOfLinks.Present() {
		t.Fatalf("the document reads %+v, want no fact of a posting no ask named a word for", facts)
	}
}
