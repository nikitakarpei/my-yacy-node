// Package textsnippet chooses the snippet of a text for a query: the run of whole
// sentences, within a length ceiling, that shows the most distinct query words in
// prose rather than in menus, lists or code. A passage drops boilerplate at its edges,
// and a query word it misses can follow in an extra fragment. The service names how
// its text splits into words, so query words match as its search matches them.
package textsnippet

import (
	"iter"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	mainFragmentShare = 0.6
	fragmentJoint     = trailingCutMark + " "
)

type Rule struct {
	WordsIn       func(text string) iter.Seq2[int, string]
	LengthCeiling int
}

func (rule Rule) SnippetOf(text, title string, queryWords []string) string {
	query := askedQueryOf(queryWords)
	sentences := sentencesIn(text, title, rule.WordsIn)
	bestPassage := query.bestPassageIn(sentences, rule.LengthCeiling)
	if query.missingFrom(bestPassage, sentences).holdsWords() {
		if snippet, found := fragmentedSnippetIn(
			text,
			sentences,
			query,
			rule.LengthCeiling,
		); found {
			return snippet
		}
	}
	return bestPassage.textWithin(text, query, rule.LengthCeiling)
}

func fragmentedSnippetIn(
	text string,
	sentences []sentence,
	query askedQuery,
	lengthCeiling int,
) (string, bool) {
	mainCeiling := int(math.Round(float64(lengthCeiling) * mainFragmentShare))
	mainPassage := query.bestPassageIn(sentences, mainCeiling)
	mainFragment := mainPassage.textWithin(text, query, mainCeiling)
	missingQuery := query.missingFrom(mainPassage, sentences)
	extraCeiling := lengthCeiling - utf8.RuneCountInString(mainFragment) -
		utf8.RuneCountInString(fragmentJoint)
	sentencesBefore, sentencesAfter := sentencesAround(sentences, mainPassage)
	passageBefore := missingQuery.bestPassageIn(sentencesBefore, extraCeiling)
	passageAfter := missingQuery.bestPassageIn(sentencesAfter, extraCeiling)
	hitsBefore := extraFragmentHitsIn(passageBefore, missingQuery, extraCeiling)
	hitsAfter := extraFragmentHitsIn(passageAfter, missingQuery, extraCeiling)
	switch {
	case hitsBefore.distinctWords == 0 && hitsAfter.distinctWords == 0:
		return "", false
	case hitsBefore.outweigh(hitsAfter):
		return joinedFragments(
			passageBefore.textWithin(text, missingQuery, extraCeiling),
			mainFragment,
		), true
	default:
		return joinedFragments(
			mainFragment,
			passageAfter.textWithin(text, missingQuery, extraCeiling),
		), true
	}
}

func sentencesAround(sentences []sentence, shown passage) ([]sentence, []sentence) {
	firstShownSentence := sort.Search(len(sentences), func(sentenceIndex int) bool {
		return sentences[sentenceIndex].start >= shown.sentences[0].start
	})
	sentenceAfterShown := sort.Search(len(sentences), func(sentenceIndex int) bool {
		return sentences[sentenceIndex].start >= shown.sentences[len(shown.sentences)-1].end
	})
	return sentences[:firstShownSentence], sentences[sentenceAfterShown:]
}

func extraFragmentHitsIn(candidate passage, missingQuery askedQuery, extraCeiling int) queryHits {
	if !candidate.isFragmentWithin(extraCeiling) {
		return queryHits{}
	}
	return missingQuery.hitsIn(candidate)
}

func joinedFragments(firstFragment, secondFragment string) string {
	return strings.TrimSuffix(firstFragment, trailingCutMark) + fragmentJoint +
		strings.TrimPrefix(secondFragment, leadingCutMark)
}
