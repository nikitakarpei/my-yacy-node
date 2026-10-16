package robotsexclusion

import (
	"errors"
	"fmt"
	"strings"
)

const tcharSymbols = "!#$%&'*+-.^_`|~"

var errNoProductToken = errors.New(
	"user agent does not start with a robots.txt product token of letters, '_' and '-'",
)

type ProductToken string

func ProductTokenOf(userAgent string) (ProductToken, error) {
	leadingToken := leadingTokenOf(userAgent)
	if leadingToken == "" || strings.IndexFunc(leadingToken, isOutsideProductToken) >= 0 {
		return "", fmt.Errorf("%q: %w", userAgent, errNoProductToken)
	}
	return ProductToken(leadingToken), nil
}

func leadingTokenOf(userAgent string) string {
	end := strings.IndexFunc(userAgent, isOutsideToken)
	if end < 0 {
		return userAgent
	}
	return userAgent[:end]
}

func isOutsideToken(character rune) bool {
	return !isLetter(character) && !isDigit(character) &&
		!strings.ContainsRune(tcharSymbols, character)
}

func isOutsideProductToken(character rune) bool {
	return !isLetter(character) && character != '_' && character != '-'
}

func isLetter(character rune) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
}

func isDigit(character rune) bool {
	return character >= '0' && character <= '9'
}
