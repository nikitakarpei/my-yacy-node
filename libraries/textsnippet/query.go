package textsnippet

type queryPhrase struct {
	firstWord  string
	secondWord string
}

type askedQuery struct {
	words   wordSet
	phrases map[queryPhrase]struct{}
}

func askedQueryOf(queryWords []string) askedQuery {
	query := askedQuery{words: wordSet{}, phrases: map[queryPhrase]struct{}{}}
	for wordIndex, word := range queryWords {
		query.words[word] = struct{}{}
		if wordIndex > 0 {
			query.phrases[queryPhrase{queryWords[wordIndex-1], word}] = struct{}{}
			query.phrases[queryPhrase{word, queryWords[wordIndex-1]}] = struct{}{}
		}
	}
	return query
}

func (query askedQuery) bestPassageIn(sentences []sentence, lengthCeiling int) passage {
	var bestPassage passage
	var bestHits queryHits
	lastSentence := 0
	for firstSentence := range sentences {
		lastSentence = max(lastSentence, firstSentence)
		for lastSentence+1 < len(sentences) &&
			sentences[lastSentence+1].endRune-sentences[firstSentence].startRune <= lengthCeiling {
			lastSentence++
		}
		candidate := passage{sentences: sentences[firstSentence : lastSentence+1]}.trimmedFor(query)
		hits := query.hitsIn(candidate)
		if firstSentence == 0 || hits.outweigh(bestHits) {
			bestPassage, bestHits = candidate, hits
		}
	}
	return bestPassage
}

func (query askedQuery) hitsIn(candidate passage) queryHits {
	counter := hitCounterFor(query)
	for _, sentenceOfThePassage := range candidate.sentences {
		for _, word := range sentenceOfThePassage.words {
			if sentenceOfThePassage.kind == proseSentence {
				counter.countProseWord(word.text)
			} else {
				counter.countWord(word.text)
			}
		}
	}
	return counter.hits()
}

func (query askedQuery) holds(word string) bool {
	return query.words.holds(word)
}

func (query askedQuery) holdsAnyOf(words []placedWord) bool {
	return query.words.holdsAnyOf(words)
}

func (query askedQuery) holdsPhrase(wordBefore, word string) bool {
	_, isQueryPhrase := query.phrases[queryPhrase{wordBefore, word}]
	return isQueryPhrase
}

func (query askedQuery) missingFrom(shown passage, sentences []sentence) askedQuery {
	shownWords := wordSet{}
	for _, sentenceOfThePassage := range shown.sentences {
		for _, word := range sentenceOfThePassage.words {
			shownWords[word.text] = struct{}{}
		}
	}
	missingQuery := askedQuery{words: wordSet{}}
	for _, sentenceOfTheText := range sentences {
		for _, word := range sentenceOfTheText.words {
			if !shownWords.holds(word.text) && query.holds(word.text) {
				missingQuery.words[word.text] = struct{}{}
			}
		}
	}
	return missingQuery
}

func (query askedQuery) holdsWords() bool {
	return len(query.words) > 0
}
