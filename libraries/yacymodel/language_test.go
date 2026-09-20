package yacymodel_test

import (
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
