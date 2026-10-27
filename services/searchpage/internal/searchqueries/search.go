// Package searchqueries reads a search from the query that a person types and the page that they
// ask for. Operators in the query choose among the offered sources and the language of the
// search; every other word is a query word.
package searchqueries

import (
	"strconv"
	"strings"
)

type Source string

const (
	CrawledPages Source = "ct"
	YaCyNetwork  Source = "ya"
)

const (
	sourceOperator   = "!"
	languageOperator = ":"
	languageLength   = 2
	firstPageNumber  = 1
)

type Search struct {
	Language   string
	Words      []string
	Sources    []Source
	PageNumber int
}

func Sources() []Source {
	return []Source{CrawledPages, YaCyNetwork}
}

func SearchFrom(query string, pageNumber string, offeredSources []Source) Search {
	search := Search{PageNumber: pageNumberFrom(pageNumber)}
	chosenSources := map[Source]bool{}
	for _, word := range strings.Fields(query) {
		if source, isOperator := sourceOf(word, offeredSources); isOperator {
			chosenSources[source] = true
			continue
		}
		if language, isOperator := languageOf(word); isOperator {
			search.Language = language
			continue
		}
		search.Words = append(search.Words, word)
	}
	search.Sources = sourcesFrom(chosenSources, offeredSources)
	return search
}

func pageNumberFrom(pageNumber string) int {
	number, err := strconv.Atoi(pageNumber)
	if err != nil || number < firstPageNumber {
		return firstPageNumber
	}
	return number
}

func sourceOf(word string, offeredSources []Source) (Source, bool) {
	shortcut, isOperator := strings.CutPrefix(word, sourceOperator)
	if !isOperator {
		return "", false
	}
	for _, source := range offeredSources {
		if string(source) == shortcut {
			return source, true
		}
	}
	return "", false
}

func languageOf(word string) (string, bool) {
	code, isOperator := strings.CutPrefix(word, languageOperator)
	if !isOperator || len(code) != languageLength {
		return "", false
	}
	code = strings.ToLower(code)
	for _, letter := range code {
		if letter < 'a' || letter > 'z' {
			return "", false
		}
	}
	return code, true
}

func sourcesFrom(chosenSources map[Source]bool, offeredSources []Source) []Source {
	if len(chosenSources) == 0 {
		return offeredSources
	}
	sources := make([]Source, 0, len(chosenSources))
	for _, source := range offeredSources {
		if chosenSources[source] {
			sources = append(sources, source)
		}
	}
	return sources
}

func (s Search) HasWords() bool {
	return len(s.Words) > 0
}

func (s Search) Text() string {
	return strings.Join(s.Words, " ")
}

func (s Search) IsFirstPage() bool {
	return s.PageNumber == firstPageNumber
}
