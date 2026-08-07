package languageindex_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
)

func TestIndexesForNamesTheCatchAllAndEachAllowedLanguage(t *testing.T) {
	indexes, err := languageindex.IndexesFor("yacy_text", []string{"en", "DE-de"})
	if err != nil {
		t.Fatalf("indexes: %v", err)
	}
	if indexes.Prefix() != "yacy_text_v1" {
		t.Errorf("prefix = %q", indexes.Prefix())
	}
	names := make([]string, 0, len(indexes.All()))
	for _, index := range indexes.All() {
		names = append(names, index.Name)
	}
	want := []string{"yacy_text_v1_und", "yacy_text_v1_en", "yacy_text_v1_de"}
	if !slices.Equal(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestIndexesForWithoutAllowedLanguagesHoldsTheCatchAllOnly(t *testing.T) {
	indexes, err := languageindex.IndexesFor("yacy_text", nil)
	if err != nil {
		t.Fatalf("indexes: %v", err)
	}
	if len(indexes.All()) != 1 {
		t.Fatalf("indexes = %v", indexes.All())
	}
	if indexes.All()[0].Language != languageindex.UndeterminedLanguage {
		t.Errorf("language = %q", indexes.All()[0].Language)
	}
}

func TestNameForRoutesByPrimarySubtag(t *testing.T) {
	indexes, err := languageindex.IndexesFor("yacy_text", []string{"en"})
	if err != nil {
		t.Fatalf("indexes: %v", err)
	}
	for _, documentLanguage := range []string{"en", "en-US", "EN"} {
		if name := indexes.NameFor(documentLanguage); name != "yacy_text_v1_en" {
			t.Errorf("name for %q = %q", documentLanguage, name)
		}
	}
	for _, documentLanguage := range []string{"", "de", "zz"} {
		if name := indexes.NameFor(documentLanguage); name != "yacy_text_v1_und" {
			t.Errorf("name for %q = %q", documentLanguage, name)
		}
	}
}

func TestIndexesForRejectsAnUnsupportedLanguage(t *testing.T) {
	if _, err := languageindex.IndexesFor("yacy_text", []string{"zz"}); err == nil {
		t.Fatal("expected error for an unsupported language")
	}
}

func TestIndexesForRejectsARepeatedLanguage(t *testing.T) {
	if _, err := languageindex.IndexesFor("yacy_text", []string{"en", "en-GB"}); err == nil {
		t.Fatal("expected error for a repeated language")
	}
}

func TestIndexesForRejectsABaseNameOutsideTheManticoreCharset(t *testing.T) {
	for _, baseName := range []string{"", "yacy-text", "Yacy", "1yacy"} {
		if _, err := languageindex.IndexesFor(baseName, nil); err == nil {
			t.Errorf("expected error for base name %q", baseName)
		}
	}
}
