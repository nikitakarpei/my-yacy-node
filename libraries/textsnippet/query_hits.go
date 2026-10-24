package textsnippet

type queryHits struct {
	distinctProseWords int
	distinctWords      int
	phrases            int
	words              int
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

type queryHitCounter struct {
	query              askedQuery
	phrases            int
	words              int
	distinctWords      wordSet
	distinctProseWords wordSet
	wordBefore         string
}

func hitCounterFor(query askedQuery) *queryHitCounter {
	return &queryHitCounter{query: query, distinctWords: wordSet{}, distinctProseWords: wordSet{}}
}

func (counter *queryHitCounter) countWord(word string) {
	if counter.query.holds(word) {
		counter.words++
		counter.distinctWords[word] = struct{}{}
	}
	if counter.query.holdsPhrase(counter.wordBefore, word) {
		counter.phrases++
	}
	counter.wordBefore = word
}

func (counter *queryHitCounter) countProseWord(word string) {
	counter.countWord(word)
	if counter.query.holds(word) {
		counter.distinctProseWords[word] = struct{}{}
	}
}

func (counter *queryHitCounter) hits() queryHits {
	return queryHits{
		distinctProseWords: len(counter.distinctProseWords),
		distinctWords:      len(counter.distinctWords),
		phrases:            counter.phrases,
		words:              counter.words,
	}
}
