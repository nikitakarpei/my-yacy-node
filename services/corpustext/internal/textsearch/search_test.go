package textsearch_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

func TestSearchFromKeepsTheQueryAndThePrimaryLanguageSubtag(t *testing.T) {
	search, err := textsearch.SearchFrom("  stemmed words ", "DE-at", 20, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	want := textsearch.Search{Query: "stemmed words", Language: "de", Offset: 20, Limit: 10}
	if search != want {
		t.Errorf("search = %+v, want %+v", search, want)
	}
}

func TestSearchFromCapsTheLimit(t *testing.T) {
	search, err := textsearch.SearchFrom("words", "", 0, textsearch.MaximumMatchLimit+1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if search.Limit != textsearch.MaximumMatchLimit {
		t.Errorf("limit = %d, want %d", search.Limit, textsearch.MaximumMatchLimit)
	}
}

func TestSearchFromRejectsAnUnanswerableSearch(t *testing.T) {
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
			if _, err := textsearch.SearchFrom(c.query, "", c.offset, c.limit); err == nil {
				t.Error("expected an error")
			}
		})
	}
}
