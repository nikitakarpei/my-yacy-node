package textsnippet

type queryPhrase struct {
	firstWord  string
	secondWord string
}

type askedQuery struct {
	words   map[string]struct{}
	phrases map[queryPhrase]struct{}
}

func askedQueryOf(queryWords []string) askedQuery {
	query := askedQuery{words: map[string]struct{}{}, phrases: map[queryPhrase]struct{}{}}
	for place, word := range queryWords {
		query.words[word] = struct{}{}
		if place > 0 {
			query.phrases[queryPhrase{queryWords[place-1], word}] = struct{}{}
			query.phrases[queryPhrase{word, queryWords[place-1]}] = struct{}{}
		}
	}
	return query
}

func (query askedQuery) bestPassageIn(sentences []sentence, lengthCeiling int) passage {
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
		answer := query.answerIn(candidate)
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

func (query askedQuery) answerIn(candidate passage) passageAnswer {
	var answer passageAnswer
	distinctQueryWords := map[string]struct{}{}
	wordBefore := ""
	for word := range candidate.words() {
		if _, isQueryWord := query.words[word.text]; isQueryWord {
			answer.queryWordHits++
			distinctQueryWords[word.text] = struct{}{}
		}
		if _, isQueryPhrase := query.phrases[queryPhrase{wordBefore, word.text}]; isQueryPhrase {
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

func (query askedQuery) firstMatchIn(candidate passage) int {
	for word := range candidate.words() {
		if _, isQueryWord := query.words[word.text]; isQueryWord {
			return word.place
		}
	}
	return candidate.sentences[0].start
}
