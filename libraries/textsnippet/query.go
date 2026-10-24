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
		candidate := passage{sentences: sentences[firstSentence : lastSentence+1]}
		hits := query.hitsIn(candidate)
		if firstSentence == 0 || hits.outweigh(bestHits) {
			bestPassage, bestHits = candidate, hits
		}
	}
	return bestPassage
}

type queryHits struct {
	distinctWords int
	phrases       int
	words         int
}

func (query askedQuery) hitsIn(candidate passage) queryHits {
	var hits queryHits
	distinctWords := map[string]struct{}{}
	wordBefore := ""
	for word := range candidate.words() {
		if _, isQueryWord := query.words[word.text]; isQueryWord {
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

func (hits queryHits) outweigh(otherHits queryHits) bool {
	if hits.distinctWords != otherHits.distinctWords {
		return hits.distinctWords > otherHits.distinctWords
	}
	if hits.phrases != otherHits.phrases {
		return hits.phrases > otherHits.phrases
	}
	return hits.words > otherHits.words
}

func (query askedQuery) firstMatchIn(candidate passage) int {
	for word := range candidate.words() {
		if _, isQueryWord := query.words[word.text]; isQueryWord {
			return word.place
		}
	}
	return candidate.sentences[0].start
}
