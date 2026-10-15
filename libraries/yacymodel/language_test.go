package yacymodel_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestALanguageReadsBackFromItsText(t *testing.T) {
	t.Parallel()

	german, err := yacymodel.ParseLanguage("de")
	if err != nil {
		t.Fatalf("ParseLanguage(de): %v", err)
	}
	text, err := german.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}

	var read yacymodel.Language
	if err := read.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText(%q): %v", text, err)
	}
	if read != german {
		t.Fatalf("the language reads %q, want %q", read, german)
	}
}

func TestALanguageOfNoTextIsTheLanguageOfADocumentThatDeclaresNone(t *testing.T) {
	t.Parallel()

	var read yacymodel.Language
	if err := read.UnmarshalText(nil); err != nil {
		t.Fatalf("UnmarshalText(nil): %v", err)
	}
	if !read.IsZero() {
		t.Fatalf("the language reads %q, want none", read)
	}
}

func TestALanguageParsedFromBytesIsTheLanguageParsedFromText(t *testing.T) {
	t.Parallel()

	for _, code := range []string{"aa", "de", "en", "zz"} {
		fromText, err := yacymodel.ParseLanguage(code)
		if err != nil {
			t.Fatalf("ParseLanguage(%q): %v", code, err)
		}
		fromBytes, err := yacymodel.ParseLanguageBytes([]byte(code))
		if err != nil {
			t.Fatalf("ParseLanguageBytes(%q): %v", code, err)
		}
		if fromBytes != fromText || fromBytes.String() != code {
			t.Fatalf("ParseLanguageBytes(%q) = %q, want %q", code, fromBytes, fromText)
		}
	}
}

func TestBytesThatAreNoLanguageCodeFailToParse(t *testing.T) {
	t.Parallel()

	for _, code := range []string{"", "d", "deu", "De", "d1", "{a", "a`"} {
		if _, err := yacymodel.ParseLanguageBytes([]byte(code)); !errors.Is(
			err,
			yacymodel.ErrInvalidLanguage,
		) {
			t.Fatalf("ParseLanguageBytes(%q) error = %v, want ErrInvalidLanguage", code, err)
		}
	}
}
