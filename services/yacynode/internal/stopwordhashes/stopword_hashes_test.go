package stopwordhashes_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/stopwordhashes"
)

func TestTheHashOfAStopwordIsAStopwordInItsLanguage(t *testing.T) {
	t.Parallel()

	if !stopwordhashes.New().Contains(yacymodel.WordHash("der"), languageFrom(t, "de")) {
		t.Fatal("der is no German stopword, want it to be one")
	}
}

func TestTheHashOfAStopwordIsNoStopwordInAnotherLanguage(t *testing.T) {
	t.Parallel()

	if stopwordhashes.New().Contains(yacymodel.WordHash("der"), languageFrom(t, "en")) {
		t.Fatal("der is an English stopword, want it absent")
	}
}

func TestTheHashOfAWordThatCarriesMeaningIsNoStopword(t *testing.T) {
	t.Parallel()

	if stopwordhashes.New().Contains(yacymodel.WordHash("kernel"), languageFrom(t, "en")) {
		t.Fatal("kernel is an English stopword, want it absent")
	}
}

func TestALanguageWithoutAListHasNoStopwords(t *testing.T) {
	t.Parallel()

	if stopwordhashes.New().Contains(yacymodel.WordHash("de"), languageFrom(t, "nl")) {
		t.Fatal("de is a Dutch stopword, want Dutch to have none")
	}
}

func languageFrom(t *testing.T, code string) yacymodel.Language {
	t.Helper()

	language, err := yacymodel.ParseLanguage(code)
	if err != nil {
		t.Fatalf("ParseLanguage(%s): %v", code, err)
	}

	return language
}
