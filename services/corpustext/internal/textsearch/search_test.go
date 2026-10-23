package textsearch_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

func TestSearchForKeepsTheQueryAndThePrimaryLanguageSubtag(t *testing.T) {
	search, err := textsearch.SearchFor("  stemmed words ", "DE-at", 20, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	want := textsearch.Search{Query: "stemmed words", Language: "de", Offset: 20, Limit: 10}
	if search != want {
		t.Errorf("search = %+v, want %+v", search, want)
	}
}

func TestSearchForCapsTheLimit(t *testing.T) {
	search, err := textsearch.SearchFor("words", "", 0, textsearch.MatchLimitMaximum+1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if search.Limit != textsearch.MatchLimitMaximum {
		t.Errorf("limit = %d, want %d", search.Limit, textsearch.MatchLimitMaximum)
	}
}

func TestSearchForRejectsAnUnanswerableSearch(t *testing.T) {
	cases := map[string]struct {
		query  string
		offset int
		limit  int
	}{
		"blank query":     {query: " ", offset: 0, limit: 10},
		"negative offset": {query: "words", offset: -1, limit: 10},
		"zero limit":      {query: "words", offset: 0, limit: 0},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := textsearch.SearchFor(c.query, "", c.offset, c.limit); err == nil {
				t.Error("expected an error")
			}
		})
	}
}
