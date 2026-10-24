package textsnippet

import "strings"

type passage struct {
	sentences []sentence
}

func (candidate passage) trimmedFor(query askedQuery) passage {
	if !candidate.holdsProse() {
		return candidate
	}
	sentences := candidate.sentences
	for len(sentences) > 1 && isBoilerplateFor(sentences[0], query) && !sentences[0].heading {
		sentences = sentences[1:]
	}
	for len(sentences) > 1 && isBoilerplateFor(sentences[len(sentences)-1], query) {
		sentences = sentences[:len(sentences)-1]
	}
	return passage{sentences: sentences}
}

func (candidate passage) holdsProse() bool {
	for _, sentenceOfThePassage := range candidate.sentences {
		if sentenceOfThePassage.prose {
			return true
		}
	}
	return false
}

func (candidate passage) isAllProse() bool {
	for _, sentenceOfThePassage := range candidate.sentences {
		if !sentenceOfThePassage.prose {
			return false
		}
	}
	return true
}

func isBoilerplateFor(edge sentence, query askedQuery) bool {
	return edge.boilerplate && !holdsAnyOf(edge.words, query.words)
}

func (candidate passage) length() int {
	return candidate.sentences[len(candidate.sentences)-1].endRune - candidate.sentences[0].startRune
}

func (candidate passage) textWithin(text string, query askedQuery, lengthCeiling int) string {
	if len(candidate.sentences) == 0 {
		return ""
	}
	first, last := candidate.sentences[0], candidate.sentences[len(candidate.sentences)-1]
	if candidate.length() <= lengthCeiling {
		return withSpacesCollapsed(text[first.start:last.end])
	}
	return longSentenceOf(text, first).windowFor(query, lengthCeiling)
}

func withSpacesCollapsed(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
