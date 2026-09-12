package documenttext_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
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

	documentText := documenttext.DocumentTextFrom(
		textOfThePage, wordsOf("berlin", "river", "paris"), snippetLengthCeiling,
	)

	if len(documentText.HitsPerQueryWord) != 3 ||
		documentText.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 2 ||
		documentText.HitsPerQueryWord[yacymodel.WordHash("river")] != 1 ||
		documentText.HitsPerQueryWord[yacymodel.WordHash("paris")] != 0 {
		t.Fatalf("the text gives %+v, want two berlin, one river, zero paris", documentText)
	}
	if documentText.AmountOfWords != len(yacymodel.WordsIn(textOfThePage)) {
		t.Fatalf("the text holds %d words, want every indexed word", documentText.AmountOfWords)
	}
}

func TestTheTextGivesTheHitsOfEachQueryPhraseInTheOrderOfTheQuery(t *testing.T) {
	t.Parallel()

	inOrder := documenttext.DocumentTextFrom(
		textOfThePage, wordsOf("berlin", "holds"), snippetLengthCeiling,
	)
	inTheOtherOrder := documenttext.DocumentTextFrom(
		textOfThePage, wordsOf("holds", "berlin"), snippetLengthCeiling,
	)
	ofOneWord := documenttext.DocumentTextFrom(
		textOfThePage, wordsOf("berlin"), snippetLengthCeiling,
	)

	if inOrder.QueryPhraseHits != 2 || inTheOtherOrder.QueryPhraseHits != 0 ||
		ofOneWord.QueryPhraseHits != 0 {
		t.Fatalf(
			"the text gives %d, %d and %d query phrase hits, want 2, 0 and 0",
			inOrder.QueryPhraseHits, inTheOtherOrder.QueryPhraseHits, ofOneWord.QueryPhraseHits,
		)
	}
}

func TestTheSnippetIsThePassageThatHoldsMoreOfTheQueryWords(t *testing.T) {
	t.Parallel()

	snippet := documenttext.DocumentTextFrom(
		"Berlin holds a wall. The wall and the river of Berlin.",
		wordsOf("berlin", "river"),
		snippetLengthCeiling,
	).Snippet

	if snippet != "The wall and the river of Berlin." {
		t.Fatalf("the snippet reads %q, want the passage that holds both query words", snippet)
	}
}

func TestTheSnippetIsThePassageThatHoldsTheQueryPhraseWhenBothHoldTheSameWords(t *testing.T) {
	t.Parallel()

	snippet := documenttext.DocumentTextFrom(
		"The wall stands near Berlin. The Berlin wall is famous.",
		wordsOf("berlin", "wall"),
		snippetLengthCeiling,
	).Snippet

	if snippet != "The Berlin wall is famous." {
		t.Fatalf("the snippet reads %q, want the passage that holds the query phrase", snippet)
	}
}

func TestTheSnippetIsTheEarlierPassageWhenTwoPassagesAnswerTheQueryTheSame(t *testing.T) {
	t.Parallel()

	snippet := documenttext.DocumentTextFrom(
		"Berlin is old. Berlin is new.", wordsOf("berlin"), shortSnippetLengthCeiling,
	).Snippet

	if snippet != "Berlin is old." {
		t.Fatalf("the snippet reads %q, want the earlier of two equal passages", snippet)
	}
}

func TestTheSnippetJoinsShortSentencesUpToTheLengthCeiling(t *testing.T) {
	t.Parallel()

	snippet := documenttext.DocumentTextFrom(
		"One. Two. Three. This other sentence holds no query word at all.",
		wordsOf("three"),
		snippetLengthCeiling,
	).Snippet

	if snippet != "One. Two. Three." {
		t.Fatalf("the snippet reads %q, want the short sentences shown together", snippet)
	}
}

func TestTheSnippetIsCutAtAWordBoundaryBeforeTheLengthCeiling(t *testing.T) {
	t.Parallel()

	snippet := documenttext.DocumentTextFrom(
		textOfThePage, wordsOf("city"), snippetLengthCeiling,
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

	snippet := documenttext.DocumentTextFrom(
		"Über kurz oder lang. Und dann.", wordsOf("paris"), shortSnippetLengthCeiling,
	).Snippet

	if snippet != "Über kurz oder lang." {
		t.Fatalf("the snippet reads %q, want the first passage of the text", snippet)
	}
}
