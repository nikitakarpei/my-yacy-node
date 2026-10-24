package textsnippet_test

import (
	"iter"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/nikitakarpei/yacy-rwi-node/textsnippet"
)

func lowercaseWordsIn(text string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		start := -1
		for place, letter := range text + " " {
			if unicode.IsLetter(letter) || unicode.IsDigit(letter) {
				if start < 0 {
					start = place
				}
				continue
			}
			if start >= 0 && !yield(start, strings.ToLower(text[start:place])) {
				return
			}
			start = -1
		}
	}
}

func snippetOf(text string, queryWords []string, lengthCeiling int) string {
	rule := textsnippet.Rule{WordsIn: lowercaseWordsIn, LengthCeiling: lengthCeiling}
	return rule.SnippetOf(text, queryWords)
}

func TestTheSnippetIsThePassageThatHoldsMoreOfTheQueryWords(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"Berlin holds a wall. The wall and the river of Berlin.",
		[]string{"berlin", "river"},
		40,
	)

	if snippet != "The wall and the river of Berlin." {
		t.Fatalf("the snippet reads %q, want the passage that holds both query words", snippet)
	}
}

func TestTheSnippetIsThePassageThatHoldsTheQueryPhraseWhenBothHoldTheSameWords(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"The wall stands near Berlin. The Berlin wall is famous.",
		[]string{"berlin", "wall"},
		40,
	)

	if snippet != "The Berlin wall is famous." {
		t.Fatalf("the snippet reads %q, want the passage that holds the query phrase", snippet)
	}
}

func TestTheSnippetIsTheEarlierPassageWhenTwoPassagesAnswerTheQueryTheSame(t *testing.T) {
	t.Parallel()

	snippet := snippetOf("Berlin is old. Berlin is new.", []string{"berlin"}, 20)

	if snippet != "Berlin is old." {
		t.Fatalf("the snippet reads %q, want the earlier of two equal passages", snippet)
	}
}

func TestTheSnippetJoinsShortSentencesUpToTheLengthCeiling(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"One. Two. Three. This other sentence holds no query word at all.",
		[]string{"three"},
		40,
	)

	if snippet != "One. Two. Three." {
		t.Fatalf("the snippet reads %q, want the short sentences shown together", snippet)
	}
}

func TestTheSnippetCanStartAtAnySentence(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"Aaaa bbbb cccc. Dddd berlin. River eeee.",
		[]string{"berlin", "river"},
		30,
	)

	if snippet != "Dddd berlin. River eeee." {
		t.Fatalf("the snippet reads %q, want the sentences that hold both query words", snippet)
	}
}

func TestTheSnippetOfALongSentenceShowsItsFirstQueryWord(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("meadow grass ", 20) + "heron " + strings.Repeat("reed ", 20)

	snippet := snippetOf(text, []string{"heron"}, 40)

	if utf8.RuneCountInString(snippet) > 40 || !strings.Contains(snippet, "heron") ||
		!strings.Contains(text, " "+snippet+" ") {
		t.Fatalf("the snippet reads %q, want whole words around the query word", snippet)
	}
}

func TestTheSnippetOfALongSentenceWithoutAQueryWordIsItsStart(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(strings.Repeat("reed ", 20), []string{"heron"}, 22)

	if snippet != "reed reed reed reed" {
		t.Fatalf("the snippet reads %q, want the start of the sentence cut at a word", snippet)
	}
}

func TestTheSnippetOfATextWithoutAQueryWordIsItsFirstPassage(t *testing.T) {
	t.Parallel()

	snippet := snippetOf("Über kurz oder lang. Und dann.", []string{"paris"}, 20)

	if snippet != "Über kurz oder lang." {
		t.Fatalf("the snippet reads %q, want the first passage of the text", snippet)
	}
}

func TestTheSnippetCollapsesItsSpaces(t *testing.T) {
	t.Parallel()

	snippet := snippetOf("Berlin\n\n  wall   river.", []string{"wall"}, 40)

	if snippet != "Berlin wall river." {
		t.Fatalf("the snippet reads %q, want its spaces collapsed", snippet)
	}
}

func TestTheSnippetOfAnEmptyTextIsEmpty(t *testing.T) {
	t.Parallel()

	if snippet := snippetOf(" \n ", []string{"wall"}, 40); snippet != "" {
		t.Fatalf("the snippet reads %q, want no snippet", snippet)
	}
}

func TestTheSnippetOfALongSentenceCanStartRightAtAWord(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("aaaa ", 20) + "heron " + strings.Repeat("bbbb ", 20)

	snippet := snippetOf(text, []string{"heron"}, 40)

	if !strings.HasPrefix(snippet, "aaaa ") || !strings.Contains(snippet, "heron") ||
		utf8.RuneCountInString(snippet) > 40 {
		t.Fatalf("the snippet reads %q, want whole words around the query word", snippet)
	}
}

func TestTheSnippetOfARunWithoutSpacesIsCutInsideIt(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("a", 60) + ",heron," + strings.Repeat("b", 60)

	snippet := snippetOf(text, []string{"heron"}, 20)

	if snippet != strings.Repeat("a", 9)+",heron,"+strings.Repeat("b", 4) {
		t.Fatalf("the snippet reads %q, want the letters around the query word", snippet)
	}
}
