package textsnippet

type queryPhrase struct {
	firstWord  string
	secondWord string
}

type query struct {
	words   map[string]struct{}
	phrases map[queryPhrase]struct{}
}

func queryOf(queryWords []string) query {
	asked := query{words: map[string]struct{}{}, phrases: map[queryPhrase]struct{}{}}
	for place, word := range queryWords {
		asked.words[word] = struct{}{}
		if place > 0 {
			asked.phrases[queryPhrase{queryWords[place-1], word}] = struct{}{}
			asked.phrases[queryPhrase{word, queryWords[place-1]}] = struct{}{}
		}
	}
	return asked
}

func (asked query) bestPassageIn(sentences []sentence, lengthCeiling int) passage {
	var bestPassage passage
	var bestAnswer passageAnswer
	lastSentence := 0
	for firstSentence := range sentences {
		lastSentence = max(lastSentence, firstSentence)
		for lastSentence+1 < len(sentences) &&
			sentences[lastSentence+1].endRune-sentences[firstSentence].startRune <= lengthCeiling {
			lastSentence++
		}
		candidate := passage{sentences: sentences[firstSentence : lastSentence+1]}
		answer := asked.answerIn(candidate)
		if firstSentence == 0 || answer.isBetterThan(bestAnswer) {
			bestPassage, bestAnswer = candidate, answer
		}
	}
	return bestPassage
}

type passageAnswer struct {
	amountOfDistinctQueryWords int
	queryPhraseHits            int
	queryWordHits              int
}

func (asked query) answerIn(candidate passage) passageAnswer {
	var answer passageAnswer
	distinctQueryWords := map[string]struct{}{}
	wordBefore := ""
	for _, word := range candidate.words() {
		if _, isQueryWord := asked.words[word.text]; isQueryWord {
			answer.queryWordHits++
			distinctQueryWords[word.text] = struct{}{}
		}
		if _, isQueryPhrase := asked.phrases[queryPhrase{wordBefore, word.text}]; isQueryPhrase {
			answer.queryPhraseHits++
		}
		wordBefore = word.text
	}
	answer.amountOfDistinctQueryWords = len(distinctQueryWords)
	return answer
}

func (answer passageAnswer) isBetterThan(otherAnswer passageAnswer) bool {
	if answer.amountOfDistinctQueryWords != otherAnswer.amountOfDistinctQueryWords {
		return answer.amountOfDistinctQueryWords > otherAnswer.amountOfDistinctQueryWords
	}
	if answer.queryPhraseHits != otherAnswer.queryPhraseHits {
		return answer.queryPhraseHits > otherAnswer.queryPhraseHits
	}
	return answer.queryWordHits > otherAnswer.queryWordHits
}

func (asked query) firstPlaceIn(candidate passage) int {
	for _, word := range candidate.words() {
		if _, isQueryWord := asked.words[word.text]; isQueryWord {
			return word.place
		}
	}
	return candidate.sentences[0].start
}
