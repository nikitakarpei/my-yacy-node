package stopwords_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stopwords"
)

func TestTheStopwordsOfTheLanguageTheClientNamedGoAway(t *testing.T) {
	t.Parallel()

	contentWords := stopwords.ContentWordsOf([]string{"die", "hard", "filme"}, "lang_de")

	if !slices.Equal(contentWords, []string{"hard", "filme"}) {
		t.Fatalf("ContentWordsOf = %v, want hard filme", contentWords)
	}
}

func TestTheLanguageTheClientNamedHoldsAgainstTheWordsThemselves(t *testing.T) {
	t.Parallel()

	contentWords := stopwords.ContentWordsOf([]string{"die", "hard"}, "en")

	if !slices.Equal(contentWords, []string{"die", "hard"}) {
		t.Fatalf("ContentWordsOf = %v, want die hard", contentWords)
	}
}

func TestTheLanguageCoveringMostWordsGovernsWhenTheClientNamedNone(t *testing.T) {
	t.Parallel()

	contentWords := stopwords.ContentWordsOf(
		[]string{"how", "do", "reset", "my", "router"}, "",
	)

	if !slices.Equal(contentWords, []string{"reset", "router"}) {
		t.Fatalf("ContentWordsOf = %v, want reset router", contentWords)
	}
}

func TestWordsTwoLanguagesCoverAsOftenStay(t *testing.T) {
	t.Parallel()

	contentWords := stopwords.ContentWordsOf([]string{"mejores", "playas", "de"}, "")

	if !slices.Equal(contentWords, []string{"mejores", "playas", "de"}) {
		t.Fatalf("ContentWordsOf = %v, want mejores playas de", contentWords)
	}
}

func TestWordsNoLanguageCoversStay(t *testing.T) {
	t.Parallel()

	contentWords := stopwords.ContentWordsOf([]string{"recette", "crepes"}, "")

	if !slices.Equal(contentWords, []string{"recette", "crepes"}) {
		t.Fatalf("ContentWordsOf = %v, want recette crepes", contentWords)
	}
}

func TestWordsThatAreAllStopwordsStay(t *testing.T) {
	t.Parallel()

	contentWords := stopwords.ContentWordsOf([]string{"the", "who"}, "en")

	if !slices.Equal(contentWords, []string{"the", "who"}) {
		t.Fatalf("ContentWordsOf = %v, want the who", contentWords)
	}
}

func TestTheStopwordsOfEachListedLanguageGoAway(t *testing.T) {
	t.Parallel()

	contentWordsPerLanguage := map[string][]string{
		"en": stopwords.ContentWordsOf([]string{"the", "kernel"}, "en"),
		"de": stopwords.ContentWordsOf([]string{"der", "kernel"}, "de"),
		"fr": stopwords.ContentWordsOf([]string{"les", "kernel"}, "fr"),
		"es": stopwords.ContentWordsOf([]string{"los", "kernel"}, "es"),
		"it": stopwords.ContentWordsOf([]string{"gli", "kernel"}, "it"),
		"ru": stopwords.ContentWordsOf([]string{"для", "kernel"}, "ru"),
	}

	for language, contentWords := range contentWordsPerLanguage {
		if !slices.Equal(contentWords, []string{"kernel"}) {
			t.Fatalf("ContentWordsOf in %s = %v, want kernel", language, contentWords)
		}
	}
}
