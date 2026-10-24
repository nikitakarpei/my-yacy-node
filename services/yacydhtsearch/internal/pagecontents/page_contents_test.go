package pagecontents_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	snippetLengthCeiling = 40
	textOfThePage        = "The city of Berlin holds a wall, and Berlin holds " +
		"a river that the wall never crossed."
)

func TestTheTextGivesTheHitsOfEachQueryWordAndNoOtherWord(t *testing.T) {
	t.Parallel()

	pageContents := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{Text: textOfThePage},
		[]string{"berlin", "river", "paris"},
		snippetLengthCeiling,
	)

	if len(pageContents.HitsPerQueryWord) != 3 ||
		pageContents.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 2 ||
		pageContents.HitsPerQueryWord[yacymodel.WordHash("river")] != 1 ||
		pageContents.HitsPerQueryWord[yacymodel.WordHash("paris")] != 0 {
		t.Fatalf("the text gives %+v, want two berlin, one river, zero paris", pageContents)
	}
	if pageContents.AmountOfWords != len(yacymodel.WordsIn(textOfThePage)) {
		t.Fatalf("the text holds %d words, want every indexed word", pageContents.AmountOfWords)
	}
}

func TestTheTextGivesTheHitsOfEachQueryPhraseInEitherOrder(t *testing.T) {
	t.Parallel()

	inOrder := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{Text: textOfThePage},
		[]string{"berlin", "holds"},
		snippetLengthCeiling,
	)
	inTheOtherOrder := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{Text: textOfThePage},
		[]string{"holds", "berlin"},
		snippetLengthCeiling,
	)
	ofOneWord := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{Text: textOfThePage},
		[]string{"berlin"},
		snippetLengthCeiling,
	)

	if inOrder.QueryPhraseHits != 2 || inTheOtherOrder.QueryPhraseHits != 2 ||
		ofOneWord.QueryPhraseHits != 0 {
		t.Fatalf(
			"the text gives %d, %d and %d query phrase hits, want 2, 2 and 0",
			inOrder.QueryPhraseHits, inTheOtherOrder.QueryPhraseHits, ofOneWord.QueryPhraseHits,
		)
	}
}

func TestTheTitleGivesTheHitsOfEachQueryPhraseAndNoWordHit(t *testing.T) {
	t.Parallel()

	contents := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{
			Title: "Berlin holds",
			Text:  "The city holds a wall.",
		},
		[]string{"berlin", "holds"},
		snippetLengthCeiling,
	)

	if contents.QueryPhraseHits != 1 ||
		contents.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 0 ||
		contents.AmountOfWords != 4 {
		t.Fatalf(
			"the title gives %d query phrase hits, %d hits of berlin and %d words, want 1, 0 and 4",
			contents.QueryPhraseHits,
			contents.HitsPerQueryWord[yacymodel.WordHash("berlin")],
			contents.AmountOfWords,
		)
	}
}

func TestTheSnippetIsThePassageThatHoldsMoreOfTheQueryWords(t *testing.T) {
	t.Parallel()

	snippet := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{
			Title: "",
			Text:  "Berlin holds a wall. The wall and the river of Berlin.",
		},
		[]string{"berlin", "river"},
		snippetLengthCeiling,
	).Snippet

	if snippet != "The wall and the river of Berlin." {
		t.Fatalf("the snippet reads %q, want the passage that holds both query words", snippet)
	}
}

func TestTheTextCarriesTheTitleOfThePageWithItsSpacesCollapsed(t *testing.T) {
	t.Parallel()

	pageContents := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{Title: "Berlin |\n    the city", Text: textOfThePage},
		[]string{"berlin"},
		snippetLengthCeiling,
	)

	if pageContents.Title != "Berlin | the city" {
		t.Fatalf(
			"the text carries the title %q, want the title of the page with its spaces collapsed",
			pageContents.Title,
		)
	}
}

func TestTheAuxiliaryTextGivesHitsButNoWordsAndNoSnippet(t *testing.T) {
	t.Parallel()

	contents := pagecontents.PageContentsFrom(
		pagecontents.ExtractedPage{
			Title:         "",
			Text:          "The city holds a wall.",
			AuxiliaryText: "Berlin holds a river.",
		},
		[]string{"berlin", "holds"},
		snippetLengthCeiling,
	)

	if contents.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 1 ||
		contents.HitsPerQueryWord[yacymodel.WordHash("holds")] != 2 ||
		contents.QueryPhraseHits != 1 ||
		contents.AmountOfWords != len(yacymodel.WordsIn("The city holds a wall.")) ||
		strings.Contains(contents.Snippet, "Berlin") {
		t.Fatalf("the auxiliary text gives %+v, want hits of berlin 1, of holds 2, "+
			"1 phrase hit, the words of the text and a snippet from the text", contents)
	}
}
