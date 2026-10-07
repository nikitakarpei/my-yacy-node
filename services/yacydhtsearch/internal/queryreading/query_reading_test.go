package queryreading_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestQueryFromKeepsEachSpokenWordOnce(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom(`Berlin  "Weather" berlin +forecast`, yacymodel.Language{})

	if !slices.Equal(query.Words, []string{"berlin", "weather", "forecast"}) {
		t.Fatalf("Words = %v, want berlin weather forecast", query.Words)
	}
}

func TestQueryFromPutsAMinusWordUnderExclusions(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("berlin -rain", yacymodel.Language{})

	if !slices.Equal(query.Words, []string{"berlin"}) {
		t.Fatalf("Words = %v, want berlin", query.Words)
	}
	if !slices.Equal(query.Exclusions, []string{"rain"}) {
		t.Fatalf("Exclusions = %v, want rain", query.Exclusions)
	}
}

func TestQueryFromDropsAWordTooShortForAnIndex(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("berlin 1", yacymodel.Language{})

	if !slices.Equal(query.Words, []string{"berlin"}) {
		t.Fatalf("Words = %v, want berlin alone", query.Words)
	}
}

func TestQueryFromDropsAWordTooShortForAnIndexWhenItStandsAlone(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("1", yacymodel.Language{})

	if len(query.Words) != 0 {
		t.Fatalf("Words = %v, want no word", query.Words)
	}
}

func TestQueryFromDropsAnExclusionTooShortForAnIndex(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("berlin -a", yacymodel.Language{})

	if len(query.Exclusions) != 0 {
		t.Fatalf("Exclusions = %v, want no exclusion", query.Exclusions)
	}
}

func TestQueryFromReadsNothingOutOfPunctuationAlone(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom(`- "" +`, yacymodel.Language{})

	if len(query.Words) != 0 || len(query.Exclusions) != 0 {
		t.Fatalf("QueryFrom = %+v, want no words and no exclusions", query)
	}
}

func TestQueryFromLeavesTheStopwordsOfTheQueryLanguageOut(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("how do i reset my router", yacymodel.Language{})

	if !slices.Equal(query.Words, []string{"reset", "router"}) {
		t.Fatalf("Words = %v, want reset router", query.Words)
	}
}

func TestQueryFromKeepsAStopwordAmongTheExclusions(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("how to install debian -the", yacymodel.Language{})

	if !slices.Equal(query.Words, []string{"install", "debian"}) {
		t.Fatalf("Words = %v, want install debian", query.Words)
	}
	if !slices.Equal(query.Exclusions, []string{"the"}) {
		t.Fatalf("Exclusions = %v, want the", query.Exclusions)
	}
}

func TestAQueryThatAsksForACompoundWordIsSpelledApartFromOneThatDoesNot(t *testing.T) {
	t.Parallel()

	joined := queryreading.QueryFrom("state art", english(t))
	broken := queryreading.QueryFrom("state of the art", english(t))

	if joined.String() == broken.String() {
		t.Fatalf("both queries are spelled %q, want different spellings", joined.String())
	}
}

func TestQueriesThatDifferOnlyInTheirStopwordsAreSpelledAlike(t *testing.T) {
	t.Parallel()

	shorter := queryreading.QueryFrom("reset router", english(t))
	longer := queryreading.QueryFrom("how do i reset router", english(t))

	if shorter.String() != longer.String() {
		t.Fatalf("spellings %q and %q, want one spelling", shorter.String(), longer.String())
	}
}

func english(t *testing.T) yacymodel.Language {
	t.Helper()

	language, err := yacymodel.ParseLanguage("en")
	if err != nil {
		t.Fatalf("ParseLanguage(en): %v", err)
	}

	return language
}
