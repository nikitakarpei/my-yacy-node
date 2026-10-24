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

type queryHits struct {
	distinctProseWords int
	distinctWords      int
	phrases            int
	words              int
}

func (query askedQuery) hitsIn(candidate passage) queryHits {
	counter := query.hitCounter()
	for _, sentenceOfThePassage := range candidate.sentences {
		counter.count(sentenceOfThePassage.words, sentenceOfThePassage.prose)
	}
	return counter.hits
}

func (query askedQuery) hitsAmong(words []placedWord) queryHits {
	counter := query.hitCounter()
	counter.count(words, false)
	return counter.hits
}

type queryHitCounter struct {
	query              askedQuery
	hits               queryHits
	distinctWords      map[string]struct{}
	distinctProseWords map[string]struct{}
	wordBefore         string
}

func (query askedQuery) hitCounter() *queryHitCounter {
	return &queryHitCounter{
		query:              query,
		distinctWords:      map[string]struct{}{},
		distinctProseWords: map[string]struct{}{},
	}
}

func (counter *queryHitCounter) count(words []placedWord, inProse bool) {
	for _, word := range words {
		if counter.query.holds(word.text) {
			counter.hits.words++
			counter.distinctWords[word.text] = struct{}{}
			if inProse {
				counter.distinctProseWords[word.text] = struct{}{}
			}
		}
		if _, isQueryPhrase := counter.query.phrases[queryPhrase{counter.wordBefore, word.text}]; isQueryPhrase {
			counter.hits.phrases++
		}
		counter.wordBefore = word.text
	}
	counter.hits.distinctWords = len(counter.distinctWords)
	counter.hits.distinctProseWords = len(counter.distinctProseWords)
}

func (query askedQuery) holds(word string) bool {
	_, isQueryWord := query.words[word]
	return isQueryWord
}

func (hits queryHits) outweigh(otherHits queryHits) bool {
	if hits.distinctProseWords != otherHits.distinctProseWords {
		return hits.distinctProseWords > otherHits.distinctProseWords
	}
	if hits.distinctWords != otherHits.distinctWords {
		return hits.distinctWords > otherHits.distinctWords
	}
	if hits.phrases != otherHits.phrases {
		return hits.phrases > otherHits.phrases
	}
	return hits.words > otherHits.words
}

func (query askedQuery) missingFrom(shown passage, sentences []sentence) askedQuery {
	shownWords := map[string]struct{}{}
	for _, shownSentence := range shown.sentences {
		for _, word := range shownSentence.words {
			shownWords[word.text] = struct{}{}
		}
	}
	missing := askedQuery{words: map[string]struct{}{}}
	for _, sentenceOfTheText := range sentences {
		for _, word := range sentenceOfTheText.words {
			if _, isShown := shownWords[word.text]; !isShown && query.holds(word.text) {
				missing.words[word.text] = struct{}{}
			}
		}
	}
	return missing
}

func (query askedQuery) holdsNoWords() bool {
	return len(query.words) == 0
}

func (query askedQuery) wholeProseHitsIn(candidate passage, lengthCeiling int) queryHits {
	if len(candidate.sentences) == 0 || candidate.length() > lengthCeiling ||
		!candidate.isAllProse() {
		return queryHits{}
	}
	return query.hitsIn(candidate)
}
