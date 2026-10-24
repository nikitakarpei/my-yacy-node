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
	hits := query.hitsAmong(candidate.words())
	distinctProseWords := map[string]struct{}{}
	for _, sentenceOfThePassage := range candidate.sentences {
		if !sentenceOfThePassage.prose {
			continue
		}
		for _, word := range sentenceOfThePassage.words {
			if query.holds(word.text) {
				distinctProseWords[word.text] = struct{}{}
			}
		}
	}
	hits.distinctProseWords = len(distinctProseWords)
	return hits
}

func (query askedQuery) hitsAmong(words []placedWord) queryHits {
	var hits queryHits
	distinctWords := map[string]struct{}{}
	wordBefore := ""
	for _, word := range words {
		if query.holds(word.text) {
			hits.words++
			distinctWords[word.text] = struct{}{}
		}
		if _, isQueryPhrase := query.phrases[queryPhrase{wordBefore, word.text}]; isQueryPhrase {
			hits.phrases++
		}
		wordBefore = word.text
	}
	hits.distinctWords = len(distinctWords)
	return hits
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
	for _, word := range shown.words() {
		shownWords[word.text] = struct{}{}
	}
	missing := askedQuery{words: map[string]struct{}{}, phrases: map[queryPhrase]struct{}{}}
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
	if len(candidate.sentences) == 0 || candidate.length() > lengthCeiling {
		return queryHits{}
	}
	for _, sentenceOfThePassage := range candidate.sentences {
		if !sentenceOfThePassage.prose {
			return queryHits{}
		}
	}
	return query.hitsIn(candidate)
}
