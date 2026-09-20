package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func postingOf(
	hits int,
	localLinks int,
	externalLinks int,
) yacymodel.Optional[yacymodel.RWIPosting] {
	return yacymodel.Some(yacymodel.RWIPosting{
		Hits:          hits,
		TextWords:     1200,
		LocalLinks:    localLinks,
		ExternalLinks: externalLinks,
	})
}

func TestThePostingOfAWordCountsItsHitsAndTheLinksOfTheDocument(t *testing.T) {
	t.Parallel()

	factsPerDocument := queryanswers.FactsPerDocument{}
	document := documentOf(t, "https://berlin.example/")

	factsPerDocument.KeepTheFirstPostingOfTheWord(
		document, yacymodel.WordHash("berlin"), postingOf(7, 20, 9),
	)

	facts := factsPerDocument[document]
	if facts.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 7 ||
		facts.AmountOfLinks.OrElse(0) != 29 {
		t.Fatalf("the document reads %+v, want the 7 hits and the 29 links the peer counted", facts)
	}
}

func TestOnlyTheFirstPostingOfAWordCountsForADocument(t *testing.T) {
	t.Parallel()

	factsPerDocument := queryanswers.FactsPerDocument{}
	document := documentOf(t, "https://berlin.example/")

	factsPerDocument.KeepTheFirstPostingOfTheWord(
		document, yacymodel.WordHash("berlin"), postingOf(7, 20, 9),
	)
	factsPerDocument.KeepTheFirstPostingOfTheWord(
		document, yacymodel.WordHash("berlin"), postingOf(3, 1, 0),
	)

	facts := factsPerDocument[document]
	if facts.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 7 ||
		facts.AmountOfLinks.OrElse(0) != 29 {
		t.Fatalf("the document reads %+v, want what the first peer counted", facts)
	}
}

func TestNoPeerCountsTheWordsOrTheQueryPhrasesOfADocument(t *testing.T) {
	t.Parallel()

	factsPerDocument := queryanswers.FactsPerDocument{}
	document := documentOf(t, "https://berlin.example/")

	factsPerDocument.KeepTheFirstPostingOfTheWord(
		document, yacymodel.WordHash("berlin"), postingOf(7, 20, 9),
	)

	facts := factsPerDocument[document]
	if facts.AmountOfWords.Present() || facts.QueryPhraseHits.Present() {
		t.Fatalf(
			"the document reads %+v, want no amount of words and no query phrase hits of a peer",
			facts,
		)
	}
}

func TestAWordNoPeerSentAPostingForCountsForNoDocument(t *testing.T) {
	t.Parallel()

	factsPerDocument := queryanswers.FactsPerDocument{}
	document := documentOf(t, "https://berlin.example/")

	factsPerDocument.KeepTheFirstPostingOfTheWord(
		document, yacymodel.WordHash("berlin"), yacymodel.None[yacymodel.RWIPosting](),
	)

	if _, counted := factsPerDocument[document]; counted {
		t.Fatalf("the answers count the facts %+v of a document no peer sent a posting for",
			factsPerDocument)
	}
}
