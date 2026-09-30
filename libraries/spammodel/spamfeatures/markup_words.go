package spamfeatures

import (
	"regexp"
	"strings"
)

const (
	markupWordsRead       = 3000
	attributeMarkerPrefix = "attr_"
)

var (
	tagPattern             = regexp.MustCompile(`<\s*([a-z][a-z0-9]*)`)
	attributeMarkerPattern = regexp.MustCompile(
		`\s(rel|style|hidden|onclick|target|nofollow|sponsored|display:\s*none|` +
			`visibility:\s*hidden|refresh|iframe|async|data-[a-z]+)`,
	)
)

func markupTextOf(lowercaseMarkup string) string {
	words := submatchesOf(tagPattern, lowercaseMarkup, "")
	words = append(
		words,
		submatchesOf(attributeMarkerPattern, lowercaseMarkup, attributeMarkerPrefix)...)
	return strings.Join(words, " ")
}

func submatchesOf(pattern *regexp.Regexp, lowercaseMarkup, prefix string) []string {
	matches := pattern.FindAllStringSubmatch(lowercaseMarkup, markupWordsRead)
	submatches := make([]string, 0, len(matches))
	for _, match := range matches {
		submatches = append(submatches, prefix+match[1])
	}
	return submatches
}
