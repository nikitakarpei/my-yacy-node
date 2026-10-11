package stopwords_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/stopwords"
)

func TestEachListedLanguageHoldsItsStopwords(t *testing.T) {
	t.Parallel()

	stopwordPerLanguage := map[string]string{
		"de": "der",
		"en": "the",
		"es": "los",
		"fr": "les",
		"it": "gli",
		"ru": "для",
	}

	for language, stopword := range stopwordPerLanguage {
		if !stopwordsOf(t, language).Contains(stopword) {
			t.Errorf("stopwords of %s lack %q", language, stopword)
		}
	}
}

func TestAWordThatCarriesMeaningIsNoStopword(t *testing.T) {
	t.Parallel()

	if stopwordsOf(t, "en").Contains("kernel") {
		t.Fatal("kernel is an English stopword, want it absent")
	}
}

func TestALanguageWithoutAListIsNotListed(t *testing.T) {
	t.Parallel()

	stopwordsOfDutch, listed := stopwords.Of("nl")

	if listed || stopwordsOfDutch.Contains("de") {
		t.Fatalf("stopwords of nl listed = %v, want nl unlisted and holding no stopword", listed)
	}
}

func TestTheLanguagesAreTheOnesWithAList(t *testing.T) {
	t.Parallel()

	languages := stopwords.Languages()

	if !slices.Equal(languages, []string{"de", "en", "es", "fr", "it", "ru"}) {
		t.Fatalf("Languages = %v, want de en es fr it ru", languages)
	}
}

func TestTheWordsAreTheWholeList(t *testing.T) {
	t.Parallel()

	words := stopwordsOf(t, "en").Words()

	if len(words) != 172 || !slices.Contains(words, "the") || !slices.IsSorted(words) {
		t.Fatalf("Words of en = %d words, want the 172 of the list in order", len(words))
	}
}

func TestTheLanguageCoveringMostWordsIsNamed(t *testing.T) {
	t.Parallel()

	language, named := stopwords.LanguageCoveringMostOf(
		[]string{"how", "do", "reset", "my", "router"},
	)

	if !named || language != "en" {
		t.Fatalf("LanguageCoveringMostOf = %q %v, want en", language, named)
	}
}

func TestTwoLanguagesCoveringAsManyWordsNameNone(t *testing.T) {
	t.Parallel()

	language, named := stopwords.LanguageCoveringMostOf([]string{"mejores", "playas", "de"})

	if named {
		t.Fatalf("LanguageCoveringMostOf = %q, want none: de is Spanish and French", language)
	}
}

func TestWordsNoLanguageCoversNameNone(t *testing.T) {
	t.Parallel()

	language, named := stopwords.LanguageCoveringMostOf([]string{"recette", "crepes"})

	if named {
		t.Fatalf("LanguageCoveringMostOf = %q, want none", language)
	}
}

func stopwordsOf(t *testing.T, language string) stopwords.Stopwords {
	t.Helper()

	stopwordsOfTheLanguage, listed := stopwords.Of(language)
	if !listed {
		t.Fatalf("Of(%s): not listed", language)
	}

	return stopwordsOfTheLanguage
}
