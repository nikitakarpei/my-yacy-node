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
	for len(sentences) > 1 && sentences[0].isDroppableFor(query) && !sentences[0].heading {
		sentences = sentences[1:]
	}
	for len(sentences) > 1 && sentences[len(sentences)-1].isDroppableFor(query) {
		sentences = sentences[:len(sentences)-1]
	}
	return passage{sentences: sentences}
}

func (candidate passage) holdsProse() bool {
	for _, sentenceOfThePassage := range candidate.sentences {
		if sentenceOfThePassage.kind == proseSentence {
			return true
		}
	}
	return false
}

func (candidate passage) isFragmentWithin(lengthCeiling int) bool {
	return len(candidate.sentences) > 0 && candidate.length() <= lengthCeiling &&
		candidate.holdsOnlyProse()
}

func (candidate passage) length() int {
	return candidate.sentences[len(candidate.sentences)-1].endRune - candidate.sentences[0].startRune
}

func (candidate passage) holdsOnlyProse() bool {
	for _, sentenceOfThePassage := range candidate.sentences {
		if sentenceOfThePassage.kind != proseSentence {
			return false
		}
	}
	return true
}

func (candidate passage) textWithin(text string, query askedQuery, lengthCeiling int) string {
	if len(candidate.sentences) == 0 {
		return ""
	}
	first, last := candidate.sentences[0], candidate.sentences[len(candidate.sentences)-1]
	if candidate.length() <= lengthCeiling {
		return withSpacesCollapsed(text[first.start:last.end])
	}
	return collapsedSentenceOf(text, first).textFor(query, lengthCeiling)
}

func withSpacesCollapsed(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
