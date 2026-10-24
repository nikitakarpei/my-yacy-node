// Package textsnippet chooses the snippet of a text for a query: the run of whole
// sentences, within a length ceiling, that shows the most distinct query words in
// prose rather than in menus, lists or code. A passage drops boilerplate at its edges,
// and a query word it misses can follow in a second fragment. The service names how
// its text splits into words, so query words match as its search matches them.
package textsnippet

import (
	"iter"
	"strings"
	"unicode/utf8"
)

const (
	mainFragmentShare   = 3
	mainFragmentDivisor = 5
	fragmentJoint       = " " + cutMark + " "
)

type Rule struct {
	WordsIn       func(text string) iter.Seq2[int, string]
	LengthCeiling int
}

func (rule Rule) SnippetOf(title, text string, queryWords []string) string {
	query := askedQueryOf(queryWords)
	sentences := sentencesIn(text, title, rule.WordsIn)
	bestPassage := query.bestPassageIn(sentences, rule.LengthCeiling)
	if query.missingFrom(bestPassage, sentences).holdsNoWords() {
		return bestPassage.textWithin(text, query, rule.LengthCeiling)
	}
	if snippet, found := fragmentedSnippetIn(text, sentences, query, rule.LengthCeiling); found {
		return snippet
	}
	return bestPassage.textWithin(text, query, rule.LengthCeiling)
}

func fragmentedSnippetIn(
	text string,
	sentences []sentence,
	query askedQuery,
	lengthCeiling int,
) (string, bool) {
	mainCeiling := lengthCeiling * mainFragmentShare / mainFragmentDivisor
	mainPassage := query.bestPassageIn(sentences, mainCeiling)
	mainFragment := mainPassage.textWithin(text, query, mainCeiling)
	missing := query.missingFrom(mainPassage, sentences)
	restCeiling := lengthCeiling - utf8.RuneCountInString(mainFragment) -
		utf8.RuneCountInString(fragmentJoint)
	sentencesBefore, sentencesAfter := sentencesAround(sentences, mainPassage)
	passageBefore := missing.bestPassageIn(sentencesBefore, restCeiling)
	passageAfter := missing.bestPassageIn(sentencesAfter, restCeiling)
	hitsBefore := missing.wholeProseHitsIn(passageBefore, restCeiling)
	hitsAfter := missing.wholeProseHitsIn(passageAfter, restCeiling)
	switch {
	case hitsBefore.distinctWords == 0 && hitsAfter.distinctWords == 0:
		return "", false
	case hitsAfter.distinctWords == 0 || hitsBefore.outweigh(hitsAfter):
		return joinedFragments(
			passageBefore.textWithin(text, missing, restCeiling),
			mainFragment,
		), true
	default:
		return joinedFragments(
			mainFragment,
			passageAfter.textWithin(text, missing, restCeiling),
		), true
	}
}

func sentencesAround(sentences []sentence, shown passage) ([]sentence, []sentence) {
	first := shown.sentences[0].start
	last := shown.sentences[len(shown.sentences)-1].end
	var sentencesBefore, sentencesAfter []sentence
	for _, sentenceOfTheText := range sentences {
		if sentenceOfTheText.end <= first {
			sentencesBefore = append(sentencesBefore, sentenceOfTheText)
		}
		if sentenceOfTheText.start >= last {
			sentencesAfter = append(sentencesAfter, sentenceOfTheText)
		}
	}
	return sentencesBefore, sentencesAfter
}

func joinedFragments(firstFragment, secondFragment string) string {
	return strings.TrimSuffix(firstFragment, " "+cutMark) + fragmentJoint +
		strings.TrimPrefix(secondFragment, cutMark+" ")
}
