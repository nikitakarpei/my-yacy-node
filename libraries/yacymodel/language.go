package yacymodel

import (
	"errors"
	"fmt"
)

const LanguageCodeLength = 2

var ErrInvalidLanguage = errors.New("invalid language code")

// Language is an ISO 639-1 two-letter language code.
type Language struct{ code [LanguageCodeLength]byte }

func ParseLanguage(code string) (Language, error) {
	if len(code) != LanguageCodeLength {
		return Language{}, lengthFailureOf(len(code))
	}

	return languageOfLetters(code[0], code[1])
}

func ParseLanguageBytes(code []byte) (Language, error) {
	if len(code) != LanguageCodeLength {
		return Language{}, lengthFailureOf(len(code))
	}

	return languageOfLetters(code[0], code[1])
}

func lengthFailureOf(length int) error {
	return fmt.Errorf("%w: length %d, want %d", ErrInvalidLanguage, length, LanguageCodeLength)
}

func languageOfLetters(first, second byte) (Language, error) {
	if !isLowercaseLetter(first) || !isLowercaseLetter(second) {
		return Language{}, fmt.Errorf(
			"%w: %q is not two lowercase letters",
			ErrInvalidLanguage,
			[]byte{first, second},
		)
	}

	return Language{code: [LanguageCodeLength]byte{first, second}}, nil
}

const lowercaseLetters = 'z' - 'a' + 1

func isLowercaseLetter(letter byte) bool {
	return letter >= 'a' && letter <= 'z'
}

func letterNumberOf(letter byte) int {
	return int(letter - 'a')
}

var everyLanguageCode = codesOfEveryLetterPair()

func codesOfEveryLetterPair() [lowercaseLetters * lowercaseLetters]string {
	var codes [lowercaseLetters * lowercaseLetters]string
	for first := range byte(lowercaseLetters) {
		for second := range byte(lowercaseLetters) {
			codes[int(first)*lowercaseLetters+int(second)] = string(
				[]byte{'a' + first, 'a' + second},
			)
		}
	}

	return codes
}

var LanguageOfUndeclaredDocument = Language{code: [LanguageCodeLength]byte{'e', 'n'}}

func (l Language) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l *Language) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*l = Language{}

		return nil
	}
	parsed, err := ParseLanguageBytes(text)
	if err != nil {
		return err
	}
	*l = parsed

	return nil
}

func (l Language) IsZero() bool { return l == Language{} }

func (l Language) String() string {
	if l.IsZero() {
		return ""
	}

	return everyLanguageCode[letterNumberOf(l.code[0])*lowercaseLetters+letterNumberOf(l.code[1])]
}
