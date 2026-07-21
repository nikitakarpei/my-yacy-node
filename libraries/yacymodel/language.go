package yacymodel

import (
	"errors"
	"fmt"
)

const LanguageCodeLength = 2

var ErrInvalidLanguage = errors.New("invalid language code")

// Language is an ISO 639-1 two-letter language code.
type Language struct{ value string }

func ParseLanguage(code string) (Language, error) {
	if len(code) != LanguageCodeLength {
		return Language{}, fmt.Errorf(
			"%w: length %d, want %d",
			ErrInvalidLanguage,
			len(code),
			LanguageCodeLength,
		)
	}
	return Language{value: code}, nil
}

func (l Language) IsZero() bool { return l.value == "" }

func (l Language) String() string { return l.value }
