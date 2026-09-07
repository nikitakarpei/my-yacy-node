package searchquery_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestQueryFromKeepsEachSpokenTermOnce(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom(`Berlin  "Weather" berlin +forecast`)

	if !slices.Equal(query.Terms, []string{"berlin", "weather", "forecast"}) {
		t.Fatalf("Terms = %v, want berlin weather forecast", query.Terms)
	}
}

func TestQueryFromPutsAMinusTermUnderExclusions(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin -rain")

	if !slices.Equal(query.Terms, []string{"berlin"}) {
		t.Fatalf("Terms = %v, want berlin", query.Terms)
	}
	if !slices.Equal(query.Exclusions, []string{"rain"}) {
		t.Fatalf("Exclusions = %v, want rain", query.Exclusions)
	}
}

func TestQueryFromDropsATermTooShortForAnIndex(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin 1")

	if !slices.Equal(query.Terms, []string{"berlin"}) {
		t.Fatalf("Terms = %v, want berlin alone", query.Terms)
	}
}

func TestQueryFromDropsATermTooShortForAnIndexWhenItStandsAlone(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("1")

	if len(query.Terms) != 0 {
		t.Fatalf("Terms = %v, want no term", query.Terms)
	}
}

func TestQueryFromDropsAnExclusionTooShortForAnIndex(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin -a")

	if len(query.Exclusions) != 0 {
		t.Fatalf("Exclusions = %v, want no exclusion", query.Exclusions)
	}
}

func TestQueryFromReadsNothingOutOfPunctuationAlone(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom(`- "" +`)

	if len(query.Terms) != 0 || len(query.Exclusions) != 0 {
		t.Fatalf("QueryFrom = %+v, want no terms and no exclusions", query)
	}
}

func TestTermHashesAddressTheWordsOnTheRing(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin -rain")

	if !slices.Equal(query.TermHashes(), []yacymodel.Hash{yacymodel.WordHash("berlin")}) {
		t.Fatalf("TermHashes = %v, want the hash of berlin", query.TermHashes())
	}
	if !slices.Equal(query.ExclusionHashes(), []yacymodel.Hash{yacymodel.WordHash("rain")}) {
		t.Fatalf("ExclusionHashes = %v, want the hash of rain", query.ExclusionHashes())
	}
}
