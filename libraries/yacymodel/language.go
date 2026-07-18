package yacymodel

import (
	"errors"
	"fmt"
)

const LanguageCodeLength = 2

var ErrInvalidLanguage = errors.New("invalid language code")

// Language is an ISO 639-1 two-letter language code.
type Language string

func ParseLanguage(code string) (Language, error) {
	if len(code) != LanguageCodeLength {
		return "", fmt.Errorf(
			"%w: length %d, want %d",
			ErrInvalidLanguage,
			len(code),
			LanguageCodeLength,
		)
	}
	return Language(code), nil
}
