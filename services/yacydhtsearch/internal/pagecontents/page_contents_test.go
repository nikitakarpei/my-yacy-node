package pagecontents_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	snippetLengthCeiling      = 40
	shortSnippetLengthCeiling = 20
	textOfThePage             = "The city of Berlin holds a wall, and Berlin holds " +
		"a river that the wall never crossed."
)

func wordsOf(spelledWords ...string) []yacymodel.Hash {
	words := make([]yacymodel.Hash, 0, len(spelledWords))
	for _, spelledWord := range spelledWords {
		words = append(words, yacymodel.WordHash(spelledWord))
	}

	return words
}

func TestTheTextGivesTheHitsOfEachQueryWordAndNoOtherWord(t *testing.T) {
	t.Parallel()

	pageContents := pagecontents.PageContentsFrom(
		"",
		textOfThePage,
		pagecontents.LinkCounts{},
		wordsOf("berlin", "river", "paris"),
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
		"",
		textOfThePage,
		pagecontents.LinkCounts{},
		wordsOf("berlin", "holds"),
		snippetLengthCeiling,
	)
	inTheOtherOrder := pagecontents.PageContentsFrom(
		"",
		textOfThePage,
		pagecontents.LinkCounts{},
		wordsOf("holds", "berlin"),
		snippetLengthCeiling,
	)
	ofOneWord := pagecontents.PageContentsFrom(
		"", textOfThePage, pagecontents.LinkCounts{}, wordsOf("berlin"), snippetLengthCeiling,
	)

	if inOrder.QueryPhraseHits != 2 || inTheOtherOrder.QueryPhraseHits != 2 ||
		ofOneWord.QueryPhraseHits != 0 {
		t.Fatalf(
			"the text gives %d, %d and %d query phrase hits, want 2, 2 and 0",
			inOrder.QueryPhraseHits, inTheOtherOrder.QueryPhraseHits, ofOneWord.QueryPhraseHits,
		)
	}
}

func TestTheSnippetIsThePassageThatHoldsMoreOfTheQueryWords(t *testing.T) {
	t.Parallel()

	snippet := pagecontents.PageContentsFrom(
		"", "Berlin holds a wall. The wall and the river of Berlin.",
		pagecontents.LinkCounts{}, wordsOf("berlin", "river"), snippetLengthCeiling,
	).Snippet

	if snippet != "The wall and the river of Berlin." {
		t.Fatalf("the snippet reads %q, want the passage that holds both query words", snippet)
	}
}

func TestTheSnippetIsThePassageThatHoldsTheQueryPhraseWhenBothHoldTheSameWords(t *testing.T) {
	t.Parallel()

	snippet := pagecontents.PageContentsFrom(
		"", "The wall stands near Berlin. The Berlin wall is famous.",
		pagecontents.LinkCounts{}, wordsOf("berlin", "wall"), snippetLengthCeiling,
	).Snippet

	if snippet != "The Berlin wall is famous." {
		t.Fatalf("the snippet reads %q, want the passage that holds the query phrase", snippet)
	}
}

func TestTheSnippetIsTheEarlierPassageWhenTwoPassagesAnswerTheQueryTheSame(t *testing.T) {
	t.Parallel()

	snippet := pagecontents.PageContentsFrom(
		"",
		"Berlin is old. Berlin is new.",
		pagecontents.LinkCounts{},
		wordsOf("berlin"),
		shortSnippetLengthCeiling,
	).Snippet

	if snippet != "Berlin is old." {
		t.Fatalf("the snippet reads %q, want the earlier of two equal passages", snippet)
	}
}

func TestTheSnippetJoinsShortSentencesUpToTheLengthCeiling(t *testing.T) {
	t.Parallel()

	snippet := pagecontents.PageContentsFrom(
		"", "One. Two. Three. This other sentence holds no query word at all.",
		pagecontents.LinkCounts{}, wordsOf("three"), snippetLengthCeiling,
	).Snippet

	if snippet != "One. Two. Three." {
		t.Fatalf("the snippet reads %q, want the short sentences shown together", snippet)
	}
}

func TestTheSnippetIsCutAtAWordBoundaryBeforeTheLengthCeiling(t *testing.T) {
	t.Parallel()

	snippet := pagecontents.PageContentsFrom(
		"", textOfThePage, pagecontents.LinkCounts{}, wordsOf("city"), snippetLengthCeiling,
	).Snippet

	if len([]rune(snippet)) > snippetLengthCeiling ||
		!strings.Contains(textOfThePage, snippet+" ") {
		t.Fatalf(
			"the snippet reads %q, want at most %d letters ending at a word boundary",
			snippet, snippetLengthCeiling,
		)
	}
}

func TestTheSnippetOfATextWithoutAQueryWordIsItsFirstPassage(t *testing.T) {
	t.Parallel()

	snippet := pagecontents.PageContentsFrom(
		"",
		"Über kurz oder lang. Und dann.",
		pagecontents.LinkCounts{},
		wordsOf("paris"),
		shortSnippetLengthCeiling,
	).Snippet

	if snippet != "Über kurz oder lang." {
		t.Fatalf("the snippet reads %q, want the first passage of the text", snippet)
	}
}

func TestTheTextCarriesTheTitleOfThePageWithItsSpacesCollapsed(t *testing.T) {
	t.Parallel()

	pageContents := pagecontents.PageContentsFrom(
		"Berlin |\n    the city",
		textOfThePage,
		pagecontents.LinkCounts{},
		wordsOf("berlin"),
		snippetLengthCeiling,
	)

	if pageContents.Title != "Berlin | the city" {
		t.Fatalf(
			"the text carries the title %q, want the title of the page with its spaces collapsed",
			pageContents.Title,
		)
	}
}
