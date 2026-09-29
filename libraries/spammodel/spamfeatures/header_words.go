package spamfeatures

import (
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

const headerValueReadRunes = 30

var (
	cacheHeaders = map[string]bool{
		"cache-control": true, "pragma": true, "expires": true, "age": true, "via": true,
		"x-cache": true, "x-cache-lookup": true,
	}
	headersWithValueRead = map[string]bool{
		"server":       true,
		"x-powered-by": true,
		"x-generator":  true,
	}
	generatorPattern = regexp.MustCompile(
		`<meta[^>]+name\s*=\s*["']generator["'][^>]*content\s*=\s*["']([^"']+)`,
	)
)

func headerWordsOf(responseHeaders http.Header, lowercaseMarkup string) []string {
	words := make([]string, 0, len(responseHeaders))
	for _, name := range slices.Sorted(maps.Keys(responseHeaders)) {
		words = append(words, fieldWordsOf(strings.ToLower(name), responseHeaders[name])...)
	}
	for _, match := range generatorPattern.FindAllStringSubmatch(lowercaseMarkup, -1) {
		words = append(words, "generator="+firstWordOf(match[1]))
	}
	return words
}

func fieldWordsOf(lowercaseName string, values []string) []string {
	if cacheHeaders[lowercaseName] {
		return nil
	}
	var words []string
	for _, value := range values {
		words = append(words, "name:"+lowercaseName)
		if headersWithValueRead[lowercaseName] {
			product, _, _ := strings.Cut(value, "/")
			words = append(words, lowercaseName+"="+firstWordOf(product))
		}
	}
	return words
}

func firstWordOf(value string) string {
	words := strings.Fields(strings.ToLower(value))
	if len(words) == 0 {
		return ""
	}
	return prefixOf(words[0], headerValueReadRunes)
}
