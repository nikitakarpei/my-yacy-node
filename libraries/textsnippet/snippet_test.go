package textsnippet_test

import (
	"iter"
	"strings"
	"testing"
	"unicode"

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
	return rule.SnippetOf(text, "", queryWords)
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

func TestTheSnippetIsTheEarlierPassageWhenTwoPassagesHoldTheSameQueryHits(t *testing.T) {
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

func TestTheSnippetOfAnOverlongSentenceShowsTheWindowWithTheMostQueryWords(t *testing.T) {
	t.Parallel()
	text := "heron " + strings.Repeat("reed ", 30) + "heron river " + strings.Repeat("reed ", 30)

	snippet := snippetOf(text, []string{"heron", "river"}, 40)

	if snippet != "… reed heron river reed reed reed …" {
		t.Fatalf("the snippet reads %q, want the window that holds both query words", snippet)
	}
}

func TestTheSnippetOfAnOverlongSentenceWithoutAQueryWordIsItsStart(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(strings.Repeat("reed ", 20), []string{"heron"}, 22)

	if snippet != "reed reed reed reed …" {
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

func TestTheSnippetOfAnOverlongSentenceCanStartRightAtAWord(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("aaaa ", 20) + "bb heron " + strings.Repeat("bbbb ", 20)

	snippet := snippetOf(text, []string{"heron"}, 40)

	if snippet != "… aaaa bb heron bbbb bbbb bbbb bbbb …" {
		t.Fatalf("the snippet reads %q, want whole words around the query word", snippet)
	}
}

func TestTheSnippetOfARunWithoutSpacesIsCutInsideIt(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("a", 60) + ",heron," + strings.Repeat("b", 60)

	snippet := snippetOf(text, []string{"heron"}, 20)

	if snippet != "… aa,heron,"+strings.Repeat("b", 7)+" …" {
		t.Fatalf("the snippet reads %q, want the letters around the query word", snippet)
	}
}

func TestTheSnippetShowsQueryWordsInProseOverAListOfMoreQueryWords(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"Tags: heron, river, reed, marsh, pond, lake.\n\nThe heron waits by the river at dawn for fish.",
		[]string{"heron", "river", "reed"},
		50,
	)

	if snippet != "The heron waits by the river at dawn for fish." {
		t.Fatalf("the snippet reads %q, want the prose sentence", snippet)
	}
}

func TestTheSnippetShowsQueryWordsInProseOverCodeOfMoreQueryWords(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"let heron = {river}; let reed = [marsh]; done.\n\nThe heron waits by the river at dawn for fish.",
		[]string{"heron", "river", "reed"},
		50,
	)

	if snippet != "The heron waits by the river at dawn for fish." {
		t.Fatalf("the snippet reads %q, want the prose sentence", snippet)
	}
}

func TestTheSnippetDropsBoilerplateAtItsEdges(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"Home Blog About me\nThe heron waits by the river at dawn for fish.\nShare on social media",
		[]string{"heron"},
		200,
	)

	if snippet != "The heron waits by the river at dawn for fish." {
		t.Fatalf("the snippet reads %q, want the prose without menus", snippet)
	}
}

func TestTheSnippetKeepsAHeadingThatSharesAWordWithTheTitle(t *testing.T) {
	t.Parallel()
	rule := textsnippet.Rule{WordsIn: lowercaseWordsIn, LengthCeiling: 200}

	snippet := rule.SnippetOf(
		"Home Blog\nWatching Notes\nThe heron waits by the river at dawn for fish.",
		"Heron Watching",
		[]string{"heron"},
	)

	if snippet != "Watching Notes The heron waits by the river at dawn for fish." {
		t.Fatalf("the snippet reads %q, want the heading before the prose", snippet)
	}
}

func TestTheSnippetOfAListKeepsItsEdges(t *testing.T) {
	t.Parallel()

	snippet := snippetOf("Home Blog\nHeron news\nRiver news", []string{"heron"}, 100)

	if snippet != "Home Blog Heron news River news" {
		t.Fatalf("the snippet reads %q, want the whole list", snippet)
	}
}

func TestTheSnippetCountsItsLengthAfterCollapsingSpaces(t *testing.T) {
	t.Parallel()

	snippet := snippetOf("Heron one."+strings.Repeat("\n", 10)+"Heron two.", []string{"heron"}, 21)

	if snippet != "Heron one. Heron two." {
		t.Fatalf("the snippet reads %q, want both sentences within the ceiling", snippet)
	}
}

const fillerProse = "Many small birds sing loudly in the old trees nearby. "

func TestTheSnippetAddsAnExtraFragmentWithAQueryWordAfterItsPassage(t *testing.T) {
	t.Parallel()
	text := "The heron waits in the reeds at dawn every day. " + strings.Repeat(fillerProse, 3) +
		"Fish swim slowly up the river in the spring."

	snippet := snippetOf(text, []string{"heron", "river"}, 120)

	if snippet != "The heron waits in the reeds at dawn every day. … Fish swim slowly up the river in the spring." {
		t.Fatalf(
			"the snippet reads %q, want the missing query word in an extra fragment after the passage",
			snippet,
		)
	}
}

func TestTheSnippetAddsAnExtraFragmentWithAQueryWordBeforeItsPassage(t *testing.T) {
	t.Parallel()
	text := "Fish swim slowly up the river in the spring. " + strings.Repeat(fillerProse, 3) +
		"The heron waits by the reed at dawn every day."

	snippet := snippetOf(text, []string{"heron", "reed", "river"}, 120)

	if snippet != "Fish swim slowly up the river in the spring. … The heron waits by the reed at dawn every day." {
		t.Fatalf(
			"the snippet reads %q, want the missing query word in an extra fragment before the passage",
			snippet,
		)
	}
}

func TestTheSnippetAddsNoExtraFragmentForAQueryWordOutsideProse(t *testing.T) {
	t.Parallel()
	text := "The heron waits in the reeds at dawn every day. " + strings.Repeat(fillerProse, 3) +
		"Tags: river, lake"

	snippet := snippetOf(text, []string{"heron", "river"}, 120)

	if snippet != "The heron waits in the reeds at dawn every day. "+strings.TrimSpace(
		fillerProse,
	) {
		t.Fatalf("the snippet reads %q, want one passage without an extra fragment", snippet)
	}
}

func TestTheSnippetOfAChineseTextDropsItsMenus(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"首页 博客 关于\n苍鹭在黎明时分静静地站在河边，等待着鱼儿游过芦苇丛。\n分享到社交媒体",
		[]string{"苍鹭在黎明时分静静地站在河边"},
		200,
	)

	if snippet != "苍鹭在黎明时分静静地站在河边，等待着鱼儿游过芦苇丛。" {
		t.Fatalf("the snippet reads %q, want the sentence without menus", snippet)
	}
}

func TestTheSnippetOfAnArabicTextDropsItsMenus(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"الرئيسية المدونة اتصل بنا\nيقف مالك الحزين على ضفة النهر عند الفجر وينتظر السمك.\nشارك على وسائل التواصل",
		[]string{"النهر"},
		200,
	)

	if snippet != "يقف مالك الحزين على ضفة النهر عند الفجر وينتظر السمك." {
		t.Fatalf("the snippet reads %q, want the sentence without menus", snippet)
	}
}

func TestTheSnippetOfAHebrewTextDropsItsMenus(t *testing.T) {
	t.Parallel()

	snippet := snippetOf(
		"דף הבית בלוג אודות\nהאנפה עומדת על גדת הנהר עם שחר ומחכה לדגים הקטנים.\nשתפו ברשתות החברתיות",
		[]string{"הנהר"},
		200,
	)

	if snippet != "האנפה עומדת על גדת הנהר עם שחר ומחכה לדגים הקטנים." {
		t.Fatalf("the snippet reads %q, want the sentence without menus", snippet)
	}
}
