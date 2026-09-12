package documenttext

import (
	"iter"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/sentences"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type passage struct {
	text                       string
	amountOfDistinctQueryWords int
	queryPhraseHits            int
	queryWordHits              int
}

func bestPassageForTheQueryWords(
	queryWords []yacymodel.Hash,
	pageText string,
	lengthCeiling int,
) passage {
	var bestPassage passage
	firstPassageRead := false
	for passageText := range passageTextsIn(pageText, lengthCeiling) {
		readPassage := passageOf(passageText, queryWords)
		if !firstPassageRead || readPassage.answersTheQueryBetterThan(bestPassage) {
			bestPassage = readPassage
			firstPassageRead = true
		}
	}

	return bestPassage
}

func passageTextsIn(pageText string, lengthCeiling int) iter.Seq[string] {
	return func(yield func(string) bool) {
		passageText := ""
		sentencesOfThePage := sentences.FromString(pageText)
		for sentencesOfThePage.Next() {
			sentence := sentencesOfThePage.Value()
			if passageText != "" &&
				utf8.RuneCountInString(passageText+sentence) > lengthCeiling {
				if !yield(passageText) {
					return
				}
				passageText = ""
			}
			passageText += sentence
		}
		if passageText != "" {
			yield(passageText)
		}
	}
}

func passageOf(passageText string, queryWords []yacymodel.Hash) passage {
	counts := textCountsOf(passageText, queryWords)
	readPassage := passage{text: passageText, queryPhraseHits: counts.queryPhraseHits}
	for _, hits := range counts.hitsPerQueryWord {
		if hits > 0 {
			readPassage.amountOfDistinctQueryWords++
		}
		readPassage.queryWordHits += hits
	}

	return readPassage
}

func (p passage) answersTheQueryBetterThan(otherPassage passage) bool {
	if p.amountOfDistinctQueryWords != otherPassage.amountOfDistinctQueryWords {
		return p.amountOfDistinctQueryWords > otherPassage.amountOfDistinctQueryWords
	}
	if p.queryPhraseHits != otherPassage.queryPhraseHits {
		return p.queryPhraseHits > otherPassage.queryPhraseHits
	}

	return p.queryWordHits > otherPassage.queryWordHits
}
